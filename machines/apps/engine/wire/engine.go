package wire

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/page"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/adapter/keychain"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	appscalls "github.com/jterrazz/studio-station/apps/engine/apps/calls"
	corejournal "github.com/jterrazz/studio-station/apps/engine/core/journal"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	enginejobs "github.com/jterrazz/studio-station/apps/engine/engine/jobs"
	journalcalls "github.com/jterrazz/studio-station/apps/engine/journal/calls"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinefilecontract "github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	portscalls "github.com/jterrazz/studio-station/apps/engine/ports/calls"
	settingscalls "github.com/jterrazz/studio-station/apps/engine/settings/calls"

	"github.com/spf13/cobra"

	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginelog "github.com/jterrazz/studio-station/apps/engine/engine/adapter/log"
	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	enginelive "github.com/jterrazz/studio-station/apps/engine/engine/live"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/engine/readers"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/adapter/dial"
	machinescalls "github.com/jterrazz/studio-station/apps/engine/machines/calls"
	machinescontract "github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/darwin"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/remote"
	"github.com/jterrazz/studio-station/apps/engine/remote/api"
	enginerun "github.com/jterrazz/studio-station/apps/engine/surface/cli/engine/run"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	"github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func (m machine) guestPort() int { return m.port(22, "STATION_GUEST_SSH_PORT") }

// value: how long the engine answers a re-ask with a reading its calls
// share (the apps, the disk walk, the overview's checks).
const engineKeep = 10 * time.Minute

