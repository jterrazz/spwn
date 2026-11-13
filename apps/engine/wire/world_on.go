//go:build world

package wire

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"

	"github.com/spf13/cobra"

	accountsfake "github.com/jterrazz/studio-station/apps/engine/accounts/adapter/fake"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	activityfake "github.com/jterrazz/studio-station/apps/engine/activity/adapter/fake"
	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	"github.com/jterrazz/studio-station/apps/engine/agents"
	agentsfake "github.com/jterrazz/studio-station/apps/engine/agents/adapter/fake"
	agentscontract "github.com/jterrazz/studio-station/apps/engine/agents/contract"
	appsfake "github.com/jterrazz/studio-station/apps/engine/apps/adapter/fake"
	appscontract "github.com/jterrazz/studio-station/apps/engine/apps/contract"
	awakefake "github.com/jterrazz/studio-station/apps/engine/awake/adapter/fake"
	awakecontract "github.com/jterrazz/studio-station/apps/engine/awake/contract"
	backgroundfake "github.com/jterrazz/studio-station/apps/engine/background/adapter/fake"
	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	corefake "github.com/jterrazz/studio-station/apps/engine/core/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	corecontract "github.com/jterrazz/studio-station/apps/engine/core/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	diskfake "github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
	diskcontract "github.com/jterrazz/studio-station/apps/engine/disk/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginefake "github.com/jterrazz/studio-station/apps/engine/engine/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	imagesfake "github.com/jterrazz/studio-station/apps/engine/images/adapter/fake"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	machinefilefake "github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	machinefilecontract "github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	portsfake "github.com/jterrazz/studio-station/apps/engine/ports/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/funnel"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
	remotefake "github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/unix"
	remotecontract "github.com/jterrazz/studio-station/apps/engine/remote/contract"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
	settingsfake "github.com/jterrazz/studio-station/apps/engine/settings/adapter/fake"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: how many quiet ticks end a stepped engine's settling, and the most
// ticks it may take: a world never waits on a clock of its own.
const (
	quietTicks = 3
	mostTicks  = 120
)

// invariant: the spec build's world: read from $STATION_WORLD at start (seeded
// from $STATION_WORLD_SEED the first time, its Macs given the hypervisor
// $STATION_WORLD_HYPERVISOR names when set), every fake over it, each of its
// Macs built in this process as its own binary would build it there, and
// the world written back once the command and every engine are done with it.
type worldRun struct {
	path string
	w    *world.World
	// invariant: macs is read and written under macsMu: an engine's guest
	// session asks for its VM's Mac while the command asks for another.
	macsMu sync.Mutex
	macs   map[string]*worldMac
	// value: builds another Mac of the world: its registry, its engine.
	far   func(w *worldMac) farMac
	moved *moves
	// value: the lock this run holds on its world file from open to close.
	held *os.File
	// value: the run only looks at its world (--page): its copy is never
	// written back.
	shared bool
}

// invariant: one Mac of the world as this process sees it: its fakes over
// its slice, its engine stepped in this process while the world says it runs.
type worldMac struct {
	run  *worldRun
	name string

	// invariant: serving guards start and served: a far Mac is served by the
	// first caller that needs it, from any of the engines' goroutines.
	serving sync.Mutex
	start   func() *process.Engine
	// invariant: mu guards starting, engine, began, tick and leave: an
	// engine's own goroutines (its surfaces, a guest session, another Mac's
	// engine) ask for this Mac's engine while the command ends it, and a
	// reader takes the pointer once under mu, never twice. It is never held
	// across a call into the engine, which may ask for this Mac again.
	mu       sync.Mutex
	starting *engineStart
	engine   *process.Engine
	// value: when the world says the running engine started: a Mac rebooted
	// under it, inside the command, ends it with its process.
	began  time.Time
	tick   int
	served *remotefake.Dialer
	bound  map[int]bool
	// value: this machine as its own binary builds it inside a VM of the
	// world, once per run.
	vm *farMac
	// value: the run's home, where the world's ~/ files are.
	home string
	// value: ends the surfaces the world holds open on the running engine.
	leave func()
}

// invariant: a run that only looks (--page) reads the
// world under the lock and lets go of it at once: it writes nothing back,
// so a slow look never holds a call up, nor undoes one.
func openWorld(vars map[string]string, look bool) (*worldRun, error) {
	path := vars["STATION_WORLD"]
	if path == "" {
		return nil, nil
	}
	held := lockWorld(path)
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) && vars["STATION_WORLD_SEED"] != "" {
		data, err = os.ReadFile(vars["STATION_WORLD_SEED"])
	}
	if err != nil {
		unlockWorld(held)
		return nil, err
	}
	w, err := world.Decode(data)
	if err != nil {
		unlockWorld(held)
		return nil, err
	}
	w.UseHypervisor(vars["STATION_WORLD_HYPERVISOR"])
	if look {
		unlockWorld(held)
		return &worldRun{path: path, w: w, macs: map[string]*worldMac{}, shared: true}, nil
	}
	r := &worldRun{path: path, w: w, macs: map[string]*worldMac{}, held: held}
	w.OnPause(r.pause)
	return r, nil
}

