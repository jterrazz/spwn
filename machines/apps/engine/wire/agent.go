package wire

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/macos"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/ssh"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func AgentSocket(state string) string { return path.Join(state, "agent.sock") }

func DeviceKeyFile(state string) string { return path.Join(state, "device-key.pub") }

func (m machine) deviceKey() contract.DeviceKey {
	if m.world != nil {
		return m.world.deviceKey()
	}
	if key := specDeviceKey(m.vars); key != nil {
		return key
	}
	return macos.NewDeviceKey(m.sys.FS, m.paths.State, keychain.PresenceBudget)
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

func (m machine) openAgent(ctx context.Context) (func(context.Context) error, error) {
	return m.openKey(ctx, m.deviceKey())
}

func (m machine) serveAgent(ctx context.Context) error {
	serve, err := m.openAgent(ctx)
	if err != nil {
		return err
	}
	return serve(ctx)
}

// invariant: an agent that answers on the socket makes a second one refuse;
// a socket file nobody answers is removed and made again.
func (m machine) openKey(ctx context.Context, key contract.DeviceKey) (func(context.Context) error, error) {
	sock := AgentSocket(m.paths.State)
	a := ssh.NewAgent(key, m.sessionAgent(sock))
	line, err := a.PublicLine(ctx)
	if err != nil {
		return nil, err
	}
	if c, dialErr := m.sys.Sockets.DialUnix(ctx, sock, time.Second); dialErr == nil {
		_ = c.Close()
		return nil, errAgentRuns
	}
	if mkErr := m.sys.FS.MkdirAll(m.paths.State, 0o700); mkErr != nil {
		return nil, mkErr
	}
	if writeErr := m.sys.FS.WriteFile(DeviceKeyFile(m.paths.State), []byte(line), 0o644); writeErr != nil {
		return nil, writeErr
	}
	_ = m.sys.FS.Remove(sock)
	l, err := m.sys.Sockets.ListenUnix(sock)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error { return a.Listen(ctx, l) }, nil
}

var errAgentRuns = errors.New("Machines' agent already serves this Mac")