// invariant: inside the engine, the menu bar reads through this run's
// Router, so its fleet reaches the other Macs as the CLI's does; the
// readings its calls share are read again once they are 10 min old, so
// what changed outside the engine reaches its next answer.
func (m machine) engine(env kit.Env) *process.Engine {
	m.pool.Keep(engineKeep)
	owed := &engineGrants{}
	registry := m.registry(env)
	calls := env.Calls
	if calls == nil {
		calls = remote.NewRouter(registry, m.profile, remote.Reach{})
	}
	live := &enginelive.Live{Calls: calls, Clock: m.sys.Clock, ThisMac: m.where.Machine, MacOf: m.macOf}
	pages := feeds.Pages{ThisMachine: m.where}
	report := func(v awake.View) *awake.Report {
		return &awake.Report{Now: m.sys.Clock.Now(), Zone: time.Local, Engine: awake.EngineRunning, Caps: m.profile.Caps, Reading: v.Reading, Session: v.Session, LastEnd: v.LastEnd, Guards: v.Guards,
			NeverSleeps: v.NeverSleeps, NeverSince: v.NeverSince, NeverHeld: v.NeverHeld, MacNeverSleeps: m.sleep.Kept(context.Background())}
	}
	guests := engineGuests{dialer: m.guestDialer(), vms: m.machines(false), files: m.sys.FS}
	notifier := func(write func(subject, text string)) process.Notifier { return enginelog.New(write) }
	var tailnet sharingNode = m.tailnetCLI().Tracked(engine.ChildrenDir(m.paths.State))
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
			seen = append(seen, enginejobs.Seen{Ref: coremachine.Ref{Name: m.profile.Ref.Name}, Kind: coremachine.Computer})
			for _, r := range running {
				seen = append(seen, enginejobs.Seen{Ref: coremachine.Ref{Name: r.Name, On: m.profile.Ref.Name}, Kind: r.Lifetime.Kind(), PID: r.PID, Guest: r.Guest})
			}
			return seen
		},
	}
	jobs := append([]process.Tick{sample.Run}, m.macJobs(calls)...)
	state := m.readers(calls, report, func() awake.View { return d.AwakeView() })
	var deviceKey string
	if m.profile.Kind == coremachine.Computer {
		deviceKey = DeviceKeyFile(m.paths.State)
	}
	d = process.New(engineMac{m.sys}, engineHypervisor{m.hypervisor()}, guests, m.machines(false), tailnet, process.Stores{Ports: m.stores.ports(), Reservations: m.stores.reservations}, process.Options{
		Remember: func(vm string, port int, stopped bool) error {
			return portscalls.Remember(m.stores.ports(), vm, port, stopped)
		},
		ForgetStops: func(vms []string) error {
			return portscalls.ForgetStopsBut(m.stores.ports(), vms)
		},
		State: m.paths.State, DeviceKey: deviceKey, Agent: m.agentOpener(), ThisMachine: m.kind, Ports: m.cat.Ports(), Rights: m.portRights, Installs: m.vmInstalls, Mounts: m.vmMounts, KindOf: m.vmKind, Version: m.engineVersion(), Abilities: m.engineAbilities(), Page: page.Revision, StartedAt: m.engineStarted(), PID: os.Getpid(), GuestPort: m.guestPort(),
		Power: m.powerPort, KeepAwakePage: m.keepAwakePage, Executable: m.sys.Env.Executable(), Records: m.engineRecords(), Commands: local.Ran,
		NeverSleeps: func() bool { on, _ := m.stores.neverSleeps().Get(); return on },
		GrantsOwed:  owed, Peer: m.engineAPI(),
		Pages: map[string]process.Projection{engine.PageMenubar: projection(pages.Home), engine.PageMachine: projection(pages.Machine),
			engine.PagePorts: projection(pages.Ports), engine.PageAwake: projection(pages.Awake), engine.PageAccounts: projection(pages.Accounts), engine.PageNeeds: projection(pages.Needs), engine.PageFind: projection(pages.Find)},
		Live:     map[string]process.LivePage{engine.PageActivity: enginelive.Activity{Calls: calls, Clock: m.sys.Clock, ThisMachine: m.where, Live: live}.Page},
		Sample:   live.Sample,
		KeptLive: live.Kept,
		Notifier: notifier, Checks: checks, Join: func(parts ...string) string { return strings.Join(parts, token.Sep) },
		Listening: m.listeners("/").Read, Declarations: m.stores.declarations.List,
		Jobs:     jobs,
		TellBack: m.machines(false).TellBack, Answers: m.answers, ToldBack: m.machines(false).ToldBack, Label: m.label,
		Drift:        func(ctx context.Context, vm string) model.KeptDrift { return keptDrift(ctx, calls, vm) },
		Changes:      driftChanges(),
		Beside:       m.beside(),
		LocalNetwork: m.localNetwork(),
		Reading:      state,
		Kept: func(ctx context.Context, line corejournal.Line) {
			if m.kind == catalog.Mac {
				_ = journalcalls.Keep(ctx, m.journalDeps(), journalcalls.Ran{Machine: m.profile.Ref.Name, By: corejournal.ByStation}, line)
			}
		},
	})
	if m.world == nil && m.kind == catalog.Mac {
		owed.g = m.vaultKept(d).Grants()
	}
	if m.world != nil {
		owed.forget = m.world.forgetGrants
	}
	m.far.engine = d
	d.ServeWith(m.engineServer(d, registry, calls, m.board(state)))
	return d
}

// value: a page projection as the engine serves it: the page in its
// document, which the menu bar refuses without its schema.
func projection(p func(*store.Snapshot, string) page.Page) process.Projection {
	return func(s *store.Snapshot, subject string) any {
		return page.Document{Schema: page.Schema, Page: p(s, subject)}
	}
}

// invariant: the engine's store is one engine's life, its epoch that
// engine's start; its readers run beside the tick, inline in a world, so a
// command's first frame reads what its engine read at its first tick.
func (m machine) readers(calls call.Router, report func(awake.View) *awake.Report, view func() awake.View) *readers.Set {
	now := m.sys.Clock.Now()
	st := store.New(store.EpochOf(cmp.Or(m.engineStarted(), now), os.Getpid()), now, func(err error) *fail.Error { return kit.Translate(err, "") })
	var run func(func())
	lazy := m.world != nil
	if lazy {
		run = func(f func()) { f() }
	}
	own := readers.Local{Calls: calls, ThisMac: m.where.Machine, Awake: func() awake.Report { return *report(view()) }}
	if m.kind == catalog.Mac {
		own.Known = m.machines(false).Known
	}
	var far *readers.Far
	if m.world == nil || m.world.holdsFar() {
		far = &readers.Far{Calls: calls, Macs: m.machines(false).OtherMacs, Following: m.world.watchContext, Drop: func(mac string) {
			if m.far != nil && m.far.router != nil {
				m.far.router.Drop(mac)
			}
		}}
	}
	return readers.New(readers.Options{Store: st, Clock: m.sys.Clock, Run: run, Readers: own.Readers(), Lazy: lazy, Far: far})
}

