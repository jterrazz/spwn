package ssh

import (
	"bytes"
	"context"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"slices"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the comment the device key carries in an agent's list and in its
// public key file.
const (
	DeviceKeyComment = "station device key"
)

var (
	errHoldsNoOther = errors.New("the agent of Machines holds only this device's key")
	errNotALogin    = errors.New("the agent of Machines signs an ssh login and nothing else")
	errForwarded    = errors.New("the agent of Machines signs nothing for a forwarded connection")
)

// value: the extension OpenSSH sends to bind an agent connection to the
// session it signs for (PROTOCOL.agent, session-bind).
const sessionBind = "session-bind@openssh.com"

type Agent struct {
	// value: the agent ssh reaches through IdentityAgent: it signs with this
	// device's key, and relays any other key to the agent the session had.
	key contract.DeviceKey
	// value: the session's own agent, dialed once per connection; nil when
	// there is none.
	relay func(ctx context.Context) (local.Conn, error)
}

func NewAgent(key contract.DeviceKey, relay func(ctx context.Context) (local.Conn, error)) *Agent {
	return &Agent{key: key, relay: relay}
}

func (a *Agent) PublicLine(ctx context.Context) (string, error) {
	// invariant: the device key is asked for before the socket opens, so a
	// machine with nowhere to keep one is refused instead of serving no key.
	signer, err := a.signer(ctx)
	if err != nil {
		return "", err
	}
	line := ssh.MarshalAuthorizedKey(signer.PublicKey())
	return string(line[:len(line)-1]) + " " + DeviceKeyComment + "\n", nil
}

func (a *Agent) signer(ctx context.Context) (ssh.Signer, error) {
	pub, err := a.key.Public(ctx)
	if err != nil {
		return nil, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	return deviceSigner{ctx: ctx, key: a.key, pub: sshPub}, nil
}

func (a *Agent) Listen(ctx context.Context, l local.Listener) error {
	// invariant: each accepted connection is served on its own, and every one
	// ends when ctx does.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	defer wg.Wait()
	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()
	for {
		conn, err := l.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
			defer stop()
			defer func() { _ = conn.Close() }()
			_ = a.Serve(ctx, conn)
		}()
	}
}

func (a *Agent) Serve(ctx context.Context, conn io.ReadWriter) error {
	k := &keyring{a: a, ctx: ctx}
	defer k.close()
	err := agent.ServeAgent(k, conn)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

// invariant: the device key signs the message itself, since the Secure
// Enclave hashes what it signs; the signature is then worded as ssh's
// ecdsa-sha2-nistp256.
type deviceSigner struct {
	ctx context.Context
	key contract.DeviceKey
	pub ssh.PublicKey
}

func (s deviceSigner) PublicKey() ssh.PublicKey { return s.pub }

func (s deviceSigner) Sign(_ io.Reader, message []byte) (*ssh.Signature, error) {
	der, err := s.key.Sign(s.ctx, message)
	if err != nil {
		return nil, err
	}
	var rs struct{ R, S *big.Int }
	if rest, parseErr := asn1.Unmarshal(der, &rs); parseErr != nil || len(rest) > 0 {
		return nil, errors.New("the device key's signature is not ASN.1 ECDSA")
	}
	return &ssh.Signature{Format: s.pub.Type(), Blob: ssh.Marshal(rs)}, nil
}

type keyring struct {
	a    *Agent
	ctx  context.Context
	mu   sync.Mutex
	up   agent.ExtendedAgent
	conn local.Conn
	// value: the sessions this connection was bound to, by their ids; empty
	// when the client bound none.
	bound [][]byte
	// value: the connection was bound to a forwarded session: it signs
	// nothing more.
	forwarded bool
}

func (k *keyring) upstream() agent.ExtendedAgent {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.up != nil || k.a.relay == nil {
		return k.up
	}
	conn, err := k.a.relay(k.ctx)
	if err != nil {
		return nil
	}
	k.conn, k.up = conn, agent.NewClient(conn)
	return k.up
}

func (k *keyring) close() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.conn != nil {
		_ = k.conn.Close()
	}
}

func (k *keyring) List() ([]*agent.Key, error) {
	signer, err := k.a.signer(k.ctx)
	if err != nil {
		return nil, err
	}
	pub := signer.PublicKey()
	keys := []*agent.Key{{Format: pub.Type(), Blob: pub.Marshal(), Comment: DeviceKeyComment}}
	if up := k.upstream(); up != nil {
		if others, err := up.List(); err == nil {
			keys = append(keys, others...)
		}
	}
	return keys, nil
}

func (k *keyring) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return k.SignWithFlags(key, data, 0)
}