// invariant: a call held in a run that writes its world back writes it back
// as it stands and lets go of it, so a command run while it waits reads what
// it made so far; its end writes the world again, over the other's.
func (r *worldRun) pause() {
	if r.shared {
		return
	}
	if data, err := r.w.Encode(); err == nil {
		_ = os.WriteFile(r.path, data, 0o600)
	}
	unlockWorld(r.held)
	r.held = nil
}

// hazard: a run that runs station as a child while it holds its world would
// wait on it for ever: the wait is bounded, and past it the run goes on
// without the lock, as every run did before it.
func lockWorld(path string) *os.File {
	f, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil
	}
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil {
			return f
		}
	}
	_ = f.Close()
	return nil
}

func unlockWorld(f *os.File) {
	if f != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}

func (r *worldRun) mac(name string) *worldMac {
	r.macsMu.Lock()
	defer r.macsMu.Unlock()
	if m, ok := r.macs[name]; ok {
		return m
	}
	m := &worldMac{run: r, name: name}
	r.macs[name] = m
	return m
}

func (r *worldRun) thisMac() *worldMac { return r.mac(r.w.ThisMac) }

func (r *worldRun) farFrom(far func(w *worldMac) farMac) { r.far = far }

func (r *worldRun) names() []string {
	r.w.Lock()
	defer r.w.Unlock()
	return slices.Sorted(maps.Keys(r.w.Macs))
}

func (m *worldMac) slice() world.Mac {
	m.run.w.Lock()
	defer m.run.w.Unlock()
	return *m.run.w.Mac(m.name)
}

func (m *worldMac) runs() bool { mac := m.slice(); return mac.EngineRuns() }

func (m *worldMac) system(h local.System, cat catalog.Catalog) local.System {
	foot := appsfake.FootprintOf(m.run.w, m.name, cat)
	h.Run = appsfake.Seen{Runner: rootless{Runner: h.Run, w: m.run.w}, Foot: foot, Home: h.Env.Home()}
	m.home = h.Env.Home()
	h.FS = worldFS{FS: appsfake.SeenFS{FS: h.FS, Foot: foot, Home: h.Env.Home()}, m: m, plist: catalog.EnginePlist(h.Env.Home()), files: m.files()}
	h.Env = worldEnv{Env: h.Env, thisMac: m.name, user: m.run.w.User, admin: m.run.w.Admin && m.name == m.run.w.ThisMac || m.slice().Root}
	h.Clock = m.run.clock()
	h.Sockets = worldSockets{Sockets: h.Sockets, m: m}
	return h
}

func (m *worldMac) hypervisor(name string, install catalog.Install) contract.Hypervisor {
	return machinesfake.HypervisorOf(m.run.w, m.name, name, install)
}

// invariant: station's own state in a world is its Mac's slice: every store
// is the fake over it, never a file of the run's home.
func (m *worldMac) stores(st *stores) {
	m.run.w.Lock()
	slice := m.run.w.Mac(m.name)
	if slice.Preferences == nil {
		slice.Preferences = map[string]json.RawMessage{}
	}
	if slice.Files == nil {
		slice.Files = map[string]string{}
	}
	st.prefs = corefake.PreferencesOver(slice.Preferences, m.name, m.run.w).Over(slice.Files).Renaming(contract.RetiredGroupMacs, contract.GroupMacs).
		Renaming(setupcontract.RetiredGroupName, setupcontract.GroupName).Renaming(portscontract.RetiredGroupPorts, portscontract.GroupPorts)
	m.run.w.Unlock()
	w := m.run.w
	st.machines = machinesfake.StoreOf(w, m.name)
	st.images = imagesfake.StoreOf(w, m.name)
	st.shares = accountsfake.SharesOf(w, m.name)
	st.reservations = portsfake.ReservationsOf(w, m.name)
	st.declarations = portsfake.DeclarationsOf(w, m.name)
	st.applied = settingsfake.AppliedOf(w, m.name)
	st.uptime = machinesfake.UptimeOf(w, m.name)
	st.ledger = machinefilefake.LedgerOf(w, m.name)
}

func (m *worldMac) self() corecontract.Self    { return corefake.SelfOf(m.run.w, m.name) }
func (m *worldMac) power() awakecontract.Power { return awakefake.PowerOf(m.run.w, m.name) }
func (m *worldMac) guests() contract.GuestDialer {
	return worldGuests{GuestDialer: machinesfake.GuestsOf(m.run.w).Serving(engine.GuestSession), r: m.run}
}
func (m *worldMac) imageSource(hypervisor string) imagescontract.ImageSource {
	return imagesfake.SourceOf(m.run.w, hypervisor)
}
func (m *worldMac) sshConfig() contract.SSHConfig { return machinesfake.SSHConfigOf(m.run.w, m.name) }
func (m *worldMac) unlocker() contract.Unlocker   { return machinesfake.MacsOf(m.run.w) }
func (m *worldMac) waker() contract.Waker         { return machinesfake.WakerOf(m.run.w, m.name) }
func (m *worldMac) jobs() backgroundcontract.Jobs { return backgroundfake.New(m.run.w, m.name) }
func (m *worldMac) meter() activitycontract.Meter { return activityfake.MeterOf(m.run.w, m.name) }
func (m *worldMac) disk() diskcontract.DiskUsage  { return m.files() }

