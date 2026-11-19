package wire

import (
	"context"
	"errors"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/cli"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// invariant: a sign-in's lines run under /bin/sh on this machine, or inside
// a VM of this Mac over its engine's own connection; a line attached inside
// a VM needs this run to be that engine.
type signInMachines struct{ m machine }

var errNoEngine = errors.New("a VM is signed in by the engine of its own Mac")

func stoppedAs(err error) error {
	var f *fail.Error
	if errors.As(err, &f) && (f.Code == enginecalls.VMStopped || f.Code == enginecalls.GuestAbsent) {
		return cli.ErrStopped
	}
	return err
}

func (s signInMachines) local(name string) bool {
	return name == "" || name == s.m.profile.Ref.Name
}

func (s signInMachines) Mac() string { return s.m.profile.Ref.Name }

func (s signInMachines) Exec(ctx context.Context, name, line, stdin string) (cli.Ran, error) {
	if s.local(name) {
		out, err := s.m.sys.Run.Output(ctx, local.Cmd{Name: "/bin/sh", Args: []string{"-c", line}, Stdin: stdin})
		return cli.Ran{Code: out.ExitCode, Out: out.Stdout + out.Stderr}, err
	}
	ran, err := s.m.macEngine().GuestExec(ctx, name, line, stdin)
	return cli.Ran{Code: ran.Code, Out: ran.Stdout + ran.Stderr}, stoppedAs(err)
}

func (s signInMachines) Attach(ctx context.Context, name, line string) (cli.Session, error) {
	if !s.local(name) {
		if s.m.far.engine == nil {
			return nil, errNoEngine
		}
		session, err := s.m.far.engine.GuestAttach(ctx, name, line)
		return session, stoppedAs(err)
	}
	spawn, ok := s.m.sys.Run.(local.Spawner)
	if !ok {
		return nil, errors.ErrUnsupported
	}
	p, err := spawn.Spawn(ctx, local.Cmd{Name: "/bin/sh", Args: []string{"-c", line}})
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		p.Kill()
	}()
	return spawned{p}, nil
}

type spawned struct{ p local.Process }

func (s spawned) Read(b []byte) (int, error)  { return s.p.Stdout().Read(b) }
func (s spawned) Write(b []byte) (int, error) { return s.p.Stdin().Write(b) }
func (s spawned) Wait() (int, error) {
	code, _, err := s.p.Wait()
	return code, err
}

func (s signInMachines) Where(ctx context.Context, name string) (accounts.Where, bool) {
	if s.local(name) {
		return accounts.Where{Kind: s.m.profile.Kind}, true
	}
	found, err := s.m.machines(false).Lookup(ctx, name)
	if err != nil {
		if mac, held := s.m.holder(ctx, name); held == nil && mac != "" {
			return accounts.Where{Kind: coremachine.Workspace, FarMac: s.m.label(mac)}, true
		}
		return accounts.Where{}, false
	}
	if found.Kind == catalog.Mac {
		return accounts.Where{}, false
	}
	running, err := s.m.vmRunning(ctx, name)
	return accounts.Where{Kind: found.Lifetime.Kind(), Stopped: err == nil && !running}, true
}

func (s signInMachines) Open(ctx context.Context, page string) error {
	out, err := s.m.sys.Run.Output(ctx, local.Cmd{Name: "/usr/bin/open", Args: []string{page}})
	if err == nil && out.ExitCode != 0 {
		return errors.New("the page did not open")
	}
	return err
}

func (s signInMachines) Now() time.Time { return s.m.sys.Clock.Now() }
