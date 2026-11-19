package wire

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"path"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/ssh"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: the Secure Enclave's tag for this Mac's one device key.
const deviceKeyLabel = "com.jterrazz.station.device-key"

const signReason = "sign in over ssh as this Mac"

func AgentSocket(state string) string { return path.Join(state, "agent.sock") }

func DeviceKeyFile(state string) string { return path.Join(state, "device-key.pub") }

type enclaveKey struct{}

func enclaveRefusal(err error) error {
	var none *darwin.NoEnclaveError
	if errors.As(err, &none) {
		return &contract.NoDeviceKeyError{Why: none.Why}
	}
	return err
}

func (enclaveKey) Public(context.Context) (*ecdsa.PublicKey, error) {
	point, err := darwin.EnclavePublic(deviceKeyLabel)
	if err != nil {
		return nil, enclaveRefusal(err)
	}
	return ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
}

func (enclaveKey) Sign(_ context.Context, digest []byte) ([]byte, error) {
	sig, err := darwin.EnclaveSign(deviceKeyLabel, signReason, keychain.PresenceBudget, digest)
	return sig, enclaveRefusal(err)
}

func (m machine) deviceKey() contract.DeviceKey {
	if m.world != nil {
		return m.world.deviceKey()
	}
	return enclaveKey{}
}

// invariant: the session's own agent is relayed only when it is not this
// one, so the agent never dials itself.
func (m machine) sessionAgent(own string) func(ctx context.Context) (local.Conn, error) {
	up := m.vars["SSH_AUTH_SOCK"]
	if up == "" || path.Clean(up) == path.Clean(own) {
		return nil
	}
	return func(ctx context.Context) (local.Conn, error) {
		return m.sys.Sockets.DialUnix(ctx, up, 5*time.Second)
	}
}

func (m machine) serveAgent(ctx context.Context) error {
	return m.serveKey(ctx, m.deviceKey())
}

// invariant: an agent that answers on the socket makes a second one refuse;
// a socket file nobody answers is removed and made again.
func (m machine) serveKey(ctx context.Context, key contract.DeviceKey) error {
	sock := AgentSocket(m.paths.State)
	a := ssh.NewAgent(key, m.sessionAgent(sock))
	line, err := a.PublicLine(ctx)
	if err != nil {
		return err
	}
	if c, dialErr := m.sys.Sockets.DialUnix(ctx, sock, time.Second); dialErr == nil {
		_ = c.Close()
		return errAgentRuns
	}
	if mkErr := m.sys.FS.MkdirAll(m.paths.State, 0o700); mkErr != nil {
		return mkErr
	}
	if writeErr := m.sys.FS.WriteFile(DeviceKeyFile(m.paths.State), []byte(line), 0o644); writeErr != nil {
		return writeErr
	}
	_ = m.sys.FS.Remove(sock)
	l, err := m.sys.Sockets.ListenUnix(sock)
	if err != nil {
		return err
	}
	return a.Listen(ctx, l)
}

var errAgentRuns = errors.New("station's agent already serves this Mac")
