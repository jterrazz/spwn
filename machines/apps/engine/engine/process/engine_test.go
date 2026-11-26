package process

import (
	"context"
	"errors"
	"net/netip"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/setup"

	enginecalls "github.com/jterrazz/studio-station/apps/engine/engine/calls"
)

type fakeMachines struct {
	mu      sync.Mutex
	stuck   map[string]bool
	silent  error
	records []model.VMRecord
	saved   []model.VMRecord
	saveErr error
	off     func(vm string)
	offed   []string
}

func (f *fakeMachines) Records() ([]model.VMRecord, error) { return f.records, nil }

func (f *fakeMachines) Save(r model.VMRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saved = append(f.saved, r)
	return nil
}

func (f *fakeMachines) Remove(context.Context, string) (model.DeleteReport, error) {
	return model.DeleteReport{}, nil
}

func (f *fakeMachines) PowerOff(_ context.Context, vm string) error {
	f.mu.Lock()
	f.offed = append(f.offed, vm)
	off := f.off
	f.mu.Unlock()
	if off != nil {
		off(vm)
	}
	return nil
}

func (f *fakeMachines) EngineRunning(_ context.Context, name string) (bool, error) {
	return f.stuck[name], f.silent
}

func (f *fakeMachines) MarkUptime(context.Context, model.UptimeMark) error { return nil }

func (f *fakeMachines) ThisMacName() string  { return "x" }
func (f *fakeMachines) ThisMacLabel() string { return "x" }

type fakeRun struct {
	vm    string
	pid   int
	ended chan model.RunEnd
}

func (r *fakeRun) VM() string { return r.vm }

func (r *fakeRun) PID() int { return r.pid }

func (r *fakeRun) Holder() int { return r.pid }

func (r *fakeRun) Ended() <-chan model.RunEnd { return r.ended }

type fakeHypervisor struct {
	mu    sync.Mutex
	boots int
	live  map[string]*fakeRun
}

func newHypervisor() *fakeHypervisor { return &fakeHypervisor{live: map[string]*fakeRun{}} }

func (h *fakeHypervisor) Boot(_ context.Context, vm string, _ model.Network, _ []model.Mount) (Run, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.boots++
	r := &fakeRun{vm: vm, pid: 1000 + h.boots, ended: make(chan model.RunEnd, 1)}
	h.live[vm] = r
	return r, nil
}

func (h *fakeHypervisor) Adopt(_ context.Context, vm string) (Run, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	r, ok := h.live[vm]
	if !ok {
		return nil, false
	}
	return r, true
}

// value: outlived starts a run of vm before the engine does, as one left by an earlier engine.
func (h *fakeHypervisor) outlived(vm string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.live[vm] = &fakeRun{vm: vm, pid: 4242, ended: make(chan model.RunEnd, 1)}
}

func (h *fakeHypervisor) end(vm string, end model.RunEnd) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.live[vm]; ok {
		delete(h.live, vm)
		r.ended <- end
	}
}

func (h *fakeHypervisor) started() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.boots
}

func (*fakeHypervisor) Shutdown(context.Context, string, time.Duration) error { return nil }

func (*fakeHypervisor) Console(context.Context, string, int) ([]string, error) { return nil, nil }

func (*fakeHypervisor) Address(context.Context, string, time.Duration) (netip.Addr, error) {
	return netip.Addr{}, errors.New("no ip in a module test")
}

type harness struct {
	d     *Engine
	hv    *fakeHypervisor
	mac   *fakeMac
	clock *stepClock
	m     *fakeMachines
}

func newHarness(t *testing.T, records ...model.VMRecord) harness {
	t.Helper()
	clock := &stepClock{At: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	h := &fakeMac{clock: clock}
	m := &fakeMachines{records: records}
	hv := newHypervisor()
	paths := setup.Paths{Prefs: "/c", State: "/s"}
	d := New(h, hv, nil, m, &fakeTailnet{}, memStores(), Options{State: paths.State, ThisMachine: catalog.Mac})
	return harness{d: d, hv: hv, mac: h, clock: clock, m: m}
}

func (h harness) runs() int { return h.hv.started() }

func (h harness) state(name string) string {
	v, err := h.d.find(name)
	if err != nil {
		return "unknown"
	}
	h.d.mu.Lock()
	defer h.d.mu.Unlock()
	return v.state
}

func cancelled(t *testing.T) context.Context {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}

func TestAtStartADeadRunLeavesEveryWorkspaceStoppedAndRunsNone(t *testing.T) {
	h := newHarness(t,
		model.VMRecord{Name: "t", Lifetime: model.Ephemeral},
		model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning},
		model.VMRecord{Name: "off", Lifetime: model.Persistent, Want: model.WantStopped},
	)
	h.d.adopt(cancelled(t))
	for _, vm := range []string{"t", "api", "off"} {
		if got := h.state(vm); got != engine.StateStopped {
			t.Errorf("%s with no live run: %s, want stopped", vm, got)
		}
	}
	if h.runs() != 0 {
		t.Errorf("%d runs started, want none: a workspace never restarts by itself", h.runs())
	}
}

