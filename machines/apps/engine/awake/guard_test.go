package awake_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

func names(gs []awake.Guard) []string {
	out := make([]string, 0, len(gs))
	for _, g := range gs {
		out = append(out, g.Name)
	}
	return out
}

func TestADesktopMacHasNeitherTheBatteryNorTheLidGuard(t *testing.T) {
	got := names(awake.Active(machine.Capabilities{Thermal: true}))
	if want := []string{awake.Thermal, awake.LowPower}; !slices.Equal(got, want) {
		t.Fatalf("a desktop's guards = %v, want %v", got, want)
	}
}

func TestALaptopHasEveryGuardInItsOrder(t *testing.T) {
	got := names(awake.Active(machine.Capabilities{Battery: true, Lid: true, Thermal: true}))
	if want := []string{awake.Thermal, awake.Battery, awake.LowPower, awake.Lid}; !slices.Equal(got, want) {
		t.Fatalf("a laptop's guards = %v, want %v", got, want)
	}
}

func TestABatteryEndsASessionOnlyWhereTheMacReportsOne(t *testing.T) {
	low := awake.Reading{Source: "battery", HasBattery: true, Percent: 15, Lid: "open"}
	if v := awake.Next(laptop, awake.Held{}, low, 0); !v.Ends || v.Trip.Guard != awake.Battery {
		t.Fatalf("a laptop at 15%% on battery: %+v; want the battery guard to end it", v)
	}
	if v := awake.Next(desktop, awake.Held{}, low, 0); v.Tripped {
		t.Fatalf("a Mac reporting no battery tripped %+v", v)
	}
}

var (
	laptop  = machine.Capabilities{Battery: true, Lid: true, Thermal: true}
	desktop = machine.Capabilities{Thermal: true}
)

func TestNeverSleepsOnlyTellsOfAGuardButTheBatteryOfALaptop(t *testing.T) {
	never := awake.Held{NeverSleeps: true}
	cases := []struct {
		name     string
		caps     machine.Capabilities
		r        awake.Reading
		heavyFor time.Duration
		guard    string
		ends     bool
	}{
		{"serious thermal on a desktop", desktop, awake.Reading{Source: "ac", Thermal: awake.ThermalEnds}, 0, awake.Thermal, false},
		{"Low Power Mode on a desktop", desktop, awake.Reading{Source: "ac", LowPower: true}, 0, awake.LowPower, false},
		{"a laptop at 20 % on battery", laptop, awake.Reading{Source: "battery", HasBattery: true, Percent: 20, Lid: "open"}, 0, awake.Battery, true},
		{"a laptop at 21 % on battery", laptop, awake.Reading{Source: "battery", HasBattery: true, Percent: 21, Lid: "open"}, 0, "", false},
		{"a laptop at 20 % on AC", laptop, awake.Reading{Source: "ac", HasBattery: true, Percent: 20, Lid: "open"}, 0, "", false},
		{"heavy thermal 5 min on a laptop", laptop, awake.Reading{Source: "battery", HasBattery: true, Percent: 80, Lid: "closed", Thermal: awake.ThermalHeavy}, 5 * time.Minute, awake.Thermal, false},
	}
	for _, c := range cases {
		v := awake.Next(c.caps, never, c.r, c.heavyFor)
		if v.Trip.Guard != c.guard || v.Ends != c.ends {
			t.Errorf("%s: %+v; want guard %q, ends %v", c.name, v, c.guard, c.ends)
		}
	}
}

func TestAKeepAwakeEndsAtEachGuardAndItsLidClosedOnBatteryUnlessAllowed(t *testing.T) {
	closed := awake.Reading{Source: "battery", HasBattery: true, Percent: 80, Lid: "closed"}
	if v := awake.Next(laptop, awake.Held{Lid: true}, closed, 0); !v.Ends || v.Trip.Guard != awake.Lid {
		t.Errorf("--lid closed on battery: %+v; want the lid guard to end it", v)
	}
	if v := awake.Next(laptop, awake.Held{Lid: true, Battery: true}, closed, 0); v.Tripped {
		t.Errorf("--lid --battery closed on battery tripped %+v", v)
	}
	heavy := closed
	heavy.Thermal = awake.ThermalHeavy
	if v := awake.Next(laptop, awake.Held{Lid: true, Battery: true}, heavy, 4*time.Minute); v.Tripped {
		t.Errorf("heavy for 4 min tripped %+v", v)
	}
	if v := awake.Next(laptop, awake.Held{Lid: true, Battery: true}, heavy, 5*time.Minute); !v.Ends || v.Trip.Detail != "thermal heavy for 5 min" {
		t.Errorf("heavy for 5 min: %+v; want it ended", v)
	}
}
