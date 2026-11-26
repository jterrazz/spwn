package door

import (
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"io"
	"math/big"
	"slices"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Admit func(ctx context.Context, key string) (device string, err error)

type Key interface {
	Public(ctx context.Context) (*ecdsa.PublicKey, error)
	Sign(ctx context.Context, message []byte) ([]byte, error)
}

type Verdict struct {
	Device      string
	Fingerprint string
	Refused     string
}

func (v Verdict) Line() string {
	if v.Refused == "" {
		return "door: " + v.Device + " let in, key " + v.Fingerprint
	}
	if v.Fingerprint == "" {
		return "door: refused: " + v.Refused
	}
	return "door: refused " + v.Fingerprint + ": " + v.Refused
}

type Door struct {
	own   Key
	pub   ssh.PublicKey
	admit Admit
	said  func(Verdict)
}

const (
	// value: the protocol line the door answers with, naming no version a
	// scanner could match.
	version = "SSH-2.0-spwn"
	// value: the keys a client may try on one connection.
	maxAuthTries = 2
	deviceField  = "device"
	keyField     = "fingerprint"
	lineField    = "key"
)

// value: the key types a device authenticates with, the book's own.
func authAlgorithms() []string { return []string{ssh.KeyAlgoECDSA256, ssh.KeyAlgoED25519} }

// invariant: the door speaks only these key exchanges, ciphers and MACs,
// its host key is one ecdsa-sha2-nistp256 key and its devices' keys are
// authAlgorithms: a library default never widens the door, since every list
// is set here and none is left empty for x/crypto to fill.
func algorithms() ssh.Config {
	return ssh.Config{
		KeyExchanges: []string{ssh.KeyExchangeMLKEM768X25519, ssh.KeyExchangeCurve25519},
		Ciphers:      []string{ssh.CipherChaCha20Poly1305, ssh.CipherAES256GCM, ssh.CipherAES128GCM},
		MACs:         []string{ssh.HMACSHA256ETM, ssh.HMACSHA512ETM},
	}
}

// value: the one key type the door's own key and a knocking device sign
// with, the Secure Enclave's.
func signingAlgorithms() []string { return []string{ssh.KeyAlgoECDSA256} }

var (
	errNoKey    = errors.New("no key was proved")
	errNoDevice = errors.New("the book names no device for this key")
	errBroke    = errors.New("the door broke on this connection")
)

func New(ctx context.Context, own Key, admit Admit, said func(Verdict)) (*Door, error) {
	pub, err := publicOf(ctx, own)
	if err != nil {
		return nil, err
	}
	return &Door{own: own, pub: pub, admit: admit, said: said}, nil
}

type refusal struct {
	fingerprint string
	reason      error
}

func (r *refusal) Error() string { return r.reason.Error() }

// invariant: a key is judged each time the door is about to trust it: when
// it is offered, once its signature is verified, and once the handshake is
// over, right before the connection is the device's. Each judgement asks
// the book again and keeps nothing: what it answers is that key's own
// Permissions or its own refusal, so a device forgotten at any point of its
// handshake is let in as nobody, and a client that asks about a book key it
// cannot sign for, then signs with another key, is that other key's device
// or nobody.
func (d *Door) judge(ctx context.Context, line, fingerprint string) (*ssh.Permissions, error) {
	device, err := d.admit(ctx, line)
	if err == nil && device == "" {
		err = errNoDevice
	}
	if err != nil {
		return nil, &refusal{fingerprint: fingerprint, reason: err}
	}
	return &ssh.Permissions{Extensions: map[string]string{deviceField: device, keyField: fingerprint, lineField: line}}, nil
}

func (d *Door) judgeKey(ctx context.Context, key ssh.PublicKey) (*ssh.Permissions, error) {
	return d.judge(ctx, strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))), ssh.FingerprintSHA256(key))
}

func (d *Door) terms(ctx context.Context) *ssh.ServerConfig {
	t := &ssh.ServerConfig{
		Config:                  algorithms(),
		MaxAuthTries:            maxAuthTries,
		ServerVersion:           version,
		PublicKeyAuthAlgorithms: authAlgorithms(),
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			return d.judgeKey(ctx, key)
		},
		VerifiedPublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey, _ *ssh.Permissions, _ string) (*ssh.Permissions, error) {
			return d.judgeKey(ctx, key)
		},
	}
	// invariant: the door's own key signs a connection's exchange under that
	// connection's context, never under the one the door was made in, so a
	// door outlives its constructor's context.
	t.AddHostKey(keySigner{ctx: ctx, key: d.own, pub: d.pub})
	return t
}