// invariant: a workspace's reading of its file is machinefile.drift's
// count over every chapter: it reads unbound only when no Machinefile is
// named after the VM and none is bound to it, the refusal naming the VM
// itself, by no extends and no path; any other refusal, a missing extends
// or a bound file gone, is the failure, worded as the CLI words it.
func keptDrift(ctx context.Context, calls call.Router, vm string) model.KeptDrift {
	out, err := call.Invoke(ctx, calls, coremachine.Ref{}, machinefilecalls.Drift, machinefilecalls.DriftIn{Machine: vm}, nil)
	switch {
	case err == nil:
		return model.KeptDrift{Differences: out.Differences}
	case unbound(err, vm):
		return model.KeptDrift{Unbound: true}
	}
	return model.KeptDrift{Failed: kit.Translate(err, "").Message}
}

func unbound(err error, vm string) bool {
	var refused *fail.Error
	return errors.As(err, &refused) && refused.Code == machinefilecalls.NotFound && refused.Subject == vm && refused.Detail["by"] == "" && refused.Detail["path"] == ""
}

// value: the calls whose end on a VM changes how it stands with its file.
func driftChanges() []string {
	return []string{
		machinefilecalls.Apply.Name, machinefilecalls.Upgrade.Name, appscalls.Install.Name, appscalls.Remove.Name, appscalls.Update.Name,
		appscalls.Move.Name, settingscalls.Apply.Name, settingscalls.Off.Name,
	}
}

// value: how the engine runs a reading beside its loop: in a goroutine of
// its own; in a world once the command is done.
func (m machine) beside() func(func()) {
	if m.world != nil {
		return m.world.beside
	}
	return nil
}

func (m machine) macJobs(calls call.Router) []process.Tick {
	if m.kind != catalog.Mac {
		return nil
	}
	return []process.Tick{m.nightly(calls).Run, m.releaseJob(calls).Run}
}

// invariant: a Mac's engine upgrades its bound Machinefile every night,
// unattended, through this machine's own calls; in a world the night runs
// inside the step, so a story reads what it did once the step ends.
func (m machine) nightly(calls call.Router) *enginejobs.Nightly {
	prefs := machinefilecontract.NewSourcePrefs(m.stores.prefs)
	run := func(f func()) { go f() }
	if m.world != nil {
		run = func(f func()) { f() }
	}
	return &enginejobs.Nightly{
		Clock: m.sys.Clock, Started: cmp.Or(m.engineStarted(), m.sys.Clock.Now()), Go: run,
		Bound: func() (string, time.Time, bool) {
			name, err := prefs.Bound()
			at, _ := prefs.BoundAt()
			return name, at, err == nil && name != ""
		},
		Last: func(ctx context.Context) (machinefile.Night, bool, error) {
			out, err := call.Invoke(ctx, calls, coremachine.Ref{}, machinefilecalls.LastNight, machinefilecalls.NightlyIn{}, nil)
			return out.Night, out.Ran, err
		},
		Upgrade: func(ctx context.Context, name string) error {
			_, err := call.Invoke(call.WithBy(ctx, corejournal.ByNight), calls, coremachine.Ref{}, machinefilecalls.Upgrade, machinefilecalls.UpgradeIn{Machinefile: name, Unattended: true}, nil)
			return err
		},
	}
}

