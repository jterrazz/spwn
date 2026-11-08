package wire

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/page"

	"github.com/jterrazz/studio-station/src/accounts"
	accountscalls "github.com/jterrazz/studio-station/src/accounts/calls"
	"github.com/jterrazz/studio-station/src/activity"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	enginejobs "github.com/jterrazz/studio-station/src/engine/jobs"
	"github.com/jterrazz/studio-station/src/ports"
	portscalls "github.com/jterrazz/studio-station/src/ports/calls"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/src/background/contract"
	"github.com/jterrazz/studio-station/src/catalog"

	"github.com/jterrazz/studio-station/src/engine"
	enginelog "github.com/jterrazz/studio-station/src/engine/adapter/log"
	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/engine/process"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/adapter/dial"
	machinescalls "github.com/jterrazz/studio-station/src/machines/calls"
	machinescontract "github.com/jterrazz/studio-station/src/machines/contract"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
	portscontract "github.com/jterrazz/studio-station/src/ports/contract"
	"github.com/jterrazz/studio-station/src/remote"
	"github.com/jterrazz/studio-station/src/remote/api"
	"github.com/jterrazz/studio-station/src/setup"
	enginerun "github.com/jterrazz/studio-station/src/surface/cli/engine/run"
	"github.com/jterrazz/studio-station/src/surface/cli/kit"
	"github.com/jterrazz/studio-station/src/ui/token"
)

func (m machine) guestPort() int { return m.port(22, "STATION_GUEST_SSH_PORT") }

// invariant: inside the engine, the menu bar reads through this run's
// Router, so its fleet reaches the other Macs as the CLI's does; a recall
// rides the engine's own connections, never its socket: the lends are bound
// to the engine they serve.
func (m machine) engine(env kit.Env) *process.Engine {
	shares := &engineShares{}
	registry := m.registry(env)
	calls := env.Calls
	if calls == nil {
		calls = remote.NewRouter(registry, m.profile, remote.Reach{})
	}
	live := &feeds.Live{Calls: calls, Clock: m.sys.Clock, ThisMac: m.where.Machine, MacOf: m.macOf}
	menubar := feeds.Menubar{Calls: calls, Clock: m.sys.Clock, ThisMachine: m.where, Live: live, Report: func(v awake.View) *awake.Report {
		return &awake.Report{Now: m.sys.Clock.Now(), Zone: time.Local, Engine: awake.EngineRunning, Reading: v.Reading, Session: v.Session, LastEnd: v.LastEnd, Guards: v.Guards}
	}, EveryMac: func(ctx context.Context) (pageawake.AllMachines, error) {
		inside := env
		inside.Calls = calls
		return m.everyMacAwake(ctx, inside)
	}}
	guests := engineGuests{dialer: m.guestDialer(), vms: m.machines(false)}
	notifier := func(write func(subject, text string)) process.Notifier { return enginelog.New(write) }
	var tailnet portscontract.Tailnet = m.tailnetCLI().Tracked(engine.ChildrenDir(m.paths.State))
	var checks machinescontract.CheckRunner = dial.Checker{}
	if m.world != nil {
		notifier = func(func(subject, text string)) process.Notifier { return m.world.notifier() }
		tailnet = m.world.tailnet()
		checks = m.world.checks()
	}
	var d *process.Engine
	sample := &enginejobs.Sample{
		Calls: calls, Meter: m.meter(), Clock: m.sys.Clock,
		History: func() *activity.History { return d.History() },
		Watched: func(r coremachine.Ref) bool {
			return d.Watched(r.Name) || (r.On == "" && d.Watched(engine.ThisMac))
		},
		Tell: func(machine, finding, text string) { d.Tell(machine, finding, text) },
		Seen: func() []enginejobs.Seen {
			running := d.Running()
			seen := make([]enginejobs.Seen, 0, 1+len(running))
			seen = append(seen, enginejobs.Seen{Ref: coremachine.Ref{Name: m.profile.Ref.Name}, Kind: coremachine.Mac})
			for _, r := range running {
				seen = append(seen, enginejobs.Seen{Ref: coremachine.Ref{Name: r.Name, On: m.profile.Ref.Name}, Kind: r.Lifetime.Kind(), PID: r.PID, Guest: r.Guest})
			}
			return seen
		},
	}
	d = process.New(engineMac{m.sys}, engineHypervisor{m.hypervisor()}, guests, m.machines(false), tailnet, process.Stores{Ports: m.stores.ports(), Reservations: m.stores.reservations}, process.Options{
		Remember: func(vm string, port int, stopped bool) error {
			return portscalls.Remember(m.stores.ports(), vm, port, stopped)
		},
		State: m.paths.State, ThisMachine: m.kind, Version: m.engineVersion(), Page: page.Revision, StartedAt: m.engineStarted(), PID: os.Getpid(), GuestPort: m.guestPort(),
		Power: m.powerPort, KeepAwakePage: m.keepAwakePage, Executable: m.sys.Env.Executable(),
		NeverSleeps:   func() bool { on, _ := m.stores.neverSleeps().Get(); return on },
		AccountShares: shares, Peer: m.engineAPI(),
		Feeds: map[string]engine.Feed{engine.PageMenubar: menubar.Feed(), engine.PageSwitcher: menubar.Switcher(), engine.PageMachine: menubar.Machine(),
			engine.PagePorts: menubar.Ports(), engine.PageAwake: menubar.Awake(), engine.PageActivity: menubar.Activity(), engine.PageAccounts: menubar.Accounts(), engine.PageNeeds: menubar.Needs(), engine.PageFind: menubar.Find()},
		Sample:   live.Sample,
		Notifier: notifier, Checks: checks, Join: func(parts ...string) string { return strings.Join(parts, token.Sep) },
		Listening: m.listeners("/").Read, Declarations: m.stores.declarations.List,
		Jobs:     []process.Tick{sample.Run},
		TellBack: m.machines(false).TellBack, Answers: m.answers, ToldBack: m.machines(false).ToldBack, Label: m.label,
		AgentsHeld: func(ctx context.Context) (agents.Held, error) {
			inside := env
			inside.Calls = calls
			return agentsEverywhere{env: inside}.heldEverywhere(ctx)
		},
	})
	shares.s = m.secretsVia(d, func(_ context.Context, vm string) (bool, error) {
		for _, v := range d.VMs() {
			if v.Name == vm {
				return v.State == engine.StateRunning, nil
			}
		}
		return false, nil
	})
	d.ServeWith(m.engineServer(d, registry, calls))
	return d
}

