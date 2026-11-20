package ssh

import (
	"bytes"
	"context"
	"encoding/asn1"
	"errors"
	"io"
	"math/big"
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

var errHoldsNoOther = errors.New("the agent of Machines holds only this device's key")

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

func (*keyring) Add(agent.AddedKey) error   { return errHoldsNoOther }
func (*keyring) Remove(ssh.PublicKey) error { return errHoldsNoOther }
func (*keyring) RemoveAll() error           { return errHoldsNoOther }
func (*keyring) Lock([]byte) error          { return errHoldsNoOther }
func (*keyring) Unlock([]byte) error        { return errHoldsNoOther }

func (*keyring) Signers() ([]ssh.Signer, error) { return nil, errHoldsNoOther }

func (*keyring) Extension(string, []byte) ([]byte, error) {
	return nil, agent.ErrExtensionUnsupported
}