func (m machine) releaseJob(calls call.Router) *enginejobs.Release {
	run := func(f func()) { go f() }
	if m.world != nil {
		run = func(f func()) { f() }
	}
	return &enginejobs.Release{
		Clock: m.sys.Clock, Go: run, Kept: m.stores.release.Read,
		Refresh: func(ctx context.Context) error {
			_, err := call.Invoke(ctx, calls, coremachine.Ref{}, enginecalls.RefreshRelease, enginecalls.ReleaseIn{}, nil)
			return err
		},
	}
}

// value: what the engine of a world Mac says of itself, the version it runs
// and when it started; outside a world, this binary's, started now.
func (m machine) engineVersion() string {
	if m.world != nil && m.world.engineVersion() != "" {
		return m.world.engineVersion()
	}
	return Version()
}

// value: the abilities the engine's hello lists: this binary's, but those a
// world Mac's older engine lacks until it starts again.
func (m machine) engineAbilities() []string {
	if m.world == nil {
		return engine.Abilities()
	}
	return slices.DeleteFunc(engine.Abilities(), m.world.engineLacks)
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

// invariant: the retry job takes out the removals a grant still owes a
// machine, each by its program's Remove; a VM owes none, and a world's
// grants only go with their machine.
type engineGrants struct {
	g      *keychain.Grants
	forget func(ctx context.Context, vm string) error
}

func (l *engineGrants) Owed() (map[string][]accounts.Program, error) {
	if l.g == nil {
		return nil, nil
	}
	return l.g.Owed(context.Background())
}

func (l *engineGrants) Retry(ctx context.Context, vm string, programs []accounts.Program) error {
	if l.g == nil {
		return nil
	}
	return l.g.Retry(ctx, vm, programs)
}

func (l *engineGrants) Forget(ctx context.Context, vm string) error {
	switch {
	case l.g != nil:
		return l.g.Forget(ctx, vm)
	case l.forget != nil:
		return l.forget(ctx, vm)
	}
	return nil
}

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
			Latency: v.Latency, Restart: v.Restart, Cause: v.Cause, LastLines: v.LastLines, ReadyAt: v.ReadyAt, Drift: v.Drift, Blocked: v.Blocked, GuestError: v.GuestError,
			Lockdown: v.Lockdown,
		}
	}
	return out, nil
}

func (d macEngine) GuestExec(ctx context.Context, vm, line, stdin string) (engine.GuestRan, error) {
	return d.GuestCall(ctx, vm, "", line, stdin)
}

func (d macEngine) GuestCall(ctx context.Context, vm, name, line, stdin string) (engine.GuestRan, error) {
	var ran engine.ExecOut
	if err := d.c.do(ctx, engine.CallGuestExec, engine.ExecIn{VM: vm, Line: line, Stdin: stdin, Call: name}, &ran); err != nil {
		return engine.GuestRan{}, downAs(err, model.ErrEngineDown)
	}
	return engine.GuestRan(ran), nil
}

func (d macEngine) Do(ctx context.Context, op, vm string) error {
	return downAs(d.c.do(ctx, engine.CallVM(op), engine.VMIn{VM: vm}, nil), model.ErrEngineDown)
}