func (k *keyring) SignWithFlags(key ssh.PublicKey, data []byte, flags agent.SignatureFlags) (*ssh.Signature, error) {
	if err := k.mayLogin(key, data); err != nil {
		return nil, err
	}
	signer, err := k.a.signer(k.ctx)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(key.Marshal(), signer.PublicKey().Marshal()) {
		return signer.Sign(nil, data)
	}
	up := k.upstream()
	if up == nil {
		return nil, errHoldsNoOther
	}
	return up.SignWithFlags(key, data, flags)
}

// invariant: the agent signs an ssh login and nothing else: the data is
// the whole of RFC 4252 §7's signed request for this key, on a session
// this connection was bound to when it bound one, and a connection bound
// to a forwarded session signs nothing. So no file content, seal or
// SSHSIG blob is ever signed through agent.sock.
func (k *keyring) mayLogin(key ssh.PublicKey, data []byte) error {
	k.mu.Lock()
	forwarded, bound := k.forwarded, k.bound
	k.mu.Unlock()
	if forwarded {
		return errForwarded
	}
	session, ok := loginFor(key, data)
	if !ok {
		return errNotALogin
	}
	if len(bound) > 0 && !slices.ContainsFunc(bound, func(id []byte) bool { return bytes.Equal(id, session) }) {
		return errNotALogin
	}
	return nil
}

// value: the userauth request's byte, SSH_MSG_USERAUTH_REQUEST.
const msgUserAuthRequest = 50

func loginFor(key ssh.PublicKey, data []byte) (session []byte, ok bool) {
	r := wire{data}
	session, ok = r.string()
	if !ok || len(session) == 0 {
		return nil, false
	}
	if b, ok := r.byte(); !ok || b != msgUserAuthRequest {
		return nil, false
	}
	_, okUser := r.string()
	layer, okLayer := r.string()
	method, okMethod := r.string()
	signed, okSigned := r.byte()
	_, okAlgo := r.string()
	blob, okBlob := r.string()
	if !okUser || !okLayer || !okMethod || !okSigned || !okAlgo || !okBlob || len(r.rest) != 0 {
		return nil, false
	}
	if string(layer) != "ssh-connection" || string(method) != "publickey" || signed != 1 || !bytes.Equal(blob, key.Marshal()) {
		return nil, false
	}
	return session, true
}

type wire struct{ rest []byte }

func (w *wire) byte() (byte, bool) {
	if len(w.rest) < 1 {
		return 0, false
	}
	b := w.rest[0]
	w.rest = w.rest[1:]
	return b, true
}

func (w *wire) string() ([]byte, bool) {
	if len(w.rest) < 4 {
		return nil, false
	}
	n := int64(binary.BigEndian.Uint32(w.rest))
	if n > int64(len(w.rest)-4) {
		return nil, false
	}
	s := w.rest[4 : 4+n]
	w.rest = w.rest[4+n:]
	return s, true
}

func (*keyring) Add(agent.AddedKey) error   { return errHoldsNoOther }
func (*keyring) Remove(ssh.PublicKey) error { return errHoldsNoOther }
func (*keyring) RemoveAll() error           { return errHoldsNoOther }
func (*keyring) Lock([]byte) error          { return errHoldsNoOther }
func (*keyring) Unlock([]byte) error        { return errHoldsNoOther }

func (*keyring) Signers() ([]ssh.Signer, error) { return nil, errHoldsNoOther }

func (k *keyring) Extension(kind string, contents []byte) ([]byte, error) {
	// invariant: a session binding is kept only once the host key's
	// signature over the session id verifies; any other extension is
	// unsupported.
	if kind != sessionBind {
		return nil, agent.ErrExtensionUnsupported
	}
	var bind struct {
		HostKey    []byte
		SessionID  []byte
		Signature  []byte
		Forwarding bool
	}
	if err := ssh.Unmarshal(contents, &bind); err != nil {
		return nil, err
	}
	server, err := ssh.ParsePublicKey(bind.HostKey)
	if err != nil {
		return nil, err
	}
	var sig ssh.Signature
	if err := ssh.Unmarshal(bind.Signature, &sig); err != nil {
		return nil, err
	}
	if err := server.Verify(bind.SessionID, &sig); err != nil {
		return nil, err
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	k.bound = append(k.bound, bytes.Clone(bind.SessionID))
	k.forwarded = k.forwarded || bind.Forwarding
	return nil, nil
}