// value: what the engine of a world Mac says of itself, the version it runs
// and when it started; outside a world, this binary's, started now.
func (m machine) engineVersion() string {
	if m.world != nil && m.world.engineVersion() != "" {
		return m.world.engineVersion()
	}
	return Version()
}

func (m machine) stationVersion() string {
	if m.world != nil && m.world.stationVersion() != "" {
		return m.world.stationVersion()
	}
	return Version()
}

func (m machine) engineStarted() time.Time {
	if m.world != nil {
		return m.world.engineStarted()
	}
	return time.Time{}
}

func engineRunCommand(env kit.Env, m machine) *cobra.Command {
	return enginerun.Command(enginerun.Deps{Engine: m.engine(env)}, env)
}

// invariant: a v2 job runs `station engine` until setup moves it to the
// engine's label; it runs the engine, hidden, for one release.
func retiredEngineCommand(env kit.Env, m machine) *cobra.Command {
	cmd := engineRunCommand(env, m)
	cmd.Use = kit.RetiredEngineCommand
	return cmd
}

type engineShares struct{ s *accountscalls.Sharing }

func (l *engineShares) Held() (map[string][]accounts.Share, error) {
	return l.s.Shares(context.Background())
}

func (l *engineShares) Expire(ctx context.Context, vm string, names []string) error {
	return l.s.Expire(ctx, vm, names)
}

func (l *engineShares) Forget(ctx context.Context, vm string) error { return l.s.Forget(ctx, vm) }

type macEngine struct{ c engineAPI }

func (m machine) macEngine() macEngine { return macEngine{c: m.engineAPI()} }

func downAs(err, sentinel error) error {
	var down *enginecalls.DownError
	var gone *api.Down
	if errors.As(err, &down) || errors.As(err, &gone) {
		return sentinel
	}
	return err
}

func (d macEngine) Ping(ctx context.Context) (string, error) {
	p, err := d.c.Hello(ctx)
	return p.Version, downAs(err, model.ErrEngineDown)
}

func (d macEngine) VMs(ctx context.Context) (map[string]machinescalls.EngineVM, error) {
	views, err := d.c.VMs(ctx)
	if err != nil {
		return nil, downAs(err, model.ErrEngineDown)
	}
	out := make(map[string]machinescalls.EngineVM, len(views))
	for _, v := range views {
		out[v.Name] = machinescalls.EngineVM{
			State: v.State, Guest: v.Guest, IP: v.IP, Detail: v.Detail, SSH: v.SSH, Restarts: v.Restarts, Check: v.Check, BootAt: v.BootAt,
			Latency: v.Latency, Restart: v.Restart, Cause: v.Cause, LastLines: v.LastLines, ReadyAt: v.ReadyAt, Agent: v.Agent,
		}
	}
	return out, nil
}

