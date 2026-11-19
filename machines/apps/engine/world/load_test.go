package world_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

func TestALoadCurveVariesWithTheClockAndRepeatsForOneSeed(t *testing.T) {
	at := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	meter := func(seed uint64) world.Meter {
		return world.Meter{
			Sample:    activity.Sample{At: at, CPUBusy: 1000, CPUTotal: 10000},
			PerSecond: &activity.Sample{CPUBusy: 500, CPUTotal: 1000},
			Load:      &world.Load{Seed: seed, Floor: 100, Ceil: 900},
		}
	}
	m := meter(7)
	first := m.Answers(at.Add(10 * time.Minute))
	second := m.Answers(at.Add(20 * time.Minute))
	if first.CPUBusy == 1000 || second.CPUBusy <= first.CPUBusy {
		t.Fatalf("busy counts %d then %d: they do not grow with the clock", first.CPUBusy, second.CPUBusy)
	}
	if again := m.Answers(at.Add(10 * time.Minute)); again.CPUBusy != first.CPUBusy {
		t.Errorf("one clock read twice gave %d then %d", first.CPUBusy, again.CPUBusy)
	}
	one, other := meter(7), meter(8)
	if a, b := one.Answers(at.Add(time.Hour)).CPUBusy, other.Answers(at.Add(time.Hour)).CPUBusy; a == b {
		t.Errorf("two seeds drew the same curve: %d", a)
	}
	total := first.CPUTotal - 10000
	if busy := first.CPUBusy - 1000; busy < total/10 || busy > total*9/10 {
		t.Errorf("%d busy over %d total leaves the curve's 10 to 90 %% band", busy, total)
	}
}
