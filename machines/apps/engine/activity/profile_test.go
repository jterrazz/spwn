package activity_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
)

func steady() activity.Profile {
	return activity.Profile{Base: 30, Swing: 4, Seed: 3, Bumps: []activity.Bump{{Ago: 20 * time.Minute, Width: 4 * time.Minute, Height: 20}}, In: 0.5, Out: 0.2}
}

func TestAProfileGivesTheSamePointsEveryTimeItIsAskedTheSameThing(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	first := steady().Points(now, time.Hour, time.Minute)
	second := steady().Points(now, time.Hour, time.Minute)

	if !reflect.DeepEqual(first, second) {
		t.Errorf("two asks of one profile gave different points")
	}
}

func TestAProfilesPointsAreEvenlySpacedAndEndAtNow(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	points := steady().Points(now, 6*time.Minute, 2*time.Second)

	if len(points) != 181 {
		t.Fatalf("len = %d; want 181 (6 min at 2 s, both ends)", len(points))
	}
	if !points[0].At.Equal(now.Add(-6*time.Minute)) || !points[len(points)-1].At.Equal(now) {
		t.Errorf("ends = %v .. %v; want now-6min .. now", points[0].At, points[len(points)-1].At)
	}
	for i := 1; i < len(points); i++ {
		if got := points[i].At.Sub(points[i-1].At); got != 2*time.Second {
			t.Fatalf("gap %d = %v; want 2s", i, got)
		}
	}
}

func TestAProfilesCountersNeverGoBack(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	points := steady().Points(now, time.Hour, time.Minute)

	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		if b.CPUBusy < a.CPUBusy || b.CPUTotal <= a.CPUTotal || b.NetIn < a.NetIn || b.NetOut < a.NetOut {
			t.Fatalf("counters went back at point %d: %+v then %+v", i, a, b)
		}
	}
}

func TestAProfilesBumpRisesWhereItWasToldRelativeToNow(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)

	points := steady().Points(now, time.Hour, time.Minute)
	at := func(ago time.Duration) float64 {
		for i := 1; i < len(points); i++ {
			if points[i].At.Equal(now.Add(-ago)) {
				return float64(points[i].CPUBusy-points[i-1].CPUBusy) / float64(points[i].CPUTotal-points[i-1].CPUTotal) * 100
			}
		}
		t.Fatalf("no point %v ago", ago)
		return 0
	}

	if peak, calm := at(20*time.Minute), at(40*time.Minute); peak < calm+10 {
		t.Errorf("busy at the bump = %.1f, away from it = %.1f; want the bump 10 points above", peak, calm)
	}
}
