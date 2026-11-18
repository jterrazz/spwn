package process

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type driftReads struct {
	asked     []string
	bounded   []bool
	answer    model.KeptDrift
	scheduled []func()
}

func driftEngine(clock *manualClock, r *driftReads) *Engine {
	paths := setup.Paths{Prefs: "/c", State: "/s"}
	d := New(&fakeMac{clock: clock}, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: paths.State, ThisMachine: catalog.Mac,
		Drift: func(ctx context.Context, vm string) model.KeptDrift {
			_, bounded := ctx.Deadline()
			r.asked, r.bounded = append(r.asked, vm), append(r.bounded, bounded)
			return r.answer
		},
		Changes: []string{"machinefile.apply"},
		Beside:  func(f func()) { r.scheduled = append(r.scheduled, f) },
	})
	up := speaksFirst{done: make(chan struct{})}
	boot := clock.Now().Add(-time.Hour)
	d.vms["linux-1"] = &vmState{name: "linux-1", rec: model.VMRecord{Name: "linux-1", Lifetime: model.Persistent}, state: engine.StateRunning, client: up, bootAt: boot}
	d.vms["infra"] = &vmState{name: "infra", rec: model.VMRecord{Name: "infra", Lifetime: model.AlwaysOn}, state: engine.StateRunning, client: up, bootAt: boot}
	d.vms["api"] = &vmState{name: "api", rec: model.VMRecord{Name: "api", Lifetime: model.Persistent}, state: engine.StateStopped}
	return d
}

func (r *driftReads) run() {
	for _, f := range r.scheduled {
		f()
	}
	r.scheduled = nil
}

func keptOf(d *Engine, vm string) *model.KeptDrift {
	for _, v := range d.vmViews() {
		if v.Name == vm {
			return v.Drift
		}
	}
	return nil
}

func TestEachRunningWorkspaceIsReadBesideTheLoopOnceThenAgainAfterAChangeOrItsPeriod(t *testing.T) {
	clock := newManualClock()
	r := &driftReads{answer: model.KeptDrift{Differences: 3}}
	d := driftEngine(clock, r)
	ctx := t.Context()

	d.keepDrift(ctx)
	if len(r.asked) != 0 || keptOf(d, "linux-1") != nil {
		t.Fatalf("a reading runs beside the loop, never inside the step: asked %v, kept %v", r.asked, keptOf(d, "linux-1"))
	}
	d.keepDrift(ctx)
	if len(r.scheduled) != 1 {
		t.Fatalf("one reading per VM at a time: %d scheduled", len(r.scheduled))
	}
	r.run()
	if len(r.asked) != 1 || r.asked[0] != "linux-1" {
		t.Fatalf("only the running workspace is read, never a server or a stopped VM: %v", r.asked)
	}
	if !r.bounded[0] {
		t.Errorf("a reading runs with no bound")
	}
	if kept := keptOf(d, "linux-1"); kept == nil || kept.Differences != 3 || !kept.At.Equal(clock.Now()) {
		t.Fatalf("the reading is kept, at when it began: %+v", kept)
	}

	d.keepDrift(ctx)
	if len(r.scheduled) != 0 {
		t.Fatal("a fresh reading is not read again")
	}
	d.changed("linux-1", "apps.list")
	d.keepDrift(ctx)
	if len(r.scheduled) != 0 {
		t.Fatal("a call that changes nothing makes no reading due")
	}
	r.answer = model.KeptDrift{}
	d.changed("linux-1", "machinefile.apply")
	d.keepDrift(ctx)
	r.run()
	if kept := keptOf(d, "linux-1"); kept == nil || kept.Differences != 0 {
		t.Fatalf("an apply that ended makes the reading due again: %+v", kept)
	}

	r.answer = model.KeptDrift{Differences: 1}
	clock.mu.Lock()
	clock.now = clock.now.Add(driftEvery - time.Second)
	clock.mu.Unlock()
	d.keepDrift(ctx)
	if len(r.scheduled) != 0 {
		t.Fatal("a reading holds for 10 min")
	}
	clock.mu.Lock()
	clock.now = clock.now.Add(time.Second)
	clock.mu.Unlock()
	d.keepDrift(ctx)
	r.run()
	if kept := keptOf(d, "linux-1"); kept == nil || kept.Differences != 1 {
		t.Fatalf("a reading 10 min old is read again: %+v", kept)
	}
}

func TestAReadingFromBeforeTheBootIsNeverSaid(t *testing.T) {
	clock := newManualClock()
	r := &driftReads{answer: model.KeptDrift{Differences: 3}}
	d := driftEngine(clock, r)
	d.keepDrift(t.Context())
	r.run()
	d.mu.Lock()
	d.vms["linux-1"].bootAt = clock.Now().Add(time.Second)
	d.mu.Unlock()
	if kept := keptOf(d, "linux-1"); kept != nil {
		t.Fatalf("a reading of the run before is not this run's: %+v", kept)
	}
	d.keepDrift(t.Context())
	if len(r.scheduled) != 1 {
		t.Fatal("a VM booted since its reading is read again")
	}
}