func (d macEngine) GuestExec(ctx context.Context, vm, line, stdin string) (engine.GuestRan, error) {
	var ran engine.ExecOut
	if err := d.c.do(ctx, engine.CallGuestExec, engine.ExecIn{VM: vm, Line: line, Stdin: stdin}, &ran); err != nil {
		return engine.GuestRan{}, downAs(err, model.ErrEngineDown)
	}
	return engine.GuestRan(ran), nil
}

func (d macEngine) Do(ctx context.Context, op, vm string) error {
	return downAs(d.c.do(ctx, engine.CallVM(op), engine.VMIn{VM: vm}, nil), model.ErrEngineDown)
}

func (d macEngine) Reserve(ctx context.Context, vm string, port int, profile string) (model.Reserved, error) {
	var v ports.SharedPort
	if err := d.c.do(ctx, engine.CallReserve, engine.ReserveIn{VM: vm, GuestPort: port, Pinned: true, Profile: profile}, &v); err != nil {
		return model.Reserved{}, downAs(err, model.ErrEngineDown)
	}
	return model.Reserved{
		GuestPort: v.GuestPort, MacPort: v.Reserved, Profile: profile, URL: v.URLs.Public, PublicName: nameOf(v.URLs.Public),
		Off: localOnlyBecause(v.Detail),
	}, nil
}

func localOnlyBecause(detail string) string {
	switch {
	case strings.Contains(detail, catalog.SignedOutDetail):
		return catalog.SignedOutLocalOnly
	case strings.Contains(detail, catalog.ServeOffDetail):
		return catalog.ServeOffDetail
	}
	return ""
}

func nameOf(url string) string {
	_, rest, ok := strings.Cut(url, "://")
	if !ok {
		rest = url
	}
	name, _, _ := strings.Cut(rest, "/")
	name, _, _ = strings.Cut(name, ":")
	return name
}

type portsEngine struct{ c engineAPI }

func (d portsEngine) MacPorts(ctx context.Context) (ports.MacPorts, error) {
	var v ports.MacPorts
	err := d.c.do(ctx, engine.CallPorts, nil, &v)
	return v, downAs(err, ports.ErrEngineDown)
}

func (d portsEngine) Reserve(ctx context.Context, vm string, port int, pinned bool) (ports.SharedPort, error) {
	var v ports.SharedPort
	err := d.c.do(ctx, engine.CallReserve, engine.ReserveIn{VM: vm, GuestPort: port, Pinned: pinned}, &v)
	return v, downAs(err, ports.ErrEngineDown)
}

func (d portsEngine) Release(ctx context.Context, vm string, port int) (ports.Reservation, error) {
	var l ports.Reservation
	err := d.c.do(ctx, engine.CallRelease, engine.ReserveIn{VM: vm, GuestPort: port}, &l)
	return l, downAs(err, ports.ErrEngineDown)
}

type reservationReleaser struct{ m machine }

func (r reservationReleaser) ReleaseVM(ctx context.Context, vm string) error {
	// invariant: a VM leaves nothing of its ports on its Mac: its ports
	// declared public, its reserved ports and their shares.
	if err := r.forgetDeclarations(ctx, vm); err != nil {
		return err
	}
	store := r.m.stores.reservations
	reservations, err := store.List(ctx)
	if err != nil {
		return err
	}
	kept, gone := ports.Without(reservations, func(l ports.Reservation) bool { return l.VM != vm })
	if len(gone) == 0 {
		return nil
	}
	g, _ := r.m.stores.ports().Get()
	mode := ports.PolicyOf(g).HTTPS
	tn := r.m.tailnet()
	if ready, err := tn.Ready(ctx); err == nil && ready.On {
		for _, l := range gone {
			_ = tn.Unshare(ctx, ports.ShareOf(l, mode))
		}
	}
	return portscalls.SaveReservations(ctx, store, kept)
}

func (r reservationReleaser) forgetDeclarations(ctx context.Context, vm string) error {
	declared, err := r.m.stores.declarations.List(ctx)
	if err != nil {
		return err
	}
	for _, d := range declared {
		if d.Machine != vm {
			continue
		}
		if err := r.m.stores.declarations.Delete(ctx, d.Machine, d.Port); err != nil {
			return err
		}
	}
	return nil
}