func (m *worldMac) files() *diskfake.Usage        { return diskfake.UsageOf(m.run.w, m.name).At(m.home) }
func (m *worldMac) tailnet() sharingNode          { return portsfake.TailnetOf(m.run.w, m.name) }
func (m *worldMac) funnel() funnel.Funnel         { return portsfake.TailnetOf(m.run.w, m.name) }
func (m *worldMac) bundler() appscontract.Bundler { return appsfake.BundlerOf(m.run.w, m.name) }
func (m *worldMac) userSource() machinefilecontract.Source {
	return machinefilefake.UserOf(m.run.w, m.name)
}
func (m *worldMac) owned() machinefilecontract.Owned { return machinefilefake.OwnedOf(m.run.w, m.name) }

func (m *worldMac) locks() machinefilecontract.Locks { return machinefilefake.LocksOf(m.run.w, m.name) }
func (m *worldMac) turn() machinefilecontract.Turn   { return machinefilefake.TurnOf(m.run.w, m.name) }
func (m *worldMac) password() machinefilecontract.Password {
	return machinefilefake.PasswordOf(m.run.w, m.name)
}
func (m *worldMac) phaseShell() machinefilecontract.Shell { return machinefilefake.ShellOf(m.run.w) }
func (m *worldMac) sharedSource(src machinefilecontract.SharedSource) machinefilecontract.Source {
	return machinefilefake.SharedOf(m.run.w, m.name, src)
}

func (m *worldMac) managers(adapters map[catalog.Method]appscontract.Installer) {
	named := make(map[catalog.Method]world.Named, len(adapters))
	for method, real := range adapters {
		named[method] = appsfake.Named(real)
	}
	m.run.w.Managers = named
}
func (m *worldMac) vault() accountscontract.Vault   { return accountsfake.NewVault(m.run.w, m.name) }
func (m *worldMac) journal() agentscontract.Journal { return agentsfake.JournalOf(m.run.w, m.name) }
func (m *worldMac) agentSession() agentscontract.Session {
	return agentsfake.SessionOf(m.run.w, m.name)
}
func (m *worldMac) forge() agentscontract.Forge { return agentsfake.ForgeOf(m.run.w, m.name) }
func (m *worldMac) multiplexer() agentscontract.Multiplexer {
	return agentsfake.MultiplexerOf(m.run.w, m.name)
}
func (m *worldMac) hooks(app agents.App) agentscontract.Hooks {
	return agentsfake.HooksOf(m.run.w, m.name, app)
}
func (m *worldMac) listeners() portscontract.Listeners {
	return portsfake.ListenersOf(m.run.w, m.name)
}

// value: every procedure of the catalogue as a fake over the Mac's slice:
// no world runs a procedure of the real machine.
func (m *worldMac) procedureAppliers() []settingscontract.Applier {
	var out []settingscontract.Applier
	for _, name := range catalog.Procedures() {
		if p := catalog.Procedure(name); p != catalog.ProcNeverSleeps {
			out = append(out, settingsfake.ApplierOf(m.run.w, m.name, p.Kind()))
		}
	}
	return out
}

func (m *worldMac) inspector() settingscontract.Inspector {
	return settingsfake.NewInspector(m.run.w, m.name)
}

func (m *worldMac) installer(method catalog.Method, lined bool) appscontract.Installer {
	return appsfake.InstallerOf(m.run.w, m.name, method, lined)
}

func (m *worldMac) identities() accountscontract.Identities {
	return accountsfake.NewIdentities(m.run.w, m.name)
}

func (m *worldMac) secretSource() accountscontract.SecretSource {
	return accountsfake.NewSecretSource(m.run.w, m.name)
}

func (m *worldMac) checks() contract.CheckRunner { return machinesfake.CheckRunnerOf(m.run.w) }

func (m *worldMac) notifier() process.Notifier {
	return lockedNotifier{w: m.run.w, n: enginefake.NewNotifier(m.run.w)}
}

func (m *worldMac) engineFrom(start func() *process.Engine) { m.start = start }

func (m *worldMac) engineVersion() string {
	if e := m.slice().Engine; e != nil {
		return e.Version
	}
	return ""
}

func (m *worldMac) stationVersion() string { return m.slice().Station }

func (m *worldMac) absent(program string) bool { return slices.Contains(m.slice().Absent, program) }

func (m *worldMac) engineStarted() time.Time {
	if e := m.slice().Engine; e != nil {
		return e.StartedAt
	}
	return time.Time{}
}

// invariant: a stop the owner asked is kept on the Mac's engine of the
// world, as the real one keeps it beside its socket.
func (m *worldMac) stopMarks() engine.StopMarks { return worldMarks{m: m} }

type worldMarks struct{ m *worldMac }

func (w worldMarks) Read() (engine.StopMark, bool) {
	e := w.m.slice().Engine
	if e == nil || e.StoppedAt.IsZero() {
		return engine.StopMark{}, false
	}
	return engine.StopMark{At: e.StoppedAt, By: e.StoppedBy}, true
}

func (w worldMarks) Write(mark engine.StopMark) error {
	w.m.run.w.Lock()
	defer w.m.run.w.Unlock()
	if e := w.m.run.w.Mac(w.m.name).Engine; e != nil {
		e.StoppedAt, e.StoppedBy = mark.At, mark.By
	}
	return nil
}