func (d macEngine) EnvPorts(ctx context.Context, vm string) ([]model.Reserved, error) {
	var v ports.MacPorts
	if err := d.c.do(ctx, engine.CallPorts, nil, &v); err != nil {
		return nil, downAs(err, model.ErrEngineDown)
	}
	var out []model.Reserved
	for _, s := range v.Shared {
		if s.VM == vm && s.Profile != "" && s.Reserved > 0 {
			out = append(out, model.Reserved{
				GuestPort: s.GuestPort, MacPort: s.Reserved, Profile: s.Profile, URL: s.URLs.Public, PublicName: nameOf(s.URLs.Public),
				Off: localOnlyBecause(s.Detail),
			})
		}
	}
	return out, nil
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

func (d portsEngine) Reserve(ctx context.Context, vm string, port int, to catalog.Audience) (ports.SharedPort, error) {
	var v ports.SharedPort
	err := d.c.do(ctx, engine.CallReserve, engine.ReserveIn{VM: vm, GuestPort: port, To: to}, &v)
	return v, downAs(err, ports.ErrEngineDown)
}

func (d portsEngine) KeepsAudience(ctx context.Context) (version string, keeps bool, err error) {
	h, err := d.c.Hello(ctx)
	if err != nil {
		return "", false, downAs(err, ports.ErrEngineDown)
	}
	return h.Version, slices.Contains(h.Abilities, engine.AbilityReserveTo), nil
}

func (d portsEngine) Release(ctx context.Context, vm string, port int) (ports.Reservation, error) {
	var l ports.Reservation
	err := d.c.do(ctx, engine.CallRelease, engine.ReserveIn{VM: vm, GuestPort: port}, &l)
	return l, downAs(err, ports.ErrEngineDown)
}

type reservationReleaser struct{ m machine }

func (r reservationReleaser) ReleaseVM(ctx context.Context, vm string) error {
	// invariant: a VM leaves nothing of its ports on its Mac: its ports
	// declared public, the stops of sharing the owner chose, its reserved
	// ports and their shares.
	if err := portscalls.ForgetDeclarations(ctx, r.m.stores.declarations, vm); err != nil {
		return err
	}
	if err := portscalls.ForgetStops(r.m.stores.ports(), vm); err != nil {
		return err
	}
	reserved := r.m.stores.reservations
	reservations, err := reserved.List(ctx)
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
	return portscalls.SaveReservations(ctx, reserved, kept)
}

func (m machine) installer() engineInstaller {
	return engineInstaller{m: m, menuBar: darwin.Agents{}}
}

type engineInstaller struct {
	m       machine
	menuBar loginItem
}

type loginItem interface {
	// value: registers the login item of the app whose bundle holds this
	// binary, named by its bundle identifier, which macOS starts now and at
	// every login.
	ShowLoginItem(identifier string) error
	HideLoginItem(identifier string) error
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

// invariant: ONE engine per system; it restarts the engine only when its
// plist changed or it runs another version, and the VMs survive that restart.
func (i engineInstaller) Install(ctx context.Context) (engine.Installed, error) {
	return i.run(ctx, false)
}

func (i engineInstaller) Restart(ctx context.Context) (engine.Installed, error) {
	return i.run(ctx, true)
}

func (i engineInstaller) run(ctx context.Context, restart bool) (engine.Installed, error) {
	jobs := i.m.jobs(0)
	if !jobs.Present() {
		names := jobs.Names()
		return engine.Installed{}, &enginecalls.NoJobManagerError{Manager: names.Manager, Command: names.Command}
	}
	return i.install(ctx, jobs, restart)
}

// invariant: the engine's job goes in the mode this Mac is in, and a job
// already gone is no failure.
func (i engineInstaller) Uninstall(ctx context.Context) error {
	return i.m.jobs(0).Remove(ctx, i.m.engineJobRef())
}

func startFailed(cause error) error {
	return &engine.StartFailedError{Last: cause.Error(), Cause: cause}
}

func (i engineInstaller) install(ctx context.Context, jobs contract.Jobs, restart bool) (engine.Installed, error) {
	started, shown := false, false
	bundle := i.m.bundle()
	if bundle != "" {
		changed, showed, err := i.intoTheBundle(ctx, bundle)
		if err != nil {
			return engine.Installed{}, startFailed(err)
		}
		started, shown = changed, showed
	}
	done, ran, err := i.installEngine(ctx, jobs, started || restart)
	if err == nil && bundle != "" && ran && !shown {
		done.MenuBar = i.restartMenuBar(ctx, jobs)
	}
	return done, err
}

func (i engineInstaller) restartMenuBar(ctx context.Context, jobs contract.Jobs) engine.MenuBarLeft {
	job := contract.Ref{Domain: contract.Domain{UID: i.m.sys.Env.Euid()}, Label: catalog.MenuBarBundle}
	names := jobs.Names()
	kick := names.Command + " kickstart -k gui/" + strconv.Itoa(job.UID) + "/" + job.Label
	if jobs.Read(ctx, job).PID == 0 {
		return engine.MenuBarLeft{}
	}
	if err := i.menuBar.HideLoginItem(catalog.MenuBarBundle); err != nil {
		return engine.MenuBarLeft{Why: "still runs the copy the update moved away: " + err.Error(), Fix: kick}
	}
	held := true
	for range menuBarPolls {
		if held = jobs.Read(ctx, job).Loaded; !held || i.m.sys.Clock.Sleep(ctx, menuBarPoll) != nil {
			break
		}
	}
	var err error
	for range menuBarShows {
		if err = i.menuBar.ShowLoginItem(catalog.MenuBarBundle); err == nil {
			break
		}
		_ = i.m.sys.Clock.Sleep(ctx, menuBarShowPause)
	}
	switch {
	case err != nil:
		return engine.MenuBarLeft{Why: "not shown again: " + err.Error(), Fix: "switch Show in menu bar on in Spwn's Settings"}
	case held:
		return engine.MenuBarLeft{Why: names.Manager + " still held it " + (menuBarPolls * menuBarPoll).String() + " after its unregister", Fix: kick}
	}
	return engine.MenuBarLeft{}
}

const (
	menuBarPoll      = 100 * time.Millisecond
	menuBarPolls     = 200
	menuBarShows     = 3
	menuBarShowPause = time.Second
)

// invariant: a Mac moves from the bare jobs to the bundle's once, at the
// bundle's first start, and keeps nothing of them: each is booted out,
// waited out of launchd and its definition deleted before the bundle
// registers its own label, which no record of the bare jobs matches, and
// the VMs outlive the restart and are adopted again (ADR-073). Then the
// standalone menu bar goes, the bundle's menu bar is shown once, the engine
// job's start is recorded, and ~/.local/bin/station links into the bundle;
// started tells whether what the engine's job starts with changed.
func (i engineInstaller) intoTheBundle(ctx context.Context, bundle string) (started, shown bool, err error) {
	bundled := i.m.bundledJobs(process.JobBudget)
	for _, job := range i.m.bareJobRefs() {
		if _, statErr := i.m.sys.FS.Stat(job.File); statErr != nil {
			continue
		}
		if retireErr := bundled.Retire(ctx, job); retireErr != nil {
			return false, false, retireErr
		}
	}
	if standaloneErr := i.retireStandaloneMenuBar(ctx); standaloneErr != nil {
		return false, false, standaloneErr
	}
	if shown, err = i.showMenuBarOnce(); err != nil {
		return false, false, err
	}
	if started, err = i.writeJobStart(); err != nil {
		return false, false, err
	}
	return started, shown, i.linkIntoTheBundle(filepath.Join(bundle, catalog.BundledEnginePath))
}

// invariant: the menu bar an earlier release installed on its own leaves with
// the bare jobs, since Spwn.app carries it: its own binary unregisters its
// login item, its running copy quits, and its bundle is deleted.
func (i engineInstaller) retireStandaloneMenuBar(ctx context.Context) error {
	app := filepath.Join(i.m.sys.Env.Home(), catalog.StandaloneMenuBarPath)
	bin := filepath.Join(app, "Contents", "MacOS", "StationMenuBar")
	if _, statErr := i.m.sys.FS.Stat(bin); statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return nil
		}
		return statErr
	}
	_, _ = i.m.sys.Run.Output(ctx, local.Cmd{Name: bin, Args: []string{"--unregister-login-item"}, Timeout: standaloneMenuBarBudget})
	i.quitRunning(ctx, bin)
	return i.m.sys.FS.RemoveAll(app)
}

