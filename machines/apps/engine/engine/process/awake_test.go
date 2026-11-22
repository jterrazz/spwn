package process

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type heldPower struct{ holds atomic.Int32 }

func (*heldPower) Capabilities() contract.PowerCaps { return contract.PowerCaps{} }
func (*heldPower) Read() (awake.Reading, error)     { return awake.Reading{Source: "ac"}, nil }
func (p *heldPower) Hold(string) (func(), error) {
	p.holds.Add(1)
	return func() { p.holds.Add(-1) }, nil
}
func (*heldPower) Clamshell(bool) error                  { return nil }
func (*heldPower) DisplaySleepNow(context.Context) error { return nil }
func (*heldPower) Sleeping() bool                        { return false }
func (*heldPower) Watch(context.Context) <-chan awake.Event {
	out := make(chan awake.Event)
	close(out)
	return out
}

// invariant: a Never sleeps written while the engine runs (settings apply
// never-sleeps, from the CLI) is held at the engine's next tick, a second
// on, never at its next 30 s read; one turned off is let go as soon.
func TestNeverSleepsWrittenIsHeldAtTheNextTick(t *testing.T) {
	clock := &stepClock{At: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)}
	power := &heldPower{}
	var on atomic.Bool
	d := New(&fakeMac{clock: clock}, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{
		State: setup.Paths{State: "/s"}.State, ThisMachine: catalog.Mac, Power: power, NeverSleeps: on.Load,
	})
	ctx := t.Context()
	d.neverSleeps(clock.At)
	defer d.closeAll()
	on.Store(true)
	d.step(ctx, 1)
	if !d.AwakeView().NeverHeld || power.holds.Load() != 1 {
		t.Fatalf("one tick after Never sleeps was written: held %v, %d holds; want held", d.AwakeView().NeverHeld, power.holds.Load())
	}
	on.Store(false)
	d.step(ctx, 2)
	if d.AwakeView().NeverHeld || power.holds.Load() != 0 {
		t.Fatalf("one tick after Never sleeps was turned off: held %v, %d holds; want let go", d.AwakeView().NeverHeld, power.holds.Load())
	}
}