func TestAtStartTheStopsOfSharingOfAVMWithNoRecordAreForgotten(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent}, model.VMRecord{Name: "off", Lifetime: model.Persistent})
	var kept [][]string
	h.d.o.ForgetStops = func(vms []string) error {
		kept = append(kept, vms)
		return nil
	}
	if err := h.d.Begin(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{"api", "off"}}; !reflect.DeepEqual(kept, want) {
		t.Errorf("the start kept the stops of %v, want %v", kept, want)
	}
}

func TestAtStartALiveRunIsAdoptedNeverRunAgain(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	h.hv.outlived("api")
	h.d.adopt(cancelled(t))
	if got := h.state("api"); got != engine.StateRunning || h.runs() != 0 {
		t.Errorf("adopted api: %s with %d runs, want running and none", got, h.runs())
	}
}

func TestAtStartAHypervisorThatDoesNotAnswerKeepsEachVMAsKnownAndBootsNone(t *testing.T) {
	h := newHarness(t,
		model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning},
		model.VMRecord{Name: "web", Lifetime: model.Persistent, Want: model.WantRunning},
	)
	h.m.silent = errors.New("the hypervisor does not answer")
	h.d.Remember(engine.Memory{"api": {State: engine.StateRunning}})
	h.d.adopt(cancelled(t))
	if api, web := h.state("api"), h.state("web"); api != engine.StateRunning || web != engine.StateStopped || h.runs() != 0 {
		t.Errorf("adopted with a silent hypervisor: api %s, web %s, %d runs; want api running as remembered, web as it was, none run", api, web, h.runs())
	}
}

func TestAStopOfAVMWithNoRunRefusesWhileItsHypervisorDoesNotAnswer(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	h.m.silent = errors.New("the hypervisor does not answer")
	v, _ := h.d.find("api")
	v.state = engine.StateRunning
	if err := h.d.endRun(t.Context(), v); !errors.Is(err, h.m.silent) {
		t.Errorf("a stop with a silent hypervisor answered %v; want its silence", err)
	}
	if got := h.state("api"); got != engine.StateRunning {
		t.Errorf("after the refused stop api is %s; want running as known", got)
	}
}

func TestAThirdRebootWithinAMinuteStopsTheVMAsALoop(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "t", Lifetime: model.Ephemeral})
	ctx := cancelled(t)
	v, _ := h.d.find("t")
	_ = h.d.start(ctx, v)
	for i := range 3 {
		h.hv.end("t", model.RunEnd{Reason: model.GuestReboot})
		h.clock.At = h.clock.At.Add(5 * time.Second)
		h.d.tick(ctx)
		if i < 2 && h.state("t") != engine.StateRunning {
			t.Fatalf("reboot %d: %s, want a re-run", i+1, h.state("t"))
		}
	}
	if got := h.state("t"); got != engine.StateBroken || h.runs() != 3 {
		t.Errorf("after 3 reboots in 15 s: %s with %d runs, want a loop stopped and no fourth run", got, h.runs())
	}
}

func TestARunWhoseGuestNeverConnectsAnswersGuestTimeoutAfterAMinute(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	h.hv.outlived("api")
	v, _ := h.d.find("api")
	v.run, _ = h.hv.Adopt(t.Context(), "api")
	v.state, v.guest = engine.StateRunning, engine.GuestConnecting
	began := h.clock.At
	err := h.d.awaitGuest(t.Context(), "api")
	var op *enginecalls.OpError
	if !errors.As(err, &op) || op.Code != "guest_timeout" {
		t.Fatalf("a guest that never connects: %v, want guest_timeout", err)
	}
	if waited := h.clock.At.Sub(began); waited < guestReady {
		t.Errorf("guest_timeout after %s, want %s", waited, guestReady)
	}
}