type engineInstaller struct {
	m        machine
	machines *machinescalls.Machinist
}

func (i engineInstaller) env() map[string]string {
	env := map[string]string{"HOME": i.m.sys.Env.Home()}
	for _, name := range []string{"PATH", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME", "STATION_GUEST_SSH_PORT"} {
		if v := i.m.vars[name]; v != "" {
			env[name] = v
		}
	}
	return env
}

// invariant: setup installs ONE engine per system, after migrating the round-3
// per-VM agents; it restarts the engine only when its plist changed or it runs
// another version, and the VMs survive that restart.
func (i engineInstaller) Install(ctx context.Context) (setup.Step, error) {
	const step = "engine"
	if hv := i.m.hypervisor(); !hv.Installed() {
		return setup.Step{Name: step, State: catalog.StateAbsent, Detail: "not needed until " + hv.Name() + " is installed"}, nil
	}
	jobs := i.m.jobs(0)
	if !jobs.Present() {
		return setup.Step{Name: step, State: catalog.StateAbsent, Detail: jobs.Names().Manager + " is not here to run it"}, nil
	}
	if _, err := i.machines.MigrateAgents(ctx); err != nil {
		return setup.Step{Name: step, State: catalog.StateBroken, Detail: err.Error()}, &setup.StepFailedError{Step: step, Cause: err}
	}
	moved, err := i.retire(ctx, jobs)
	if err != nil {
		return setup.Step{Name: step, State: catalog.StateBroken, Detail: err.Error()}, &setup.StepFailedError{Step: step, Cause: err}
	}
	s, err := i.install(ctx, jobs)
	if moved {
		s.Detail, s.Changed = s.Detail+", its job moved to "+catalog.EngineLabel, true
	}
	return s, err
}

// invariant: the job station v2 named com.jterrazz.station.daemon is
// removed once, before the engine's own is written: one engine per Mac.
func (i engineInstaller) retire(ctx context.Context, jobs contract.Jobs) (bool, error) {
	ref := contract.Ref{Domain: contract.Domain{UID: i.m.sys.Env.Euid()}, Label: catalog.RetiredEngineLabel, File: catalog.RetiredEnginePlist(i.m.sys.Env.Home())}
	if _, err := i.m.sys.FS.Stat(ref.File); err != nil && !jobs.Read(ctx, ref).Loaded {
		return false, nil
	}
	return true, jobs.Remove(ctx, ref)
}

func (i engineInstaller) install(ctx context.Context, jobs contract.Jobs) (setup.Step, error) {
	const step = "engine"
	job := i.m.engineJobRef()
	changed, err := jobs.Install(ctx, contract.JobSpec{
		Label: job.Label, Program: []string{i.m.sys.Env.Executable(), "engine", "run"}, Env: i.env(),
		AtLogin: true, KeepAlive: true, Log: filepath.Join(i.m.paths.State, "logs", "engine.log"), File: job.File,
	})
	if err != nil {
		return setup.Step{Name: step, State: catalog.StateBroken, Detail: err.Error()}, &setup.StepFailedError{Step: step, Cause: err}
	}
	version := Version()
	if !jobs.Read(ctx, job).Loaded {
		for _, run := range []func(context.Context, contract.Ref) error{jobs.Load, jobs.Start} {
			if runErr := run(ctx, job); runErr != nil {
				return setup.Step{Name: step, State: catalog.StateBroken, Detail: runErr.Error()}, &setup.StepFailedError{Step: step, Cause: runErr}
			}
		}
		return setup.Step{Name: step, State: catalog.StateOK, Detail: "running " + version, Changed: true}, nil
	}
	running, err := i.m.engineAPI().Hello(ctx)
	if changed || (err == nil && running.Version != version) {
		if kickErr := jobs.Restart(ctx, job); kickErr != nil {
			return setup.Step{Name: step, State: catalog.StateBroken, Detail: kickErr.Error()}, &setup.StepFailedError{Step: step, Cause: kickErr}
		}
		detail := "restarted"
		if err == nil && running.Version != version {
			detail += " (" + running.Version + " to " + version + ")"
		}
		return setup.Step{Name: step, State: catalog.StateOK, Detail: detail, Changed: true}, nil
	}
	return setup.Step{Name: step, State: catalog.StateOK, Detail: "running " + version}, nil
}