func refused(err error) Verdict {
	var auth *ssh.ServerAuthError
	if errors.As(err, &auth) {
		for i := len(auth.Errors) - 1; i >= 0; i-- {
			var r *refusal
			if errors.As(auth.Errors[i], &r) {
				return Verdict{Fingerprint: r.fingerprint, Refused: r.Error()}
			}
		}
	}
	var r *refusal
	if errors.As(err, &r) {
		return Verdict{Fingerprint: r.fingerprint, Refused: r.Error()}
	}
	return Verdict{Refused: errNoKey.Error()}
}

func (d *Door) Serve(ctx context.Context, c local.NetConn) (err error) {
	// invariant: a panic while the door judges or answers a connection ends
	// that connection alone, with the door's line; the engine that serves
	// its other connections never sees it.
	defer func() {
		if broke := recover(); broke != nil {
			_ = c.Close()
			d.said(Verdict{Refused: fmt.Sprint(errBroke.Error()+": ", broke)})
			err = errBroke
		}
	}()
	return d.serve(ctx, c)
}

func (d *Door) serve(ctx context.Context, c local.NetConn) error {
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	conn, channels, requests, err := ssh.NewServerConn(c, d.terms(ctx))
	if err != nil {
		d.said(refused(err))
		return err
	}
	defer func() { _ = conn.Close() }()
	last, err := d.judge(ctx, conn.Permissions.Extensions[lineField], conn.Permissions.Extensions[keyField])
	if err != nil {
		d.said(refused(err))
		return err
	}
	d.said(Verdict{Device: last.Extensions[deviceField], Fingerprint: last.Extensions[keyField]})
	go ssh.DiscardRequests(requests)
	for ch := range channels {
		// invariant: no channel is served yet: every type is refused, and
		// a session (a shell, an exec, a subsystem) as prohibited.
		reason := ssh.UnknownChannelType
		if ch.ChannelType() == "session" {
			reason = ssh.Prohibited
		}
		_ = ch.Reject(reason, "this door opens no "+ch.ChannelType())
	}
	return conn.Wait()
}

func Knock(ctx context.Context, c local.NetConn, key Key, pinned *ecdsa.PublicKey) error {
	signer, err := signerOf(ctx, key)
	if err != nil {
		return err
	}
	pinnedKey, err := ssh.NewPublicKey(pinned)
	if err != nil {
		return err
	}
	cfg := &ssh.ClientConfig{
		Config:            algorithms(),
		User:              "spwn",
		Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback:   ssh.FixedHostKey(pinnedKey),
		HostKeyAlgorithms: signingAlgorithms(),
	}
	conn, channels, requests, err := ssh.NewClientConn(c, "door", cfg)
	if err != nil {
		return err
	}
	go ssh.DiscardRequests(requests)
	go func() {
		for ch := range channels {
			_ = ch.Reject(ssh.Prohibited, "a knock opens nothing")
		}
	}()
	return conn.Close()
}

func signerOf(ctx context.Context, k Key) (ssh.Signer, error) {
	pub, err := publicOf(ctx, k)
	if err != nil {
		return nil, err
	}
	return keySigner{ctx: ctx, key: k, pub: pub}, nil
}

func publicOf(ctx context.Context, k Key) (ssh.PublicKey, error) {
	pub, err := k.Public(ctx)
	if err != nil {
		return nil, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(signingAlgorithms(), sshPub.Type()) {
		return nil, errors.New("the key is " + sshPub.Type() + ", not a P-256 key")
	}
	return sshPub, nil
}

// invariant: the key signs the message itself, since the Secure Enclave
// hashes what it signs; the signature is then worded as ssh's
// ecdsa-sha2-nistp256.
type keySigner struct {
	ctx context.Context
	key Key
	pub ssh.PublicKey
}

func (s keySigner) PublicKey() ssh.PublicKey { return s.pub }

func (s keySigner) Sign(_ io.Reader, message []byte) (*ssh.Signature, error) {
	der, err := s.key.Sign(s.ctx, message)
	if err != nil {
		return nil, err
	}
	var rs struct{ R, S *big.Int }
	if rest, parseErr := asn1.Unmarshal(der, &rs); parseErr != nil || len(rest) > 0 {
		return nil, errors.New("the key's signature is not ASN.1 ECDSA")
	}
	return &ssh.Signature{Format: s.pub.Type(), Blob: ssh.Marshal(rs)}, nil
}
