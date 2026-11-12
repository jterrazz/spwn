package process

import (
	"context"
	"errors"
	"net/netip"
	"sync"
	"testing"
	"time"

	model "github.com/jterrazz/studio-station/src/machines"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/setup"

	enginecalls "github.com/jterrazz/studio-station/src/engine/calls"
)

type fakeMachines struct {
	stuck   map[string]bool
	records []model.VMRecord
	saved   []model.VMRecord
}

func (f *fakeMachines) Records() ([]model.VMRecord, error) { return f.records, nil }

func (f *fakeMachines) Save(r model.VMRecord) error {
	f.saved = append(f.saved, r)
	return nil
}

func (f *fakeMachines) Remove(context.Context, string) (model.DeleteReport, error) {
	return model.DeleteReport{}, nil
}

func (f *fakeMachines) PowerOff(context.Context, string) error { return nil }

func (f *fakeMachines) EngineRunning(_ context.Context, name string) bool { return f.stuck[name] }

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

func TestAtStartALiveRunIsAdoptedNeverRunAgain(t *testing.T) {
	h := newHarness(t, model.VMRecord{Name: "api", Lifetime: model.Persistent, Want: model.WantRunning})
	h.hv.outlived("api")
	h.d.adopt(cancelled(t))
	if got := h.state("api"); got != engine.StateRunning || h.runs() != 0 {
		t.Errorf("adopted api: %s with %d runs, want running and none", got, h.runs())
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
