package contracttest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/awake/contract"
)

func Laptop() awake.Reading {
	return awake.Reading{Lid: "open", Source: "ac", HasBattery: true, Percent: 80, Charging: true}
}

func Desktop() awake.Reading { return awake.Reading{Source: "ac"} }

type Open func(t *testing.T, r awake.Reading, none bool) contract.Power

func Power(t *testing.T, open Open) {
	t.Run("a laptop reports a battery and a lid, and its reading carries them", func(t *testing.T) {
		p := open(t, Laptop(), false)
		if got := p.Capabilities(); got != (contract.PowerCaps{Battery: true, Lid: true, Thermal: true}) {
			t.Fatalf("Capabilities = %+v", got)
		}
		r, err := p.Read()
		if err != nil || r != Laptop() {
			t.Fatalf("Read = %+v, %v", r, err)
		}
	})
	t.Run("a Mac macOS has not told it goes to sleep reads awake", func(t *testing.T) {
		if open(t, Laptop(), false).Sleeping() {
			t.Fatal("Sleeping = true with no will-sleep heard")
		}
		if open(t, awake.Reading{}, true).Sleeping() {
			t.Fatal("Sleeping = true with no power management")
		}
	})
	t.Run("a desktop reports neither battery nor lid, and those fields stay zero", func(t *testing.T) {
		p := open(t, Desktop(), false)
		if got := p.Capabilities(); got.Battery || got.Lid {
			t.Fatalf("Capabilities = %+v", got)
		}
		r, err := p.Read()
		if err != nil || r.HasBattery || r.Lid != "" || r.Percent != 0 {
			t.Fatalf("Read = %+v, %v", r, err)
		}
	})
	t.Run("a hold is released once, however many times its release is called", func(t *testing.T) {
		p := open(t, Laptop(), false)
		release, err := p.Hold("station: spec")
		if err != nil {
			t.Fatalf("Hold: %v", err)
		}
		release()
		release()
		if err := p.Clamshell(true); err != nil {
			t.Fatalf("Clamshell(true): %v", err)
		}
		if err := p.Clamshell(false); err != nil {
			t.Fatalf("Clamshell(false): %v", err)
		}
	})
	t.Run("a machine with no power management reports nothing and holds nothing", func(t *testing.T) {
		p := open(t, awake.Reading{}, true)
		if got := p.Capabilities(); got != (contract.PowerCaps{}) {
			t.Fatalf("Capabilities = %+v", got)
		}
		if _, err := p.Read(); !errors.Is(err, contract.ErrUnsupported) {
			t.Fatalf("Read: %v, want ErrUnsupported", err)
		}
		if _, err := p.Hold("station: spec"); !errors.Is(err, contract.ErrUnsupported) {
			t.Fatalf("Hold: %v, want ErrUnsupported", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		events := p.Watch(ctx)
		select {
		case _, open := <-events:
			if open {
				t.Fatal("the watch sent an event with no power management")
			}
		case <-ctx.Done():
			t.Fatal("the watch stayed open with no power management")
		}
	})
}