// invariant: the engine's job definition is the job manager's file; in a
// world it is there while the Mac's engine is set up, whatever the run's
// home holds.
type worldFS struct {
	local.FS
	m     *worldMac
	plist string
	// value: the files of the Mac's disk in the world, seen beside the
	// run's home: what an app keeps there is the world's, never real bytes.
	files *diskfake.Usage
}

func (f worldFS) Stat(p string) (fs.FileInfo, error) {
	if p != f.plist {
		info, err := f.FS.Stat(p)
		if err != nil && f.files.Has(p) {
			return plistInfo{name: filepath.Base(p), dir: f.files.Folder(p)}, nil
		}
		return info, err
	}
	f.m.run.w.Lock()
	mac := f.m.run.w.Mac(f.m.name)
	set := mac.Engine != nil || mac.Jobs[catalog.EngineLabel] != nil
	f.m.run.w.Unlock()
	if !set {
		return nil, &fs.PathError{Op: "stat", Path: p, Err: fs.ErrNotExist}
	}
	return plistInfo{name: filepath.Base(p)}, nil
}

func (f worldFS) Glob(pattern string) ([]string, error) {
	found, err := f.FS.Glob(pattern)
	if err != nil {
		return nil, err
	}
	for _, p := range f.files.Glob(pattern) {
		if !slices.Contains(found, p) {
			found = append(found, p)
		}
	}
	slices.Sort(found)
	return found, nil
}

func (f worldFS) ReadDir(p string) ([]fs.DirEntry, error) {
	entries, err := f.FS.ReadDir(p)
	names := f.files.Children(p)
	if err != nil && len(names) == 0 {
		return nil, err
	}
	for _, name := range names {
		if !slices.ContainsFunc(entries, func(e fs.DirEntry) bool { return e.Name() == name }) {
			full := filepath.Join(p, name)
			entries = append(entries, fs.FileInfoToDirEntry(plistInfo{name: name, dir: f.files.Folder(full)}))
		}
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, nil
}

func (f worldFS) Remove(p string) error {
	if f.files.Folder(p) {
		return &fs.PathError{Op: "remove", Path: p, Err: syscall.ENOTEMPTY}
	}
	err := f.FS.Remove(p)
	if f.files.Has(p) {
		f.files.Drop([]string{p})
		return nil
	}
	return err
}

func (f worldFS) Rename(from, to string) error {
	if f.files.Has(from) {
		if err := f.files.Move(context.Background(), from, to); err != nil {
			return err
		}
		if _, err := f.FS.Stat(from); errors.Is(err, fs.ErrNotExist) {
			return nil
		}
	}
	return f.FS.Rename(from, to)
}

func (f worldFS) RemoveAll(p string) error {
	f.files.Drop([]string{p})
	return f.FS.RemoveAll(p)
}

type plistInfo struct {
	name string
	dir  bool
}

func (i plistInfo) Name() string { return i.name }
func (plistInfo) Size() int64    { return 0 }
func (i plistInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o755
	}
	return 0o644
}
func (plistInfo) ModTime() time.Time { return time.Time{} }
func (i plistInfo) IsDir() bool      { return i.dir }
func (plistInfo) Sys() any           { return nil }

func (m *worldMac) macs() remotecontract.EndpointDialer { return worldDialer{r: m.run} }

// invariant: a Mac's engine runs in this process while the world says it
// runs, and ends once it says otherwise or says it started since: begun at the first need of it,
// then stepped over the world's clock until it is quiet, so a story that
// waits on the engine is deterministic. One caller starts it, for every
// caller after it, so the start outlives the caller's context.
//
// invariant: a caller from outside the engine that finds a start in progress
// waits for it, or for its own context, and then reads the engine that start
// left, or its failure. Two callers are refused instead, since the start may
// be waiting on them and would never end: what the engine runs in its own
// lifetime (waits), and a job a pool shares (pooled), which runs on the
// pool's context, not its asker's, and which a start that reads through the
// pool joins by its key.
func (m *worldMac) up(ctx context.Context) (*process.Engine, bool) {
	for {
		d, began := m.running()
		if d != nil && m.runs() && !m.engineStarted().Equal(began) {
			m.down(true)
			d = m.current()
		}
		if d != nil {
			if m.runs() {
				return d, true
			}
			m.down(false)
			return nil, false
		}
		start := m.starter()
		if start == nil || !m.runs() {
			return nil, false
		}
		s, mine := m.beginStarting()
		if mine {
			return m.begin(context.WithoutCancel(ctx), s, start)
		}
		if ctx.Value(waits{}) != nil || ctx.Value(pooled{}) != nil {
			return nil, false
		}
		select {
		case <-s.done:
		case <-ctx.Done():
			return nil, false
		}
		if s.failed {
			return nil, false
		}
	}
}

