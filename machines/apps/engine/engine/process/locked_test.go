package process

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

// value: a clock whose next read panics, as a bug under the engine's lock would.
type breakingClock struct {
	stepClock
	breaks atomic.Bool
}

func (c *breakingClock) Now() time.Time {
	if c.breaks.CompareAndSwap(true, false) {
		panic("the clock broke")
	}
	return c.stepClock.Now()
}

func breakingEngine(t *testing.T) (*Engine, *breakingClock) {
	t.Helper()
	clock := &breakingClock{stepClock: stepClock{At: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}}
	paths := setup.Paths{Prefs: "/c", State: "/s"}
	m := &fakeMachines{records: []model.VMRecord{{Name: "api", Lifetime: model.Persistent}}}
	d := New(&fakeMac{clock: clock}, newHypervisor(), nil, m, &fakeTailnet{}, memStores(), Options{State: paths.State, ThisMachine: catalog.Mac})
	return d, clock
}

func lockFreeAfterAPanic(t *testing.T, d *Engine, f func()) {
	t.Helper()
	var broke any
	func() {
		defer func() { broke = recover() }()
		f()
	}()
	if broke == nil {
		t.Fatal("the call did not panic: the test proves nothing")
	}
	done := make(chan struct{})
	go func() {
		d.vmViews()
		close(done)
	}()
	bound, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	select {
	case <-done:
	case <-bound.Done():
		t.Fatal("the engine's lock stayed held after a panic under it: the next call hangs")
	}
}

func TestAReadyThatPanicsUnderTheEngineLockLeavesItFree(t *testing.T) {
	d, clock := breakingEngine(t)
	clock.breaks.Store(true)
	lockFreeAfterAPanic(t, d, func() { _ = d.ready(context.Background(), "api") })
}

func TestALocalNetworkReadThatPanicsUnderTheEngineLockLeavesItFree(t *testing.T) {
	d, clock := breakingEngine(t)
	clock.breaks.Store(true)
	lockFreeAfterAPanic(t, d, func() { d.rereadLocalNetwork(context.Background()) })
}
