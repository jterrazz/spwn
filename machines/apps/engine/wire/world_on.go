//go:build world

package wire

import (
	"bytes"
	"cmp"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"io"
	"io/fs"
	"maps"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/metrics"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/secret"

	"github.com/spf13/cobra"

	accountsfake "github.com/jterrazz/studio-station/apps/engine/accounts/adapter/fake"
	accountscontract "github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	activityfake "github.com/jterrazz/studio-station/apps/engine/activity/adapter/fake"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	agentsfake "github.com/jterrazz/studio-station/apps/engine/agents/adapter/fake"
	agentscontract "github.com/jterrazz/studio-station/apps/engine/agents/contract"
	appsfake "github.com/jterrazz/studio-station/apps/engine/apps/adapter/fake"
	appscontract "github.com/jterrazz/studio-station/apps/engine/apps/contract"
	automationsfake "github.com/jterrazz/studio-station/apps/engine/automations/adapter/fake"
	automationscontract "github.com/jterrazz/studio-station/apps/engine/automations/contract"
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
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	devicesfake "github.com/jterrazz/studio-station/apps/engine/devices/adapter/fake"
	devicescontract "github.com/jterrazz/studio-station/apps/engine/devices/contract"
	diskfake "github.com/jterrazz/studio-station/apps/engine/disk/adapter/fake"
	diskcontract "github.com/jterrazz/studio-station/apps/engine/disk/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	enginefake "github.com/jterrazz/studio-station/apps/engine/engine/adapter/fake"
	enginecontract "github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	imagesfake "github.com/jterrazz/studio-station/apps/engine/images/adapter/fake"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	journalfake "github.com/jterrazz/studio-station/apps/engine/journal/adapter/fake"
	machinefilefake "github.com/jterrazz/studio-station/apps/engine/machinefile/adapter/fake"
	machinefilecalls "github.com/jterrazz/studio-station/apps/engine/machinefile/calls"
	machinefilecontract "github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
	machinesfake "github.com/jterrazz/studio-station/apps/engine/machines/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	portsfake "github.com/jterrazz/studio-station/apps/engine/ports/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/ports/adapter/funnel"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
	projectfake "github.com/jterrazz/studio-station/apps/engine/project/adapter/fake"
	projectcontract "github.com/jterrazz/studio-station/apps/engine/project/contract"
	remotefake "github.com/jterrazz/studio-station/apps/engine/remote/adapter/fake"
	"github.com/jterrazz/studio-station/apps/engine/remote/adapter/unix"
	remotecontract "github.com/jterrazz/studio-station/apps/engine/remote/contract"
	"github.com/jterrazz/studio-station/apps/engine/remote/serve"
	settingsfake "github.com/jterrazz/studio-station/apps/engine/settings/adapter/fake"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
	skillsfake "github.com/jterrazz/studio-station/apps/engine/skills/adapter/fake"
	skillscontract "github.com/jterrazz/studio-station/apps/engine/skills/contract"
	"github.com/jterrazz/studio-station/apps/engine/surface/cli/kit"
	teamsfake "github.com/jterrazz/studio-station/apps/engine/teams/adapter/fake"
	teamscontract "github.com/jterrazz/studio-station/apps/engine/teams/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: how many quiet ticks end a stepped engine's settling, and the most
// ticks it may take: a world never waits on a clock of its own.
const (
	quietTicks = 3
	mostTicks  = 120
	// value: the quiet readings in a row that let go of a waiter still out.
	letGo = 4
	// value: the busy readings a leap takes a turn of the scheduler after,
	// before it sleeps between readings, from the first pause to the longest.
	spinReadings = 8
	firstPause   = 20 * time.Microsecond
	longestPause = time.Millisecond
	// value: the real time a settling waits, in all, for its guests' greetings.
	greetBound = 2 * time.Second
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
	// value: where this Mac's engine log is, as its paths say.
	logPath string

	// invariant: serving guards start and served: a far Mac is served by the
	// first caller that needs it, from any of the engines' goroutines.
	serving sync.Mutex
	start   func() *process.Engine
	// invariant: mu guards starting, engine, began, tick, leave and home: an
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
	vmOnce sync.Once
	vm     *farMac
	// value: the run's home, where the world's ~/ files are.
	home string
	// value: ends the surfaces the world holds open on the running engine.
	leave func()
	// value: the readings its engine ran beside its loop during the run,
	// left for the run's close.
	besides []func()
}

// invariant: a run that only looks (--page) reads the
// world under the lock and lets go of it at once: it writes nothing back,
// so a slow look never holds a call up, nor undoes one.
func openWorld(vars map[string]string, look bool) (*worldRun, error) {
	path := vars["STATION_WORLD"]
	if path == "" {
		armTripwire(nil)
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
	armTripwire(w)
	if look {
		unlockWorld(held)
		return &worldRun{path: path, w: w, macs: map[string]*worldMac{}, shared: true}, nil
	}
	r := &worldRun{path: path, w: w, macs: map[string]*worldMac{}, held: held}
	w.OnPause(r.pause)
	w.OnUnhold(r.unheld)
	return r, nil
}

// value: the spec build's tripwire is armed: every frame, CLI write and
// sealed page that carries a secret of the world, or a token shaped by its
// issuer, panics, so every story is a leak check.
const tripwireArmed = true

func armTripwire(w *world.World) {
	if !tripwireArmed {
		return
	}
	var secrets []string
	if w != nil {
		for _, m := range w.Macs {
			for _, v := range m.Vault {
				secrets = append(secrets, string(v))
			}
		}
	}
	secret.Arm(secrets, func(said string) { panic("tripwire: " + said) })
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

// invariant: a world unhold is another command's, so a held call learns of
// it from the world's file: it reads the file until its hold is gone from
// the script, a file caught half written read again at the next look.
func (r *worldRun) unheld(ctx context.Context, f world.Fault) <-chan struct{} {
	gone := make(chan struct{})
	go func() {
		for {
			var file struct {
				Script []world.Fault `json:"script"`
			}
			if data, err := os.ReadFile(r.path); err == nil && json.Unmarshal(data, &file) == nil && !slices.Contains(file.Script, f) {
				close(gone)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	}()
	return gone
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
	m.mu.Lock()
	m.home = h.Env.Home()
	m.mu.Unlock()
	h.FS = worldFS{FS: appsfake.SeenFS{FS: h.FS, Foot: foot, Home: h.Env.Home()}, m: m, plist: catalog.EnginePlist(h.Env.Home()), files: m.files()}
	h.Env = worldEnv{Env: h.Env, thisMac: m.name, user: m.run.w.User, admin: m.run.w.Admin && m.name == m.run.w.ThisMac || m.slice().Root}
	h.Clock = m.run.clock()
	h.Sockets = worldSockets{Sockets: h.Sockets, m: m}
	return h
}

func (m *worldMac) logAt(state string) { m.logPath = engine.LogPath(state) }

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
	st.prefs = corefake.PreferencesOver(slice.Preferences, m.name, m.run.w).Over(slice.Files)
	m.run.w.Unlock()
	w := m.run.w
	st.machines = machinesfake.StoreOf(w, m.name)
	st.images = imagesfake.StoreOf(w, m.name)
	st.marks = imagesfake.MarksOf(w, m.name)
	st.reservations = portsfake.ReservationsOf(w, m.name)
	st.declarations = portsfake.DeclarationsOf(w, m.name)
	st.applied = settingsfake.AppliedOf(w, m.name)
	st.uptime = machinesfake.UptimeOf(w, m.name)
	st.ledger = machinefilefake.LedgerOf(w, m.name)
	st.release = enginefake.ReleaseCacheOf(w, m.name)
	st.journal = journalfake.Of(w, m.name)
	st.devices = devicesfake.StoreOf(w, m.name)
}

func (m *worldMac) releases() enginecontract.Releases { return enginefake.ReleasesOf(m.run.w) }

func (m *worldMac) self() corecontract.Self { return corefake.SelfOf(m.run.w, m.name) }

func (m *worldMac) withAwake(kept map[string]activity.Kept) map[string]activity.Kept {
	// invariant: This Mac's kept reading carries the thermal and battery its
	// reading says, as a live sample composes them; a VM's are its Mac's.
	own, ok := kept[m.name]
	if !ok || m.power() == nil {
		return kept
	}
	out := maps.Clone(kept)
	activitycalls.ComposeAwake(&own.Last.Sample, m.power())
	out[m.name] = own
	return out
}

func (m *worldMac) power() awakecontract.Power { return awakefake.PowerOf(m.run.w, m.name) }
func (m *worldMac) localNetwork() func(context.Context) model.LocalNetwork {
	return func(context.Context) model.LocalNetwork { return m.slice().LocalNetwork }
}

func (m *worldMac) agent() func(context.Context) (func(context.Context) error, error) {
	return func(context.Context) (func(context.Context) error, error) {
		if held := m.slice().AgentHeld; held != nil {
			holder := model.Agent{PID: held.PID, Executable: held.Executable, Free: agentFree(held.PID, held.Label)}
			return nil, &engine.AgentHeldError{Holder: holder, Freed: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			}}
		}
		return func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		}, nil
	}
}

func (m *worldMac) guests() contract.GuestDialer {
	return worldGuests{GuestDialer: machinesfake.GuestsOf(m.run.w).Serving(engine.GuestSession), r: m.run}
}
func (m *worldMac) imageSource(hypervisor string) imagescontract.ImageSource {
	return imagesfake.SourceOf(m.run.w, hypervisor)
}
func (m *worldMac) sshConfig() contract.SSHConfig { return machinesfake.SSHConfigOf(m.run.w, m.name) }
func (m *worldMac) deviceKey() contract.DeviceKey { return machinesfake.DeviceKeyFor(m.name) }

// invariant: outside a world the spec build never reaches a Secure Enclave:
// its device key is the one kept in the file STATION_DEVICE_KEY_STUB names,
// made there at its first ask, so a restarted engine serves the key it
// served before; without that file, the Mac has no device key.
func specDeviceKey(vars map[string]string) contract.DeviceKey {
	file := vars["STATION_DEVICE_KEY_STUB"]
	if !filepath.IsAbs(file) {
		return &machinesfake.DeviceKey{Absent: true}
	}
	if data, err := os.ReadFile(file); err == nil {
		if block, _ := pem.Decode(data); block != nil {
			if key, parseErr := x509.ParseECPrivateKey(block.Bytes); parseErr == nil {
				return machinesfake.DeviceKeyOf(key)
			}
		}
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return &machinesfake.DeviceKey{Absent: true}
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return &machinesfake.DeviceKey{Absent: true}
	}
	if err := os.WriteFile(file, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		return &machinesfake.DeviceKey{Absent: true}
	}
	return machinesfake.DeviceKeyOf(key)
}

// value: the root the spec build reads `security` under, the sandbox's;
// the release build has none.
func securityRoot(vars map[string]string) string { return systemRootOf(vars) }

// invariant: outside a world the spec build never reaches a keychain: the
// device book's head is kept in the file STATION_DEVICE_ANCHOR_STUB names,
// "<generation> <digest>"; without that variable the real keychain's
// adapter runs, over whatever `security` the document puts on PATH.
func specDeviceAnchor(vars map[string]string) devicescontract.Anchor {
	file := vars["STATION_DEVICE_ANCHOR_STUB"]
	if !filepath.IsAbs(file) {
		return nil
	}
	return fileAnchor(file)
}

type fileAnchor string

func (f fileAnchor) Read(context.Context) (devices.Head, error) {
	data, err := os.ReadFile(string(f))
	if errors.Is(err, fs.ErrNotExist) {
		return devices.Head{}, nil
	}
	if err != nil {
		return devices.Head{}, err
	}
	return devices.ParseHead(string(data))
}

func (f fileAnchor) Write(_ context.Context, h devices.Head) error {
	return os.WriteFile(string(f), []byte(h.String()+"\n"), 0o600)
}

func (m *worldMac) unlocker() contract.Unlocker { return machinesfake.MacsOf(m.run.w) }
func (m *worldMac) waker() contract.Waker       { return machinesfake.WakerOf(m.run.w, m.name) }
func (m *worldMac) processes(kernel contract.Processes) contract.Processes {
	return machinesfake.ProcessesOf(m.run.w, m.name, kernel)
}
func (m *worldMac) minds() agentscontract.Minds        { return agentsfake.Of(m.run.w, m.name) }
func (m *worldMac) skillFiles() skillscontract.Files   { return skillsfake.Of(m.run.w, m.name) }
func (m *worldMac) projects() projectcontract.Projects { return projectfake.Of(m.run.w, m.name) }
func (m *worldMac) teams() teamscontract.Teams         { return teamsfake.Of(m.run.w, m.name) }
func (m *worldMac) automations() automationscontract.Automations {
	return automationsfake.Of(m.run.w, m.name)
}
func (m *worldMac) jobs() backgroundcontract.Jobs { return backgroundfake.New(m.run.w, m.name) }
func (m *worldMac) meter() activitycontract.Meter { return activityfake.MeterOf(m.run.w, m.name) }
func (m *worldMac) disk() diskcontract.DiskUsage  { return m.files() }

func (m *worldMac) files() *diskfake.Usage {
	m.mu.Lock()
	home := m.home
	m.mu.Unlock()
	return diskfake.UsageOf(m.run.w, m.name).At(home)
}
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
	m.run.w.Lock()
	m.run.w.Managers = named
	m.run.w.Unlock()
}
func (m *worldMac) vault() accountscontract.Vault { return accountsfake.NewVault(m.run.w, m.name) }
func (m *worldMac) signIns(now func() time.Time, sleep func(context.Context, time.Duration) error) accountscontract.SignIns {
	return accountsfake.NewSignIns(m.run.w, m.name, accountsfake.Clock{Now: now, Sleep: sleep})
}
func (m *worldMac) passwords(now func() time.Time, sleep func(context.Context, time.Duration) error) accountscontract.Passwords {
	return accountsfake.NewPasswords(m.run.w, m.name, accountsfake.Clock{Now: now, Sleep: sleep})
}
func (m *worldMac) forgetGrants(ctx context.Context, vm string) error {
	return accountsfake.NewGrants(m.run.w, m.name, accountsfake.Clock{}).Forget(ctx, vm)
}

func (m *worldMac) grants(now func() time.Time, sleep func(context.Context, time.Duration) error) accountscontract.Grants {
	return accountsfake.NewGrants(m.run.w, m.name, accountsfake.Clock{Now: now, Sleep: sleep})
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

func (m *worldMac) engineLacks(ability string) bool {
	e := m.slice().Engine
	return e != nil && slices.Contains(e.Lacks, ability)
}

func (m *worldMac) stationVersion() string { return m.slice().Station }

// invariant: in a world only This Mac's engine holds the other Macs'
// streams: another Mac's engine runs only to answer This Mac's command, and
// what it would copy is read by no one, while its streams would wait on the
// very start that asks it.
func (m *worldMac) holdsFar() bool { return m.name == m.run.w.ThisMac }

func (m *worldMac) absent(program string) bool { return slices.Contains(m.slice().Absent, program) }

// invariant: a reading the engine runs beside its loop is, in a world, run
// once the command is done, before its engine ends and keeps it: the next
// command reads it. It runs in the close's own flow, one after the other in
// the order they were asked, so the clock moves under it only as what it
// waits on moves it, never by real time; then the clock is back where it
// was: it takes none of the story's time, as the real one runs beside the
// command, never in its way.
func (m *worldMac) beside(f func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.besides = append(m.besides, f)
}

func (m *worldMac) aside() {
	m.mu.Lock()
	none := len(m.besides) == 0
	m.mu.Unlock()
	if none {
		return
	}
	w := m.run.w
	w.Lock()
	was := w.Now
	w.Unlock()
	for {
		m.mu.Lock()
		if len(m.besides) == 0 {
			m.mu.Unlock()
			break
		}
		f := m.besides[0]
		m.besides = m.besides[1:]
		m.mu.Unlock()
		f()
	}
	w.Lock()
	w.Now = was
	w.Unlock()
}

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

// invariant: the binary the engine started from is kept on the Mac's engine
// of the world, as the real one keeps it beside its socket.
func (m *worldMac) records() engine.Records { return worldRecords{m: m} }

type worldRecords struct{ m *worldMac }

func (w worldRecords) Write(r engine.Record) error {
	w.m.run.w.Lock()
	defer w.m.run.w.Unlock()
	if e := w.m.run.w.Mac(w.m.name).Engine; e != nil {
		e.Recorded = &r
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
	if text, ok := f.engineLog(p); ok {
		return plistInfo{name: filepath.Base(p), size: int64(len(text))}, nil
	}
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

// invariant: the lines the world gives a Mac's engine log are its file, read
// whole as one, whatever the run's own state folder holds; a log its disk
// counts with no line given is a file with no line, never one the run's
// folder is asked for.
func (f worldFS) engineLog(p string) (string, bool) {
	if p != f.m.logPath {
		return "", false
	}
	lines := f.m.slice().EngineLog
	if len(lines) == 0 {
		return "", f.files.Has(p)
	}
	return strings.Join(lines, "\n") + "\n", true
}

func (f worldFS) Open(p string) (fs.File, error) {
	text, ok := f.engineLog(p)
	if !ok {
		return f.FS.Open(p)
	}
	return &logFile{Reader: bytes.NewReader([]byte(text)), info: plistInfo{name: filepath.Base(p), size: int64(len(text))}}, nil
}

type logFile struct {
	*bytes.Reader
	info plistInfo
}

func (f *logFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (*logFile) Close() error                 { return nil }

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
	size int64
}

func (i plistInfo) Name() string { return i.name }
func (i plistInfo) Size() int64  { return i.size }
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
			d, ok := m.begin(context.WithoutCancel(ctx), s, start)
			if ok && ctx.Value(waits{}) != nil {
				greeted(d)
			}
			return d, ok
		}
		if s == nil {
			continue
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
		d.History().Take(m.withAwake(kept))
	}
	// invariant: when it began is set before the engine is, so a surface of
	// that engine that asks for it meanwhile never reads it as a successor.
	m.hold(d, m.engineStarted())
	d.RememberHeld(m.slice().Engine.Held)
	m.surfaces(ctx, d)
	// invariant: a start begun in an engine's lifetime (waits) leaves the
	// settling to the command: that command waits on this very start at its
	// end (close, up), and is the one that moves the clock a settle waits on.
	if ctx.Value(waits{}) == nil {
		m.settle(ctx)
	}
	return d, true
}

// invariant: an engine begun in another engine's lifetime is read by that
// engine at once, with no settling of its own: it is handed over once every
// guest greeting it has said what it says at the start, as an engine the
// world says runs has long said it, so what the reader reads of its VMs'
// guests is the same however loaded the gate. A greeting lands on no clock;
// one that waits on the clock all the same is waited for greetBound of real
// time.
func greeted(d *process.Engine) {
	bound := time.NewTimer(greetBound)
	defer bound.Stop()
	select {
	case <-d.Greeted():
	case <-bound.C:
	}
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
			return d.Stream(ctx, engine.PageWatchIn{Pages: []string{engine.PageMenubar}, Revision: page.Revision}, send)
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

// invariant: a start begins only while the Mac holds no engine, judged under
// the lock that holds it: a start that ended since the caller looked left its
// engine, and the caller reads that one (nil, false), never begins a second.
func (m *worldMac) beginStarting() (*engineStart, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.engine != nil {
		return nil, false
	}
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

// invariant: a tick of the settling leaps the clock only once every guest
// greeting the engine has said what it says at the start: in a world a
// greeting waits on no clock, so it lands before the first leap however
// loaded the gate, and the ticks a start takes never count it.
//
// hazard: a greeting that waits on the clock all the same (a guest's reply
// asked for on the world's time) is waited for greetBound of real time in all,
// once per settling, never once per tick.
func (m *worldMac) settle(ctx context.Context) {
	greeting, stop := context.WithTimeout(context.WithoutCancel(ctx), greetBound)
	defer stop()
	quiet := 0
	for n := 0; n < mostTicks && quiet < quietTicks; n++ {
		tick := m.nextTick()
		if d := m.current(); d != nil {
			select {
			case <-d.Greeted():
			case <-greeting.Done():
			}
		}
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
	history := d.History().Hand()
	read := d.SettingsKept()
	held := d.Held()
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
		e.Awake, e.Exposure, e.Activity, e.Settings, e.Held = &kept, &exposure, history, read, held
		if forget || e.Stopped {
			e.Awake, e.Exposure, e.Activity, e.Settings, e.Held = nil, nil, nil, nil, nil
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
		m.served.Serve(m.name, remotefake.Mac{Registry: far.registry, Profile: far.profile, Version: cmp.Or(m.stationVersion(), Version()), Lacks: m.slice().Lacks, Word: func(err error) *fail.Error { return kit.Translate(err, "") }, Relay: far.relay, Kept: far.kept})
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
			m.aside()
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

// invariant: a watch that follows a VM's create, as the window holds one
// open, reads the world as it moves and never moves it: a create the seed
// holds mid-way keeps the clock where the story left it however long the
// watch stays open.
func (*worldMac) watchContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, waits{}, true)
}

// invariant: a leap is a handshake with what runs. It waits until nothing
// but leapers can run, so what already runs takes its turn; it moves the time
// and wakes the waiters parked on it; it goes on once each is back, parked on
// the clock again or, its wait over, at its goroutine's next call on the
// clock, and nothing but leapers can run. No real time decides what a leap
// sees.
type moves struct {
	mu sync.Mutex
	ch chan struct{}
	// value: the leaps so far, and the waiters parked since the last one.
	gen    int
	parked int
	// value: the waiters a leap woke that are not back on the clock, and
	// those of them whose wait is over, by goroutine, back at their next call.
	out  int
	held map[int64]bool
	// value: the leapers waiting on the handshake.
	leapers atomic.Int32
	// invariant: while a Mac's surfaces are being let go, a wait on the clock
	// does not wait for it to move: the flow that moves it waits on them.
	leaving atomic.Int32
}

func (r *worldRun) clock() worldClock {
	if r.moved == nil {
		r.moved = &moves{ch: make(chan struct{}), held: map[int64]bool{}}
	}
	return worldClock{w: r.w, moved: r.moved}
}

// value: one wait on the clock: the leap it parked after, if parked.
type sleeper struct {
	gen    int
	parked bool
}

// invariant: a waiter whose wait is over keeps its turn when a leap woke it,
// and parks again otherwise; the world lock is taken under mu, never mu
// under it.
func (m *moves) park(c worldClock, until time.Time, s *sleeper) (<-chan struct{}, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	woken := m.unpark(s)
	if !c.Now().Before(until) {
		if woken {
			m.held[goroutine()] = true
		}
		return nil, true
	}
	m.back(woken)
	m.parked++
	s.gen, s.parked = m.gen, true
	return m.ch, false
}

func (m *moves) drop(s *sleeper) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.back(m.unpark(s))
}

func (m *moves) unpark(s *sleeper) bool {
	if !s.parked {
		return false
	}
	s.parked = false
	if s.gen == m.gen {
		m.parked--
		return false
	}
	return true
}

func (m *moves) back(woken bool) {
	if woken {
		m.out--
	}
}

// invariant: every wait the clock offers begins here: a goroutine that waits
// again, on the clock's time or on what the clock bounds, gives back the turn
// it kept.
func (m *moves) enter() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.held) == 0 {
		return
	}
	if id := goroutine(); m.held[id] {
		delete(m.held, id)
		m.out--
	}
}

func (m *moves) leave() func() {
	m.leaving.Add(1)
	m.tell()
	return func() { m.leaving.Add(-1) }
}

func (m *moves) tell() {
	m.mu.Lock()
	m.out += m.parked
	m.parked = 0
	m.gen++
	close(m.ch)
	m.ch = make(chan struct{})
	m.mu.Unlock()
}

func (m *moves) settled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.out <= 0
}

// invariant: a leap goes on once every waiter it woke is back and nothing but
// leapers can run. While a turn is out, quiet must hold for letGo readings in
// a row, a turn of the scheduler between: the turn is then a goroutine that
// waits on something other than the clock, or has ended, and is let go.
//
// invariant: a leap that reads something else running gives the CPU back
// past spinReadings such readings in a row, so what it waits for runs on it,
// however loaded the machine: the pause spaces the readings and never ends
// the wait, which only the readings do.
func (m *moves) await(ctx context.Context) {
	busy := 0
	for still := 0; ctx.Err() == nil; runtime.Gosched() {
		if !m.quiet() {
			still = 0
			busy++
			m.yield(busy)
			continue
		}
		busy = 0
		if m.settled() {
			return
		}
		still++
		if still == letGo {
			m.mu.Lock()
			m.out = 0
			clear(m.held)
			m.mu.Unlock()
			return
		}
	}
}

// invariant: a leap asleep between its readings is not counted among the
// leapers, so the others' readings never count it out of what runs; awake
// again before it counts, it reads as running, which only delays a leap.
func (m *moves) yield(busy int) {
	if busy <= spinReadings {
		return
	}
	m.leapers.Add(-1)
	time.Sleep(min(firstPause<<min(busy-spinReadings-1, 6), longestPause))
	m.leapers.Add(1)
}

// value: nothing but leapers can run, twice in a row with a turn of the
// scheduler between.
func (m *moves) quiet() bool {
	if !m.idle() {
		return false
	}
	runtime.Gosched()
	return m.idle()
}

func (m *moves) idle() bool {
	read := []metrics.Sample{{Name: "/sched/goroutines/running:goroutines"}, {Name: "/sched/goroutines/runnable:goroutines"}}
	metrics.Read(read)
	busy := read[0].Value.Uint64() + read[1].Value.Uint64()
	for n := m.leapers.Load(); n > 0 && busy > 0; n-- {
		busy--
	}
	return busy == 0
}

func goroutine() int64 {
	var buf [64]byte
	field := bytes.Fields(buf[:runtime.Stack(buf[:], false)])
	if len(field) < 2 {
		return 0
	}
	id, _ := strconv.ParseInt(string(field[1]), 10, 64)
	return id
}

func (c worldClock) Now() time.Time {
	c.w.Lock()
	defer c.w.Unlock()
	return c.w.Now
}

// invariant: what a wait waits for (a guest's greeting) lands on no clock, so
// it is waited for before the clock moves, however loaded the gate: one that
// is in moves the clock not at all.
//
// hazard: one that waits on the clock all the same is waited for greetBound
// of real time, then passes as a sleep of d.
func (c worldClock) Await(ctx context.Context, ready <-chan struct{}, d time.Duration) error {
	c.moved.enter()
	bound := time.NewTimer(greetBound)
	defer bound.Stop()
	select {
	case <-ready:
		return ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	case <-bound.C:
	}
	return c.Sleep(ctx, d)
}

// invariant: a bounded wait moves no clock and waits no real time for what
// only a move would bring: what busy may still bring (on no clock) is waited
// for, greetBound of real time at most, and what is not ready once it ends is
// judged at once, however loaded the gate.
func (c worldClock) AwaitWhile(ctx context.Context, ready, busy <-chan struct{}, _ time.Duration) error {
	c.moved.enter()
	if busy != nil {
		bound := time.NewTimer(greetBound)
		defer bound.Stop()
		select {
		case <-ready:
		case <-busy:
		case <-ctx.Done():
		case <-bound.C:
		}
	}
	return ctx.Err()
}

func (c worldClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.moved.enter()
	if ctx.Value(waits{}) == nil {
		c.moved.leapers.Add(1)
		defer c.moved.leapers.Add(-1)
		c.moved.await(ctx)
		c.w.Lock()
		c.w.Now = c.w.Now.Add(d)
		c.w.Unlock()
		c.moved.tell()
		c.moved.await(ctx)
		return nil
	}
	until := c.Now().Add(d)
	var s sleeper
	for {
		moved, over := c.moved.park(c, until, &s)
		if over {
			return nil
		}
		if c.moved.leaving.Load() > 0 {
			c.moved.drop(&s)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Millisecond):
				return nil
			}
		}
		select {
		case <-ctx.Done():
			c.moved.drop(&s)
			return ctx.Err()
		case <-moved:
		}
	}
}