// value: a Mac whose waits for an event pass no time, as a world's do: it
// tells awaiting once a wait begins.
type eventMac struct {
	*fakeMac
	awaiting chan struct{}
}

func (m eventMac) Await(ctx context.Context, ready <-chan struct{}, _ time.Duration) error {
	close(m.awaiting)
	select {
	case <-ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestAGuestsGreetingIsWaitedForBeforeTheGuestIsJudgedSoItMovesNoPoll(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	mac := eventMac{fakeMac: h.mac, awaiting: make(chan struct{})}
	d := New(mac, h.hv, nil, h.m, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac})
	h.hv.outlived("api")
	v, _ := d.find("api")
	v.run, _ = h.hv.Adopt(t.Context(), "api")
	v.state, v.guest = engine.StateRunning, engine.GuestConnecting
	d.mu.Lock()
	d.greetLocked(v)
	d.mu.Unlock()
	go func() {
		<-mac.awaiting
		d.mu.Lock()
		v.guest = engine.GuestConnected
		d.greetedLocked(v)
		d.mu.Unlock()
	}()
	began := h.clock.At
	if err := d.awaitGuest(t.Context(), "api"); err != nil {
		t.Fatalf("a guest that greets and connects: %v, want it run", err)
	}
	if moved := h.clock.At.Sub(began); moved != 0 {
		t.Errorf("the clock moved %s while the guest greeted, want none", moved)
	}
}

// value: a Mac whose clock only its waiters move, as a world's: a bounded
// wait waits for the work in flight that may end it, never for time. It tells
// waiting once such a wait begins.
type stillMac struct {
	*fakeMac
	waiting chan struct{}
}

func (m stillMac) AwaitWhile(ctx context.Context, ready, busy <-chan struct{}, _ time.Duration) error {
	if busy == nil {
		return ctx.Err()
	}
	close(m.waiting)
	select {
	case <-ready:
	case <-busy:
	case <-ctx.Done():
	}
	return ctx.Err()
}

func dialingVM(t *testing.T) (*Engine, *vmState, stillMac) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	mac := stillMac{fakeMac: h.mac, waiting: make(chan struct{})}
	d := New(mac, h.hv, nil, h.m, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac})
	h.hv.outlived("api")
	v, _ := d.find("api")
	v.run, _ = h.hv.Adopt(t.Context(), "api")
	v.state, v.guest, v.up = engine.StateRunning, engine.GuestConnecting, make(chan struct{})
	return d, v, mac
}

func TestACallToAVMWhoseGuestDialsNoMoreIsAbsentOnTheClockNotAfterRealTime(t *testing.T) {
	d, _, _ := dialingVM(t)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_, err := d.GuestExec(ctx, "api", "true", "")
	if ctx.Err() != nil {
		t.Fatalf("a guest between two dials was judged after %s of real time, want at once", time.Second)
	}
	var op *enginecalls.OpError
	if !errors.As(err, &op) || op.Code != enginecalls.GuestAbsent {
		t.Fatalf("a guest between two dials: %v, want guest_absent", err)
	}
}

func TestACallThatLandsWhileTheGuestIsDialedWaitsForThatDial(t *testing.T) {
	d, v, mac := dialingVM(t)
	d.mu.Lock()
	d.greetLocked(v)
	d.mu.Unlock()
	go func() {
		<-mac.waiting
		d.mu.Lock()
		v.client = speaksFirst{}
		v.release()
		d.greetedLocked(v)
		d.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	_, err := d.GuestExec(ctx, "api", "true", "")
	var op *enginecalls.OpError
	if errors.As(err, &op) || err == nil {
		t.Fatalf("a call during a dial that lands: %v, want the guest's own answer", err)
	}
}

func TestStopAllAnswersFailedForALastingVMTheEngineStillRuns(t *testing.T) {
	h := newHarness(t,
		model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning},
		model.VMRecord{Name: "web", Lifetime: model.Persistent, Want: model.WantRunning},
	)
	h.m.stuck = map[string]bool{"api": true}
	for _, name := range []string{"api", "web"} {
		v, _ := h.d.find(name)
		v.state = engine.StateRunning
	}
	got := h.d.stopAll(t.Context())
	want := []engine.Stopped{
		{Name: "api", Lifetime: "persistent", Result: engine.ResultFailed, Detail: "the engine still runs it"},
		{Name: "web", Lifetime: "persistent", Result: engine.ResultStopped},
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("stop_all answered %+v, want %+v", got, want)
	}
}

