package wire

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

func TestTheAgentRelaysTheSessionsAgentButNeverItself(t *testing.T) {
	own := "/state/station/agent.sock"
	if relay := (machine{vars: map[string]string{"SSH_AUTH_SOCK": "/state/station//agent.sock"}}).sessionAgent(own); relay != nil {
		t.Fatal("the agent relays to its own socket")
	}
	if relay := (machine{vars: map[string]string{}}).sessionAgent(own); relay != nil {
		t.Fatal("the agent relays with no session agent")
	}
	if relay := (machine{vars: map[string]string{"SSH_AUTH_SOCK": "/private/tmp/session/Listeners"}}).sessionAgent(own); relay == nil {
		t.Fatal("the agent does not relay to the session's agent")
	}
}

func agentMachine(t *testing.T) machine {
	t.Helper()
	return machine{sys: local.New(nil, nil, io.Discard, io.Discard), vars: map[string]string{}, paths: setup.Paths{State: t.TempDir()}}
}

func TestAnAgentOnAHeldSocketNamesItsHolderAndBindsItOnceItsHolderGoes(t *testing.T) {
	m := agentMachine(t)
	holder, err := m.sys.Sockets.ListenUnix(AgentSocket(m.paths.State))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	_, err = m.openKey(ctx, &machinesfake.DeviceKey{})
	held := new(engine.AgentHeldError)
	if !errors.As(err, &held) {
		t.Fatalf("the agent served beside the socket's holder: %v", err)
	}
	if held.Holder.PID != os.Getpid() || held.Holder.Executable == "" || held.Holder.Executable != local.ExecutableOf(os.Getpid()) || held.Holder.Free != "kill "+strconv.Itoa(os.Getpid()) {
		t.Fatalf("the holder read %+v, want this process", held.Holder)
	}
	if _, statErr := m.sys.FS.Stat(DeviceKeyFile(m.paths.State)); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("the refused agent wrote the device key's line: %v", statErr)
	}
	go func() { _ = holder.Close() }()
	if freedErr := held.Freed(ctx); freedErr != nil {
		t.Fatalf("the holder's going was not told: %v", freedErr)
	}
	serve, err := m.openKey(ctx, &machinesfake.DeviceKey{})
	if err != nil {
		t.Fatalf("the agent did not bind the socket its holder let go: %v", err)
	}
	done, stop := context.WithCancel(ctx)
	stop()
	if serveErr := serve(done); serveErr != nil {
		t.Fatalf("the agent served with an error after its ctx ended: %v", serveErr)
	}
}

func TestAWaitOnAHeldSocketEndsWithTheEngine(t *testing.T) {
	m := agentMachine(t)
	holder, err := m.sys.Sockets.ListenUnix(AgentSocket(m.paths.State))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = holder.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	_, err = m.openKey(ctx, &machinesfake.DeviceKey{})
	held := new(engine.AgentHeldError)
	if !errors.As(err, &held) {
		t.Fatalf("the agent served beside the socket's holder: %v", err)
	}
	cancel()
	if freedErr := held.Freed(ctx); freedErr == nil {
		t.Fatal("a wait the engine's end cut read as the holder gone")
	}
}

func TestAnAgentTakesOverASocketNobodyAnswers(t *testing.T) {
	m := agentMachine(t)
	if err := m.sys.FS.WriteFile(AgentSocket(m.paths.State), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	serve, err := m.openKey(ctx, &machinesfake.DeviceKey{})
	if err != nil {
		t.Fatalf("the agent refused a socket nobody answers: %v", err)
	}
	if serveErr := serve(ctx); serveErr != nil {
		t.Fatalf("the agent served with an error after its ctx ended: %v", serveErr)
	}
	line, err := m.sys.FS.ReadFile(DeviceKeyFile(m.paths.State))
	if err != nil || !strings.HasPrefix(string(line), "ecdsa-sha2-nistp256 ") {
		t.Fatalf("the device key's line: %q %v", line, err)
	}
}