// invariant: a VM of the world whose own slice says what it is answers
// station's API as `station guest serve --stdio` answers inside it: its calls
// run over that slice, each exchange from no shared reading, as each is a
// process there, every value through the wire form; any other line,
// and that one while a Guest.Serve fault is scripted, is the scripted guest's.
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
	if err := g.hostKey(t); err != nil {
		return nil, err
	}
	c, err := g.GuestDialer.Dial(ctx, t)
	if err != nil {
		return nil, err
	}
	return worldGuest{Guest: c, vm: t.VM, r: g.r}, nil
}

// invariant: in a world whose pulled images ship their publisher's login, a
// guest serves the host key its last lockdown made (the clone's own before
// any), checked as ssh checks it, accept-new against the VM's known hosts:
// an absent entry is appended, a different key refuses.
func (g worldGuests) hostKey(t contract.GuestTarget) error {
	g.r.w.Lock()
	upstream := g.r.w.UpstreamLogin != nil
	mac := g.r.w.Holder(t.VM)
	var guest *world.Guest
	if mac != "" {
		guest = g.r.w.Macs[mac].VMs[t.VM].Guest
	}
	n := 0
	if guest != nil {
		n = guest.HostKeys
	}
	g.r.w.Unlock()
	if !upstream || guest == nil || t.KnownHosts == "" {
		return nil
	}
	served := strings.Fields(machinesfake.HostKey(t.VM, n))
	known, err := os.ReadFile(t.KnownHosts)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	for line := range strings.Lines(string(known)) {
		f := strings.Fields(line)
		if len(f) < 3 || f[0] != t.VM {
			continue
		}
		if f[1] == served[0] && f[2] == served[1] {
			return nil
		}
		return fmt.Errorf("%w: ssh: handshake failed: knownhosts: key mismatch", contract.ErrHostKeyChanged)
	}
	if err := os.MkdirAll(filepath.Dir(t.KnownHosts), 0o700); err != nil {
		return err
	}
	return os.WriteFile(t.KnownHosts, append(known, t.VM+" "+served[0]+" "+served[1]+"\n"...), 0o600)
}