func TestStopAllNeverAnswersStoppedForAVMItsHypervisorCannotSay(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	v, _ := h.d.find("api")
	if err := h.d.start(cancelled(t), v); err != nil {
		t.Fatal(err)
	}
	h.m.silent = errors.New("the hypervisor does not answer")
	got := h.d.stopAll(t.Context())
	if len(got) != 1 || got[0].Result != engine.ResultFailed || got[0].Detail != h.m.silent.Error() {
		t.Errorf("stop_all with a silent hypervisor answered %+v; want api failed with its silence", got)
	}
}

func TestAStopWhoseWishCannotBeSavedStopsTheVMAndSaysWhy(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "mac-1", Lifetime: model.Persistent, Want: model.WantRunning})
	v, _ := h.d.find("mac-1")
	if err := h.d.start(cancelled(t), v); err != nil {
		t.Fatal(err)
	}
	h.m.saveErr = errors.New("the store is read-only")
	if err := h.d.halt(t.Context(), v, true); !errors.Is(err, h.m.saveErr) {
		t.Errorf("halt answered %v; want the wish's save error", err)
	}
	if got := h.state("mac-1"); got != engine.StateStopped {
		t.Errorf("mac-1 reads %s after its stop; want it stopped though its wish was not saved", got)
	}
}

func TestAStartSaysWhyItsMountsAreNotItsMachinefilesAndKeepsWhatItShared(t *testing.T) {
	// invariant: a start whose folders are not its Machinefile's says why
	// in daemon.log and on the VM's detail line, and the record keeps the
	// folders the run shares, for a later start that cannot read them.
	h := newHarness(t, model.VMRecord{Name: "web", Lifetime: model.Persistent, Template: "studio-linux"})
	shared := []model.Mount{{Computer: "/Users/me/data", Guest: "/data"}}
	why := "web's Machinefile studio-linux could not be read: it shares the folders of its last start"
	h.d.o.Mounts = func(context.Context, string) (model.MountsRead, error) {
		return model.MountsRead{Mounts: shared, Why: why}, nil
	}
	var notes []string
	h.d.note = func(line string) { notes = append(notes, line) }
	v, _ := h.d.find("web")
	if err := h.d.start(cancelled(t), v); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(notes, func(n string) bool { return strings.HasSuffix(n, " web mounts: "+why) }) {
		t.Errorf("daemon.log: %q; want the line of why", notes)
	}
	h.d.mu.Lock()
	detail := v.detail
	h.d.mu.Unlock()
	if detail != why {
		t.Errorf("detail %q; want %q", detail, why)
	}
	if n := len(h.m.saved); n != 1 || !slices.Equal(h.m.saved[n-1].Mounts, shared) {
		t.Errorf("saved %v; want the record keeping %v", h.m.saved, shared)
	}
}

// value: a guest that answers every command with one exit code, or a line of
// replies with its own, and keeps the lines it ran.
type answering struct {
	speaksFirst
	code    int
	err     error
	replies map[string]reply
	lines   []string
	then    func()
}

type reply struct {
	code int
	err  error
}

func (g *answering) Exec(_ context.Context, line, _ string) (engine.GuestRan, error) {
	g.lines = append(g.lines, line)
	if g.then != nil {
		g.then()
	}
	if r, ok := g.replies[line]; ok {
		return engine.GuestRan{Code: r.code}, r.err
	}
	return engine.GuestRan{Code: g.code}, g.err
}

func stoppingVM(t *testing.T, kind catalog.MachineKind, code int) (harness, *answering) {
	t.Helper()
	h := newHarness(t, model.VMRecord{Name: "vm", Lifetime: model.Persistent, Want: model.WantRunning})
	h.d.o.KindOf = func(context.Context, string) catalog.MachineKind { return kind }
	v, _ := h.d.find("vm")
	if err := h.d.start(cancelled(t), v); err != nil {
		t.Fatal(err)
	}
	g := &answering{code: code}
	h.d.mu.Lock()
	v.client = g
	h.d.mu.Unlock()
	return h, g
}

func TestAMacOSVMIsPoweredOffByMacOSsOwnShutdown(t *testing.T) {
	h, g := stoppingVM(t, catalog.MacOS, 0)
	g.then = func() { h.hv.end("vm", model.RunEnd{}) }
	if err := h.d.opStop(t.Context(), "vm", false); err != nil {
		t.Fatal(err)
	}
	if want := []string{"sudo -n -l /sbin/shutdown -h now >/dev/null && (nohup sudo -n /sbin/shutdown -h now >/dev/null 2>&1 &)"}; !reflect.DeepEqual(g.lines, want) {
		t.Errorf("the guest ran %q, want %q", g.lines, want)
	}
}