// invariant: the bundle's menu bar is on by default: the first start of the
// bundle on a Mac shows it at every login, then records that it did, so no
// later start, an update's among them, shows it again and the Off a person
// chose in Settings stays. A refusal of macOS is recorded alike, since the
// window reads what macOS holds and its switch alone changes it from there.
func (i engineInstaller) showMenuBarOnce() (bool, error) {
	record := catalog.MenuBarOfferedPath(i.m.sys.Env.Home())
	if _, err := i.m.sys.FS.Stat(record); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	_ = i.menuBar.ShowLoginItem(catalog.MenuBarBundle)
	if err := i.m.sys.FS.MkdirAll(filepath.Dir(record), 0o700); err != nil {
		return true, err
	}
	return true, i.m.sys.FS.Replace(record, nil, 0o600)
}

const standaloneMenuBarBudget = 5 * time.Second

func (i engineInstaller) quitRunning(ctx context.Context, executable string) {
	listed, err := i.m.sys.Run.Output(ctx, local.Cmd{Name: "/bin/ps", Args: []string{"-axo", "pid=,comm="}})
	if err != nil {
		return
	}
	for _, line := range strings.Split(listed.Stdout, "\n") {
		pid, path, _ := strings.Cut(strings.TrimSpace(line), " ")
		if n, convErr := strconv.Atoi(pid); convErr == nil && strings.TrimSpace(path) == executable {
			_ = i.m.sys.Run.Signal(n, local.SigTerm)
		}
	}
}