func (g worldGuest) Exec(ctx context.Context, cmd contract.GuestCommand) (contract.GuestResult, error) {
	// invariant: a guest is another process: what the Mac's context carries
	// (the steps its act records) never reaches it, only its end does, and
	// the world's clock, which is one for every process of the world: the
	// guest of a run marked as waiting on it (waits, pooled) waits on it as
	// that run would, never leaping it.
	asked := ctx
	ctx = context.Background()
	if asked.Value(waits{}) != nil {
		ctx = context.WithValue(ctx, waits{}, true)
	}
	if asked.Value(pooled{}) != nil {
		ctx = context.WithValue(ctx, pooled{}, true)
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	defer context.AfterFunc(asked, stop)()
	g.r.w.Lock()
	slice, ok := g.r.w.Macs[g.vm]
	ok = ok && slice.Self != nil
	bake := ""
	if !ok {
		slice, bake, ok = madeSlice(g.r.w, g.vm)
	}
	if bake != "" && g.r.far != nil {
		g.r.w.Unlock()
		if err := g.bakeSlice(ctx, bake); err != nil {
			return contract.GuestResult{}, err
		}
		g.r.w.Lock()
	}
	ok = ok && (cmd.Line != unix.Guest || !g.r.w.Fails("Guest.Serve"))
	if ok && cmd.Line == "bash -s" && cmd.Stdin != nil {
		script, err := io.ReadAll(cmd.Stdin)
		if err != nil {
			g.r.w.Unlock()
			return contract.GuestResult{}, err
		}
		copied(slice, script)
		cmd.Stdin = bytes.NewReader(script)
	}
	g.r.w.Unlock()
	if cmd.Line != unix.Guest || !ok || g.r.far == nil || cmd.Stdin == nil {
		return g.Guest.Exec(ctx, cmd)
	}
	far := g.r.mac(g.vm).inVM()
	far.fresh()
	var out bytes.Buffer
	w := cmd.Stdout
	if w == nil {
		w = &out
	}
	g.r.w.Lock()
	version, lacks := cmp.Or(slice.Station, Version()), slices.Clone(slice.Lacks)
	g.r.w.Unlock()
	err := serve.Stream(ctx, cmd.Stdin, w, lacking{Calls: serve.Local(far.registry, far.profile, version), lacks: lacks}, func(err error) *fail.Error { return kit.Translate(err, "") })
	return contract.GuestResult{Stdout: out.String()}, err
}

// invariant: a VM of the world whose slice lacks calls or abilities runs
// the station of an image built before them: its guest's hello lists
// neither, as that older binary's does.
type lacking struct {
	serve.Calls
	lacks []string
}

func (l lacking) Hello() remotecontract.Hello {
	h := l.Calls.Hello()
	h.Calls = slices.DeleteFunc(h.Calls, func(c string) bool { return slices.Contains(l.lacks, c) })
	h.Abilities = slices.DeleteFunc(h.Abilities, func(a string) bool { return slices.Contains(l.lacks, a) })
	return h
}

// invariant: the guest of the world runs no shell, so each file a script
// writes into the VM user's station folder (the Machinefile and lock a
// create copies in), one line of it, lands in the VM's slice as that line
// writes it in a real guest. The caller holds the world's lock.
func copied(slice *world.Mac, script []byte) {
	for line := range strings.Lines(string(script)) {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "printf %s ")
		if !ok {
			continue
		}
		encoded, at, ok := strings.Cut(rest, ` | base64 -d > "$HOME/.config/station/`)
		if !ok {
			continue
		}
		body, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			continue
		}
		if slice.Files == nil {
			slice.Files = map[string]string{}
		}
		slice.Files[strings.TrimSuffix(at, `"`)] = string(body)
	}
}

