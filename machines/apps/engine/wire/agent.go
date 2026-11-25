package wire

import (
	"context"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/background/adapter/launchd"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/macos"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/ssh"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

func AgentSocket(state string) string { return path.Join(state, "agent.sock") }

func DeviceKeyFile(state string) string { return path.Join(state, "device-key.pub") }

// invariant: a process holds one device key, made at its first ask with
// the access Never sleeps then says, so it serves, seals and lists with the
// key it started with for its whole life, and reads its public half from
// the Enclave once.
type heldKey struct {
	once sync.Once
	key  contract.DeviceKey
}

func (m machine) deviceKey() contract.DeviceKey {
	if m.world != nil {
		return m.world.deviceKey()
	}
	if m.key == nil {
		return m.newDeviceKey()
	}
	m.key.once.Do(func() { m.key.key = m.newDeviceKey() })
	return m.key.key
}

func (m machine) newDeviceKey() contract.DeviceKey {
	if key := specDeviceKey(m.vars); key != nil {
		return key
	}
	return macos.NewDeviceKey(m.sys.FS, m.paths.State, keychain.PresenceBudget, m.keyAccess())
}

// invariant: a Mac run alone is one whose Never sleeps is applied, the
// owner's word that it hosts servers with nobody at it; its device key then
// signs with nobody present. A preference that cannot be read keeps the key
// that asks for a present owner.
func (m machine) keyAccess() darwin.Access {
	if alone, err := m.stores.neverSleeps().Get(); err == nil && alone {
		return darwin.AccessUnattended
	}
	return darwin.AccessPresent
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

func (m machine) agentOpener() func(context.Context) (func(context.Context) error, error) {
	if m.world != nil {
		return m.world.agent()
	}
	return m.openAgent
}

func (m machine) openAgent(ctx context.Context) (func(context.Context) error, error) {
	return m.openKey(ctx, m.deviceKey())
}

// invariant: a process that answers on the socket holds it: the engine is
// refused with who holds it, and the connection kept open tells when the
// holder lets it go. A socket file nobody answers is removed and made again.
func (m machine) openKey(ctx context.Context, key contract.DeviceKey) (func(context.Context) error, error) {
	sock := AgentSocket(m.paths.State)
	a := ssh.NewAgent(key, m.sessionAgent(sock))
	line, err := a.PublicLine(ctx)
	if err != nil {
		return nil, err
	}
	if c, dialErr := m.sys.Sockets.DialUnix(ctx, sock, time.Second); dialErr == nil {
		return nil, m.heldBy(c)
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

func (m machine) heldBy(c local.Conn) *engine.AgentHeldError {
	pid, exe := local.PeerOf(c)
	return &engine.AgentHeldError{
		Holder: model.Agent{PID: pid, Executable: exe, Free: agentFree(pid, m.jobOf(pid))},
		Freed: func(ctx context.Context) error {
			stop := context.AfterFunc(ctx, func() { _ = c.Close() })
			defer stop()
			buf := make([]byte, 512)
			for {
				if _, err := c.Read(buf); err != nil {
					break
				}
			}
			_ = c.Close()
			return ctx.Err()
		},
	}
}

func agentFree(pid int, label string) string {
	if label != "" {
		return launchd.Bootout(label)
	}
	return "kill " + strconv.Itoa(pid)
}

func (m machine) jobOf(pid int) string {
	if pid <= 0 || m.world != nil {
		return ""
	}
	listed, err := m.bareJobs(time.Second).List(context.Background())
	if err != nil {
		return ""
	}
	for _, j := range listed {
		if j.Running && j.PID == pid {
			return j.Label
		}
	}
	return ""
}