func (m *worldMac) begin(ctx context.Context, s *engineStart, start func() *process.Engine) (d *process.Engine, ok bool) {
	defer func() { m.endStarting(s, ok) }()
	d = start()
	d.Remember(m.slice().Engine.Memory)
	if kept := m.slice().Engine.Awake; kept != nil {
		d.RememberAwake(*kept)
	}
	if kept := m.slice().Engine.Exposure; kept != nil {
		d.RememberExposure(*kept)
	}
	d.RememberSettings(m.slice().Engine.Settings)
	if err := d.Begin(context.WithValue(ctx, waits{}, true), func(string) {}); err != nil {
		return nil, false
	}
	if kept := m.slice().Engine.Activity; kept != nil {
		d.History().Take(kept)
	}
	// invariant: when it began is set before the engine is, so a surface of
	// that engine that asks for it meanwhile never reads it as a successor.
	m.hold(d, m.engineStarted())
	m.surfaces(ctx, d)
	// invariant: a start begun in an engine's lifetime (waits) leaves the
	// settling to the command: that command waits on this very start at its
	// end (close, up), and is the one that moves the clock a settle waits on.
	if ctx.Value(waits{}) == nil {
		m.settle(ctx)
	}
	return d, true
}

// invariant: the surfaces the world says watch this engine hold the very
// watches the real ones hold, over the engine's own streams, for as long as
// it runs in the command: the menu bar its page and events watches from
// launch, and This Mac's activity only while open.
func (m *worldMac) surfaces(ctx context.Context, d *process.Engine) {
	e := m.slice().Engine
	ctx, cancel := context.WithCancel(context.WithValue(ctx, waits{}, true))
	var held sync.WaitGroup
	hold := func(watch func(ctx context.Context, send func(any) error) error) {
		opened := make(chan struct{})
		var once sync.Once
		held.Add(1)
		go func() {
			defer held.Done()
			defer once.Do(func() { close(opened) })
			_ = watch(ctx, func(any) error { once.Do(func() { close(opened) }); return nil })
		}()
		<-opened
	}
	events := func(ctx context.Context, send func(any) error) error {
		_ = send(nil)
		return d.Events(ctx, send)
	}
	if e.MenuBar != "" {
		hold(func(ctx context.Context, send func(any) error) error {
			return d.Stream(ctx, []string{engine.PageMenubar}, send)
		})
		hold(events)
	}
	if e.MenuBar == "open" {
		hold(func(ctx context.Context, send func(any) error) error { return d.Watch(ctx, nil, send) })
	}
	if len(e.Watching) > 0 {
		hold(func(ctx context.Context, send func(any) error) error { return d.Watch(ctx, e.Watching, send) })
	}
	m.mu.Lock()
	m.leave = func() {
		cancel()
		held.Wait()
	}
	m.mu.Unlock()
}

func (m *worldMac) running() (*process.Engine, time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.engine, m.began
}

func (m *worldMac) current() *process.Engine {
	d, _ := m.running()
	return d
}

func (m *worldMac) hold(d *process.Engine, began time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.began, m.engine = began, d
}

// value: one start of a Mac's engine: done closes once it ended, and failed,
// set before, says it began no engine.
type engineStart struct {
	done   chan struct{}
	failed bool
}

func (m *worldMac) beginStarting() (*engineStart, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.starting != nil {
		return m.starting, false
	}
	m.starting = &engineStart{done: make(chan struct{})}
	return m.starting, true
}

func (m *worldMac) endStarting(s *engineStart, ok bool) {
	m.mu.Lock()
	m.starting = nil
	s.failed = !ok
	m.mu.Unlock()
	close(s.done)
}

func (m *worldMac) nextTick() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tick++
	return m.tick
}

func (m *worldMac) takeLeave() func() {
	m.mu.Lock()
	defer m.mu.Unlock()
	leave := m.leave
	m.leave = nil
	return leave
}

func (m *worldMac) settle(ctx context.Context) {
	quiet := 0
	for n := 0; n < mostTicks && quiet < quietTicks; n++ {
		tick := m.nextTick()
		_ = m.run.clock().Sleep(ctx, time.Second)
		d := m.current()
		if d == nil {
			return
		}
		if d.Step(ctx, tick) {
			quiet = 0
		} else {
			quiet++
		}
	}
}

// invariant: an engine that ends keeps what it knew in the world, unless its
// process ends with it (a reboot, a crash): its successor knows nothing.
func (m *worldMac) down(forget bool) {
	d := m.current()
	if d == nil {
		return
	}
	kept := d.AwakeKept()
	exposure := d.ExposureKept()
	memory := d.Memory()
	activity := d.History().Hand()
	read := d.SettingsKept()
	if leave := m.takeLeave(); leave != nil {
		left := m.run.clock().moved.leave()
		leave()
		left()
	}
	if !forget && !m.runs() {
		d.Stop()
	} else {
		d.Suspend()
	}
	m.release(d)
	if forget {
		memory = nil
	}
	m.run.w.Lock()
	defer m.run.w.Unlock()
	if e := m.run.w.Mac(m.name).Engine; e != nil {
		e.Memory = memory
		e.Awake, e.Exposure, e.Activity, e.Settings = &kept, &exposure, activity, read
		if forget || e.Stopped {
			e.Awake, e.Exposure, e.Activity, e.Settings = nil, nil, nil, nil
		}
	}
}

func (m *worldMac) release(d *process.Engine) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine == d {
		m.engine = nil
	}
}

func (m *worldMac) serve() *remotefake.Dialer {
	m.serving.Lock()
	defer m.serving.Unlock()
	return m.serveLocked()
}

// invariant: a Mac of the world other than This Mac is served once, by
// whichever caller needs it first, and its engine start is read under the
// same lock that wrote it.
func (m *worldMac) starter() func() *process.Engine {
	m.serving.Lock()
	defer m.serving.Unlock()
	if m.start == nil && m.name != m.run.w.ThisMac && m.run.far != nil && m.runs() {
		m.serveLocked()
	}
	return m.start
}