// value: the slice of a VM a create made in this world, made the first
// time its guest is asked: a Linux workspace or server, sized as its disk
// was, its user's sudo asking no password as station's image makes it,
// following the Machinefile it was made from, which its image was built
// from: the files of the Machinefiles its Mac held at that build
// (world.Mac.Built), or holds now when it built none, as the build copied
// them in, and the name of the one to bake (bakeSlice), empty for a VM
// made from no Machinefile; false for any other VM. The caller holds the
// world's lock.
func madeSlice(w *world.World, vm string) (*world.Mac, string, bool) {
	holder := w.Holder(vm)
	if holder == "" {
		return nil, "", false
	}
	v := w.Macs[holder].VMs[vm]
	if !v.Made || v.Kept == nil || v.Guest == nil || v.Guest.Replaced {
		return nil, "", false
	}
	slice := w.Mac(vm)
	slice.Self = &world.Self{Kind: v.Kept.Lifetime.Kind(), System: coremachine.System{OS: coremachine.Linux}, Hardware: coremachine.Hardware{Memory: v.Size.Memory, CPUs: v.Size.CPUs}}
	slice.Passwordless = true
	if v.Kept.Template == "" {
		return slice, "", true
	}
	if slice.Preferences == nil {
		named, err := json.Marshal(map[string]string{"name": v.Kept.Template})
		if err == nil {
			slice.Preferences = map[string]json.RawMessage{"machinefile": named}
		}
	}
	files, built := w.Macs[holder].Built[v.Kept.Template]
	if !built {
		files = w.Macs[holder].Files
	}
	for at, body := range files {
		if strings.HasPrefix(at, "machinefiles/") {
			if slice.Files == nil {
				slice.Files = map[string]string{}
			}
			slice.Files[at] = body
		}
	}
	return slice, v.Kept.Template, true
}

// invariant: a VM cloned from an image holds what the image's bake
// installed and set, as it was when the image was built: the image phase
// of its Machinefile is carried out in its slice once, as the build ran
// it, before anything else asks its guest, beyond the reach of a fault
// or a hold scripted since (world.Baked).
func (g worldGuest) bakeSlice(ctx context.Context, file string) error {
	far := g.r.mac(g.vm).inVM()
	far.fresh()
	_, err := far.registry.Run(world.Baked(context.WithoutCancel(ctx)), far.profile, machinefilecalls.Apply.Name, machinefilecalls.ApplyIn{Machinefile: file, Phase: catalog.PhaseImage}, nil)
	return err
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
	m.vmOnce.Do(func() {
		far := m.run.far(m)
		m.vm = &far
	})
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

func (m *worldMac) presence() devices.Answer {
	switch m.run.w.Mac(m.name).Presence {
	case "refused":
		return devices.Refused
	case "absent":
		return devices.Absent
	}
	return devices.Given
}
