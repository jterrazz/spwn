package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// value: the data an ssh client asks an agent to sign for a login of key
// on the session of that id (RFC 4252 §7).
func login(key ssh.PublicKey, session string) []byte {
	return ssh.Marshal(struct {
		Session   string
		Request   byte
		User      string
		Service   string
		Method    string
		Signed    bool
		Algorithm string
		Key       []byte
	}{session, msgUserAuthRequest, "jterrazz", "ssh-connection", "publickey", true, key.Type(), key.Marshal()})
}

// value: a session-bind request for the session of that id, signed by a
// host key made for it.
func binding(t *testing.T, session string, forwarding bool) []byte {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	host, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := host.Sign(rand.Reader, []byte(session))
	if err != nil {
		t.Fatal(err)
	}
	return ssh.Marshal(struct {
		HostKey    []byte
		SessionID  []byte
		Signature  []byte
		Forwarding bool
	}{host.PublicKey().Marshal(), []byte(session), ssh.Marshal(sig), forwarding})
}

func deviceOf(t *testing.T, c agent.ExtendedAgent) *agent.Key {
	t.Helper()
	keys, err := c.List()
	if err != nil || len(keys) == 0 {
		t.Fatalf("List = %v, %v", keys, err)
	}
	return keys[0]
}

func TestTheAgentSignsALoginAndNothingElse(t *testing.T) {
	key := &deviceKey{}
	c := served(t, NewAgent(key, nil))
	device := deviceOf(t, c)
	if _, err := c.Sign(device, login(device, "session-1")); err != nil {
		t.Fatalf("a login was refused: %v", err)
	}
	seal := []byte("spwn-device-book/v1\nmac-mini\nAAAA\nlaptop\nssh-ed25519 AAAA\npipe\n2026-11-26T10:00:00Z")
	sshsig := append([]byte("SSHSIG"), ssh.Marshal(struct{ Namespace, Reserved, Hash string }{"file", "", "sha512"})...)
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	otherKey, _ := ssh.NewSignerFromKey(other)
	trailing := append(login(device, "session-1"), 0)
	asLogin := login(device, "session-1")
	asLogin[len("session-1")+4] = 51
	refused := map[string][]byte{
		"a device book seal":           seal,
		"an SSHSIG blob":               sshsig,
		"random bytes":                 []byte{0x9f, 0x00, 0x13, 0x37, 0x42},
		"a login of another key":       login(otherKey.PublicKey(), "session-1"),
		"a login with a byte left":     trailing,
		"another message than a login": asLogin,
	}
	for name, data := range refused {
		t.Run(name, func(t *testing.T) {
			if _, err := c.Sign(device, data); err == nil {
				t.Fatal("signed data that is no login of this key")
			}
		})
	}
	if key.signed != 1 {
		t.Fatalf("the device key signed %d times; want the login alone", key.signed)
	}
}

func TestAConnectionBoundToAForwardedSessionSignsNothing(t *testing.T) {
	c := served(t, NewAgent(&deviceKey{}, nil))
	device := deviceOf(t, c)
	if _, err := c.Extension(sessionBind, binding(t, "session-1", true)); err != nil {
		t.Fatalf("the binding was refused: %v", err)
	}
	if _, err := c.Sign(device, login(device, "session-1")); err == nil {
		t.Fatal("a forwarded connection signed a login")
	}
}

func TestAConnectionBoundToItsOwnSessionSignsItsLoginsAsBefore(t *testing.T) {
	c := served(t, NewAgent(&deviceKey{}, nil))
	device := deviceOf(t, c)
	if _, err := c.Extension(sessionBind, binding(t, "session-1", false)); err != nil {
		t.Fatalf("the binding was refused: %v", err)
	}
	if _, err := c.Sign(device, login(device, "session-1")); err != nil {
		t.Fatalf("a login of the bound session was refused: %v", err)
	}
	if _, err := c.Sign(device, login(device, "session-2")); err == nil {
		t.Fatal("signed a login of a session the connection was never bound to")
	}
}

func TestABindingWhoseSignatureFailsIsRefused(t *testing.T) {
	c := served(t, NewAgent(&deviceKey{}, nil))
	forged := binding(t, "session-1", false)
	forged[len(forged)-2] ^= 0xff
	if _, err := c.Extension(sessionBind, forged); err == nil {
		t.Fatal("kept a binding whose host signature does not verify")
	}
	if _, err := c.Extension("another@openssh.com", nil); err == nil {
		t.Fatal("answered an extension it does not know")
	}
}
