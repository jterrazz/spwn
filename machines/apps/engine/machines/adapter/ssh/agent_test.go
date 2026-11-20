package ssh

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract/contracttest"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: a device key in memory; absent is a machine with nowhere to keep one.
type deviceKey struct {
	absent bool
	key    *ecdsa.PrivateKey
	signed int
}

func (k *deviceKey) private() (*ecdsa.PrivateKey, error) {
	if k.absent {
		return nil, &contract.NoDeviceKeyError{Why: "none"}
	}
	if k.key == nil {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		k.key = key
	}
	return k.key, nil
}

func (k *deviceKey) Public(context.Context) (*ecdsa.PublicKey, error) {
	key, err := k.private()
	if err != nil {
		return nil, err
	}
	return &key.PublicKey, nil
}

func (k *deviceKey) Sign(_ context.Context, message []byte) ([]byte, error) {
	key, err := k.private()
	if err != nil {
		return nil, err
	}
	k.signed++
	digest := sha256.Sum256(message)
	return ecdsa.SignASN1(rand.Reader, key, digest[:])
}

func TestTheInMemoryKeyKeepsTheDeviceKeyContract(t *testing.T) {
	contracttest.DeviceKey(t, func(*testing.T) contract.DeviceKey { return &deviceKey{} })
}

type duplex struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (d duplex) Read(b []byte) (int, error)  { return d.r.Read(b) }
func (d duplex) Write(b []byte) (int, error) { return d.w.Write(b) }

func (d duplex) Close() error {
	_ = d.r.Close()
	return d.w.Close()
}

func pipes() (client, server local.Conn) {
	ar, bw := io.Pipe()
	br, aw := io.Pipe()
	return duplex{ar, aw}, duplex{br, bw}
}

func served(t *testing.T, a *Agent) agent.ExtendedAgent {
	t.Helper()
	client, server := pipes()
	done := make(chan error, 1)
	go func() { done <- a.Serve(t.Context(), server) }()
	t.Cleanup(func() {
		_ = client.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve = %v; want nil once its connection closed", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("the agent did not end once its connection closed")
		}
	})
	return agent.NewClient(client)
}

func upstream(t *testing.T) (func(context.Context) (local.Conn, error), ssh.PublicKey) {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ring := agent.NewKeyring()
	if addErr := ring.Add(agent.AddedKey{PrivateKey: private, Comment: "id_ed25519"}); addErr != nil {
		t.Fatal(addErr)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return func(context.Context) (local.Conn, error) {
		mine, theirs := pipes()
		go func() { _ = agent.ServeAgent(ring, theirs) }()
		return mine, nil
	}, signer.PublicKey()
}

func TestTheAgentSignsWithTheDeviceKey(t *testing.T) {
	key := &deviceKey{}
	c := served(t, NewAgent(key, nil))
	keys, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Comment != DeviceKeyComment || keys[0].Format != ssh.KeyAlgoECDSA256 {
		t.Fatalf("List = %v; want the device key alone, %s", keys, ssh.KeyAlgoECDSA256)
	}
	data := []byte("session")
	sig, err := c.Sign(keys[0], data)
	if err != nil {
		t.Fatal(err)
	}
	if err := keys[0].Verify(data, sig); err != nil {
		t.Fatalf("the device key's signature does not verify: %v", err)
	}
	if key.signed != 1 {
		t.Fatalf("the device key signed %d times; want once", key.signed)
	}
}

func TestTheAgentRelaysTheSessionsOtherKeys(t *testing.T) {
	relay, theirs := upstream(t)
	key := &deviceKey{}
	c := served(t, NewAgent(key, relay))
	keys, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0].Comment != DeviceKeyComment || keys[1].Comment != "id_ed25519" {
		t.Fatalf("List = %v; want the device key, then the session's", keys)
	}
	data := []byte("session")
	sig, err := c.Sign(theirs, data)
	if err != nil {
		t.Fatal(err)
	}
	if err := theirs.Verify(data, sig); err != nil {
		t.Fatalf("the relayed signature does not verify: %v", err)
	}
	if key.signed != 0 {
		t.Fatalf("the device key signed for another key %d times", key.signed)
	}
}

func TestTheAgentRefusesAKeyItDoesNotHold(t *testing.T) {
	_, other := upstream(t)
	c := served(t, NewAgent(&deviceKey{}, nil))
	if _, err := c.Sign(other, []byte("x")); err == nil {
		t.Fatal("signed with a key no agent holds")
	}
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Add(agent.AddedKey{PrivateKey: private}); err == nil {
		t.Fatal("the agent took a key: it holds only the device's")
	}
	if keys, err := c.List(); err != nil || len(keys) != 1 {
		t.Fatalf("List after an add = %v, %v; want the device key alone", keys, err)
	}
}

func TestThePublicLineIsTheDeviceKeysAuthorizedLine(t *testing.T) {
	a := NewAgent(&deviceKey{}, nil)
	line, err := a.PublicLine(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	pub, comment, _, rest, err := ssh.ParseAuthorizedKey([]byte(line))
	if err != nil {
		t.Fatal(err)
	}
	if pub.Type() != ssh.KeyAlgoECDSA256 || comment != DeviceKeyComment || len(rest) != 0 || !strings.HasSuffix(line, "\n") {
		t.Fatalf("PublicLine = %q", line)
	}
	var none *contract.NoDeviceKeyError
	if _, err := NewAgent(&deviceKey{absent: true}, nil).PublicLine(t.Context()); !errors.As(err, &none) {
		t.Fatalf("PublicLine on a machine with no device key = %v; want a NoDeviceKeyError", err)
	}
}

type listener struct {
	conns  chan local.Conn
	closed chan struct{}
}

func (l *listener) Accept() (local.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, errors.New("closed")
	}
}

func (l *listener) Close() error {
	select {
	case <-l.closed:
	default:
		close(l.closed)
	}
	return nil
}

func (*listener) Port() int { return 0 }

func TestTheAgentServesEachConnectionUntilItsContextEnds(t *testing.T) {
	l := &listener{conns: make(chan local.Conn), closed: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- NewAgent(&deviceKey{}, nil).Listen(ctx, l) }()
	for range 2 {
		client, server := pipes()
		select {
		case l.conns <- server:
		case <-time.After(5 * time.Second):
			t.Fatal("the agent did not accept a connection")
		}
		if keys, err := agent.NewClient(client).List(); err != nil || len(keys) != 1 {
			t.Fatalf("List = %v, %v", keys, err)
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Listen = %v; want nil once its context ends", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Listen did not end with its context")
	}
}
