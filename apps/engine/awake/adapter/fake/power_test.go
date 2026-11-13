package fake

import (
	"context"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract/contracttest"
)

func TestTheFakePowerKeepsTheContract(t *testing.T) {
	contracttest.Power(t, func(_ *testing.T, r awake.Reading, none bool) contract.Power {
		if none {
			return NewNoPower()
		}
		return NewPower(r)
	})
}

func TestUnpluggingTheFakeLaptopTellsTheWatch(t *testing.T) {
	p := NewPower(contracttest.Laptop())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := p.Watch(ctx)
	unplugged := contracttest.Laptop()
	unplugged.Source = "battery"
	p.Set(unplugged)
	select {
	case e := <-events:
		if e.Kind != "source" {
			t.Fatalf("event %q, want source", e.Kind)
		}
	case <-ctx.Done():
		t.Fatal("the watch never heard the unplug")
	}
}