func (m *worldMac) serveLocked() *remotefake.Dialer {
	if m.served == nil {
		far := m.run.far(m)
		m.served = remotefake.New()
		m.served.Serve(m.name, remotefake.Mac{Registry: far.registry, Profile: far.profile, Version: cmp.Or(m.stationVersion(), Version()), Lacks: m.slice().Lacks, Word: func(err error) *fail.Error { return kit.Translate(err, "") }, Relay: far.relay})
		m.start = far.start
	}
	return m.served
}

// invariant: another Mac of the world answers while it is on the tailnet,
// unlocked and set up, its engine stopped or not, as `station engine attach`
// answers over ssh; it is then reached as over ssh, every value through the
// wire form. A fault of EndpointDialer.Dial is a Mac whose ssh answers with
// no station there.
type worldDialer struct{ r *worldRun }

func (d worldDialer) Dial(ctx context.Context, mac string) (remotecontract.Endpoint, error) {
	d.r.w.Lock()
	slice, ok := d.r.w.Macs[mac]
	reach := ok && !slice.Offline && !slice.Locked && slice.Engine != nil && !d.r.w.Now.Before(slice.Back)
	bare := reach && d.r.w.Fails("EndpointDialer.Dial")
	d.r.w.Unlock()
	if !reach || d.r.far == nil {
		return nil, fail.New(fail.Unreachable, call.Unreachable, mac, nil)
	}
	if bare {
		return nil, fail.New(fail.Failed, call.NoStation, mac, nil)
	}
	return d.r.mac(mac).serve().Dial(ctx, mac)
}

func (r *worldRun) command() *cobra.Command {
	return &cobra.Command{
		Use:    "world <event> [args]",
		Short:  "Apply an event to the spec world: a reboot, a crash, time passing",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			ev, err := world.Parse(args)
			if err != nil {
				return err
			}
			if ev.Name == "edit" && len(ev.Args) == 2 {
				text, readErr := io.ReadAll(c.InOrStdin())
				if readErr != nil {
					return readErr
				}
				ev.Args = append(ev.Args, string(text))
			}
			for _, name := range r.names() {
				r.mac(name).up(c.Context())
			}
			r.w.Lock()
			err = r.w.Apply(ev)
			r.w.Unlock()
			if err != nil {
				return err
			}
			for _, name := range r.names() {
				if m := r.mac(name); restarts(ev, name) || !m.runs() {
					m.down(true)
				}
			}
			return nil
		},
	}
}

// value: an event that ends a Mac's engine process: its successor adopts
// what still runs.
func restarts(ev world.Event, mac string) bool {
	switch ev.Name {
	case "reboot", "power-cut", "engine-crash", "unlock":
		return len(ev.Args) > 0 && ev.Args[0] == mac
	}
	return false
}

// invariant: once the command is done, every engine of the world takes its
// turn on it, as the real ones run beside every command, then the world is
// written back.
func (r *worldRun) close(ctx context.Context) error {
	if r.shared {
		return nil
	}
	defer func() { unlockWorld(r.held); r.held = nil }()
	for _, name := range r.names() {
		m := r.mac(name)
		if _, ok := m.up(ctx); ok {
			m.settle(ctx)
			m.down(false)
		}
	}
	r.w.Lock()
	data, err := r.w.Encode()
	r.w.Unlock()
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o600)
}

type worldEnv struct {
	local.Env
	thisMac, user string
	// value: this run has admin rights, on This Mac's own terminal.
	admin bool
}

func (e worldEnv) Hostname() string { return e.thisMac }
func (e worldEnv) User() string     { return e.user }

func (e worldEnv) Euid() int {
	if e.admin {
		return 0
	}
	return e.Env.Euid()
}

// invariant: the world's clock moves only when the story moves it: a
// command that waits, a stepped engine's tick, an advance. What an engine
// runs in its own lifetime (a guest session's backoff) waits for the clock
// instead of moving it, so no loop of its own runs the clock ahead.
type worldClock struct {
	w     *world.World
	moved *moves
}

type waits struct{}

type pooled struct{}

func (*worldMac) poolContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, pooled{}, true)
}

type moves struct {
	mu sync.Mutex
	ch chan struct{}
	// invariant: while a Mac's surfaces are being let go, a wait on the clock
	// does not wait for it to move: the flow that moves it waits on them.
	leaving atomic.Int32
}

func (r *worldRun) clock() worldClock {
	if r.moved == nil {
		r.moved = &moves{ch: make(chan struct{})}
	}
	return worldClock{w: r.w, moved: r.moved}
}

func (m *moves) next() <-chan struct{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ch
}

func (m *moves) leave() func() {
	m.leaving.Add(1)
	m.tell()
	return func() { m.leaving.Add(-1) }
}

func (m *moves) tell() {
	m.mu.Lock()
	close(m.ch)
	m.ch = make(chan struct{})
	m.mu.Unlock()
}

func (c worldClock) Now() time.Time {
	c.w.Lock()
	defer c.w.Unlock()
	return c.w.Now
}