func TestALinuxVMIsPoweredOffBySystemd(t *testing.T) {
	h, g := stoppingVM(t, catalog.Linux, 0)
	g.then = func() { h.hv.end("vm", model.RunEnd{}) }
	if err := h.d.opStop(t.Context(), "vm", false); err != nil {
		t.Fatal(err)
	}
	if want := []string{"sudo systemctl --no-block poweroff"}; !reflect.DeepEqual(g.lines, want) {
		t.Errorf("the guest ran %q, want %q", g.lines, want)
	}
}

// value: a Mac whose clock waits, once, for the hypervisor's power-off to have run, which the engine starts beside its wait.
type afterPoweroff struct {
	Mac
	done <-chan struct{}
}

func (m afterPoweroff) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case <-m.done:
	case <-ctx.Done():
	}
	return m.Mac.Sleep(ctx, d)
}

func TestAPoweroffTheGuestRefusesGoesStraightToTheHardStop(t *testing.T) {
	stoppedAtOnce(t, catalog.MacOS, 1, nil)
}

func TestALinuxGuestThatRefusesItsPoweroffAndIsNotGoingDownGoesStraightToTheHardStop(t *testing.T) {
	stoppedAtOnce(t, catalog.Linux, 1, nil)
}

func TestAConnectionThatDropsAfterARefusedLineGoesStraightToTheHardStop(t *testing.T) {
	stoppedAtOnce(t, catalog.MacOS, 1, errors.New("ssh: disconnect"))
}

func TestAConnectionThatDropsBeforeAnyAnswerGoesStraightToTheHardStop(t *testing.T) {
	stoppedAtOnce(t, catalog.MacOS, 0, errors.New("ssh: disconnect"))
}

func TestALinuxGuestAlreadyGoingDownThatRefusesItsPoweroffIsLeftToItsOwnShutdown(t *testing.T) {
	leftToItsShutdown(t, reply{code: 0})
}

func TestALinuxGuestThatRefusesItsPoweroffThenDropsTheConnectionIsLeftToItsOwnShutdown(t *testing.T) {
	leftToItsShutdown(t, reply{code: 255, err: errors.New("ssh: disconnect")})
}

func leftToItsShutdown(t *testing.T, asked reply) {
	t.Helper()
	h, g := stoppingVM(t, catalog.Linux, 0)
	g.replies = map[string]reply{
		catalog.PoweroffLine(catalog.Linux):  {code: 1},
		catalog.GoingDownLine(catalog.Linux): asked,
	}
	g.then = func() {
		if len(g.lines) == 2 {
			h.hv.end("vm", model.RunEnd{})
		}
	}
	if err := h.d.opStop(t.Context(), "vm", false); err != nil {
		t.Fatal(err)
	}
	if want := []string{catalog.PoweroffLine(catalog.Linux), catalog.GoingDownLine(catalog.Linux)}; !reflect.DeepEqual(g.lines, want) {
		t.Errorf("the guest ran %q, want %q", g.lines, want)
	}
	h.m.mu.Lock()
	defer h.m.mu.Unlock()
	if len(h.m.offed) != 0 {
		t.Errorf("the hypervisor powered off %v, a guest going down by itself", h.m.offed)
	}
}

func stoppedAtOnce(t *testing.T, kind catalog.MachineKind, code int, drop error) {
	t.Helper()
	h, g := stoppingVM(t, kind, code)
	g.err = drop
	done := make(chan struct{})
	h.d.h = afterPoweroff{Mac: h.d.h, done: done}
	h.m.off = func(vm string) {
		h.hv.end(vm, model.RunEnd{})
		close(done)
	}
	start := h.clock.At
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := h.d.opStop(ctx, "vm", false); err != nil {
		t.Fatal(err)
	}
	if got := h.clock.At.Sub(start); got >= stopWait {
		t.Errorf("the stop waited %s after a refused poweroff, want less than %s", got, stopWait)
	}
	h.m.mu.Lock()
	defer h.m.mu.Unlock()
	if !reflect.DeepEqual(h.m.offed, []string{"vm"}) {
		t.Errorf("the hypervisor powered off %v, want [vm]", h.m.offed)
	}
}