func (i engineInstaller) writeJobStart() (bool, error) {
	body, err := json.MarshalIndent(local.JobStart{Env: i.env(), Logs: map[string]string{
		catalog.BundledEngineLabel: filepath.Join(i.m.paths.State, "logs", "engine.log"),
	}}, "", "  ")
	if err != nil {
		return false, err
	}
	file := catalog.JobStartPath(i.m.sys.Env.Home())
	if current, _ := i.m.sys.FS.ReadFile(file); bytes.Equal(current, body) {
		return false, nil
	}
	if err := i.m.sys.FS.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return false, err
	}
	return true, i.m.sys.FS.Replace(file, body, 0o600)
}

// invariant: the CLI on the PATH is the bundle's engine through a link, so
// the bare binary a move leaves is replaced, never kept beside it.
func (i engineInstaller) linkIntoTheBundle(engineFile string) error {
	link := filepath.Join(catalog.InstallDirIn(i.m.sys.Env.Home()), catalog.StationBinary)
	if target, err := i.m.sys.FS.Readlink(link); err == nil && target == engineFile {
		return nil
	}
	if err := i.m.sys.FS.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := i.m.sys.FS.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	return i.m.sys.FS.Symlink(engineFile, link)
}

func (i engineInstaller) installEngine(ctx context.Context, jobs contract.Jobs, started bool) (done engine.Installed, ran bool, err error) {
	job := i.m.engineJobRef()
	changed, err := jobs.Install(ctx, contract.JobSpec{
		Label: job.Label, Program: append([]string{i.m.sys.Env.Executable()}, catalog.EngineArgs...), Env: i.env(),
		AtLogin: true, KeepAlive: true, Log: filepath.Join(i.m.paths.State, "logs", "engine.log"), File: job.File,
	})
	if err != nil {
		return engine.Installed{}, false, startFailed(err)
	}
	changed = changed || started
	if !jobs.Read(ctx, job).Loaded {
		for _, run := range []func(context.Context, contract.Ref) error{jobs.Load, jobs.Start} {
			if runErr := run(ctx, job); runErr != nil {
				return engine.Installed{}, false, startFailed(runErr)
			}
		}
		return engine.Installed{Written: changed}, true, nil
	}
	running, err := i.m.engineAPI().Hello(ctx)
	version := Version()
	if changed || (err == nil && (running.Version != version || running.Agent.Unserved())) {
		if kickErr := jobs.Restart(ctx, job); kickErr != nil {
			return engine.Installed{}, false, startFailed(kickErr)
		}
		done = engine.Installed{Restarted: true}
		if err == nil && running.Version != version {
			done.From = running.Version
		}
		return done, true, nil
	}
	return engine.Installed{}, false, nil
}