func (c worldClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ctx.Value(waits{}) == nil {
		// invariant: what already runs takes its turn before the clock
		// leaps, and a moment of real time with it (a fiftieth of the leap,
		// at most 5 ms), so a poll counts the work it waits on, a guest's
		// pipe under a loaded gate included, not the scheduler.
		runtime.Gosched()
		time.Sleep(min(d/50, 5*time.Millisecond))
	}
	c.w.Lock()
	until := c.w.Now.Add(d)
	if ctx.Value(waits{}) == nil {
		c.w.Now = until
		c.w.Unlock()
		c.moved.tell()
		return nil
	}
	c.w.Unlock()
	for {
		moved := c.moved.next()
		if !c.Now().Before(until) {
			return nil
		}
		if c.moved.leaving.Load() > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Millisecond):
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-moved:
		}
	}
}

// invariant: a VM of the world whose own slice says what it is answers
// station's API as `station guest serve --stdio` answers inside it: its calls
// run over that slice, every value through the wire form; any other line is
// the scripted guest's.
type worldGuests struct {
	contract.GuestDialer
	r *worldRun
}

type worldGuest struct {
	contract.Guest
	vm string
	r  *worldRun
}

func (g worldGuests) Dial(ctx context.Context, t contract.GuestTarget) (contract.Guest, error) {
	c, err := g.GuestDialer.Dial(ctx, t)
	if err != nil {
		return nil, err
	}
	return worldGuest{Guest: c, vm: t.VM, r: g.r}, nil
}

func (g worldGuest) Exec(ctx context.Context, cmd contract.GuestCommand) (contract.GuestResult, error) {
	g.r.w.Lock()
	slice, ok := g.r.w.Macs[g.vm]
	ok = ok && slice.Self != nil
	g.r.w.Unlock()
	if ok {
		if res, handled, err := g.credentials(cmd); handled {
			return res, err
		}
	}
	if cmd.Line != unix.Guest || !ok || g.r.far == nil || cmd.Stdin == nil {
		return g.Guest.Exec(ctx, cmd)
	}
	far := g.r.mac(g.vm).inVM()
	var out bytes.Buffer
	w := cmd.Stdout
	if w == nil {
		w = &out
	}
	g.r.w.Lock()
	version := cmp.Or(slice.Station, Version())
	g.r.w.Unlock()
	err := serve.Stream(ctx, cmd.Stdin, w, serve.Local(far.registry, far.profile, version), func(err error) *fail.Error { return kit.Translate(err, "") })
	return contract.GuestResult{Stdout: out.String()}, err
}

// value: a VM of the world as its guest's hello placed it: the Mac that
// holds it and that Mac's name on the tailnet; false for a Mac.
func (m *worldMac) placement() (ports.VMLivesOn, bool) {
	w := m.run.w
	w.Lock()
	defer w.Unlock()
	mac := w.Holder(m.name)
	if mac == "" {
		return ports.VMLivesOn{}, false
	}
	p := ports.VMLivesOn{Mac: mac, VM: m.name}
	if n := w.Macs[mac].Tailnet; n != nil {
		p.PublicName = n.Self.Sharing.DNSName
	}
	return p, true
}

func (m *worldMac) inVM() farMac {
	if m.vm == nil {
		far := m.run.far(m)
		m.vm = &far
	}
	return *m.vm
}

// invariant: a probe's budget in a world is a deadline the story moves, never
// one that leaps the clock: a fake answers before it, however the scheduler
// orders the two.
type budgetClock struct{ worldClock }

func (c budgetClock) Sleep(ctx context.Context, d time.Duration) error {
	return c.worldClock.Sleep(context.WithValue(ctx, waits{}, true), d)
}

func (m *worldMac) budgets() lanes.Clock { return budgetClock{m.run.clock()} }

// invariant: a Mac's engine socket is its engine in this process, reached
// through a pipe; its loopback ports are the world's, busy when the Mac
// lists a listener there or the world already bound them, so no run binds a
// real port another run holds.
type worldSockets struct {
	local.Sockets
	m *worldMac
}

var errPortBusy = errors.New("the port is busy in the world")

func (s worldSockets) ListenTCP(_ string, port int) (local.Listener, error) {
	w := s.m.run.w
	w.Lock()
	defer w.Unlock()
	mac := w.Mac(s.m.name)
	if slices.ContainsFunc(mac.Listening, func(l ports.Listener) bool { return l.Port == port }) {
		return nil, errPortBusy
	}
	if s.m.bound == nil {
		s.m.bound = map[int]bool{}
	}
	if s.m.bound[port] {
		return nil, errPortBusy
	}
	s.m.bound[port] = true
	return &worldListener{port: port, done: make(chan struct{}), release: func() {
		w.Lock()
		defer w.Unlock()
		delete(s.m.bound, port)
	}}, nil
}

type worldListener struct {
	port    int
	done    chan struct{}
	once    sync.Once
	release func()
}

func (l *worldListener) Accept() (local.Conn, error) {
	<-l.done
	return nil, net.ErrClosed
}

func (l *worldListener) Close() error {
	l.once.Do(func() {
		close(l.done)
		l.release()
	})
	return nil
}

func (l *worldListener) Port() int { return l.port }

var errEngineDown = errors.New("this Mac's engine does not run in the world")

// invariant: ssh's bytes reach the VM of this Mac that runs at ip and whose
// guest answers: its sshd greets with its name; any other address is no
// sshd. A world never dials the real network.
func (s worldSockets) Pipe(_ context.Context, ip string, port int, _ io.Reader, out io.Writer) error {
	w := s.m.run.w
	w.Lock()
	defer w.Unlock()
	for _, name := range slices.Sorted(maps.Keys(w.Mac(s.m.name).VMs)) {
		v := w.Mac(s.m.name).VMs[name]
		at := cmp.Or(v.Address, "192.168.64.5")
		if v.Running() && v.Guest != nil && at == ip {
			_, err := fmt.Fprintf(out, "SSH-2.0-world %s %s:%d\r\n", name, ip, port)
			return err
		}
	}
	return errNoSSHD
}

var errNoSSHD = errors.New("no sshd answers at this address in the world")

// invariant: inside a VM of the world, the socket its Mac's engine offers
// there is that engine, as the guest relays it: its guest's calls are
// answered as the guest's, judged by the VM's rights.
func (worldSockets) DialTCP(context.Context, string, time.Duration) (net.Conn, error) {
	return nil, errors.New("the world dials no TCP")
}

func (s worldSockets) DialUnix(ctx context.Context, _ string, _ time.Duration) (local.Conn, error) {
	m, guest := s.m, ""
	if p, inside := m.placement(); inside {
		m, guest = m.run.mac(p.Mac), p.VM
	}
	d, ok := m.up(ctx)
	if !ok {
		return nil, errEngineDown
	}
	near, far := net.Pipe()
	go d.ServeGuest(context.WithoutCancel(ctx), far, guest)
	return near, nil
}

func (worldSockets) ListenUnix(string) (local.Listener, error) {
	return nil, errors.New("a world's engine is stepped, never listening")
}

type lockedNotifier struct {
	w *world.World
	n process.Notifier
}

func (l lockedNotifier) Notify(ctx context.Context, n engine.Notification) {
	l.w.Lock()
	defer l.w.Unlock()
	l.n.Notify(ctx, n)
}

// invariant: a Mac of the world runs nothing as root: a command that asks
// sudo, even inside a shell line, is refused and kept in the world.
type rootless struct {
	local.Runner
	w *world.World
}

var errRoot = errors.New("this world runs nothing as root")

func (r rootless) refuses(c local.Cmd) bool {
	line := strings.Join(append([]string{c.Name}, c.Args...), " ")
	if !slices.Contains(strings.FieldsFunc(line, func(r rune) bool {
		return r == ' ' || r == ';' || r == '&' || r == '|' || r == '(' || r == '\'' || r == '"'
	}), catalog.Sudo) {
		return false
	}
	r.w.Lock()
	r.w.AsRoot = append(r.w.AsRoot, line)
	r.w.Unlock()
	return true
}

func (r rootless) Output(ctx context.Context, c local.Cmd) (local.Output, error) {
	if r.refuses(c) {
		return local.Output{ExitCode: 1, Stderr: errRoot.Error()}, errRoot
	}
	return r.Runner.Output(ctx, c)
}

func (r rootless) Attach(ctx context.Context, c local.Cmd, out local.Stream) (local.Attached, error) {
	if r.refuses(c) {
		return local.Attached{}, errRoot
	}
	return r.Runner.Attach(ctx, c, out)
}

func (r rootless) Start(c local.Cmd, logPath string) (int, error) {
	if r.refuses(c) {
		return 0, errRoot
	}
	return r.Runner.Start(c, logPath)
}

// invariant: studio inside a VM of the world keeps its credentials in the
// VM's own slice: the line station runs to import reads them from stdin,
// the one to delete takes their names; the world's guest answers both, a
// scripted Guest.Exec fault breaks either, and that studio is there.
func (g worldGuest) credentials(cmd contract.GuestCommand) (contract.GuestResult, bool, error) {
	store := accountsfake.CredentialsOf(g.r.w, g.vm)
	switch {
	case cmd.Line == "command -v studio":
		return contract.GuestResult{Stdout: "/usr/local/bin/studio\n"}, true, nil
	case cmd.Line == accounts.VMStudio("import"):
		g.r.w.Lock()
		broke := g.r.w.Fails("Guest.Exec")
		g.r.w.Unlock()
		if broke {
			return contract.GuestResult{}, true, errWorldGuestBroke
		}
		var in []byte
		if cmd.Stdin != nil {
			var err error
			if in, err = io.ReadAll(cmd.Stdin); err != nil {
				return contract.GuestResult{}, true, err
			}
		}
		if !store.Import(in) {
			return contract.GuestResult{Code: 1, Stderr: "studio: the import is not a JSON object of names\n"}, true, nil
		}
		return contract.GuestResult{}, true, nil
	case strings.HasPrefix(cmd.Line, accounts.VMStudio("delete")+" "):
		g.r.w.Lock()
		broke := g.r.w.Fails("Guest.Exec")
		g.r.w.Unlock()
		if broke {
			return contract.GuestResult{Code: 1, Stderr: "studio: the credential store is not writable\n"}, true, nil
		}
		store.Delete(strings.Fields(strings.TrimPrefix(cmd.Line, accounts.VMStudio("delete"))))
		return contract.GuestResult{}, true, nil
	}
	return contract.GuestResult{}, false, nil
}

var errWorldGuestBroke = errors.New("the guest connection broke, as the world scripted it")
