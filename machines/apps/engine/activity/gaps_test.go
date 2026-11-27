package activity

import (
	"encoding/json"
	"testing"
	"time"
)

func half(time.Time) float64 { return 50 }

func (sp *sampler) reboot() {
	// value: the machine booted again at sp.at: its counters start near
	// zero, its boot is now.
	sp.last.CPUTotal, sp.last.CPUBusy, sp.last.DiskRead, sp.last.Boot = 1000, 500, 0, sp.at
}

func gapsOf(points []Point) []time.Time {
	var out []time.Time
	for _, p := range points {
		if p.Rated == nil {
			out = append(out, p.At)
		}
	}
	return out
}

func TestARebootIsAGapAtItsOwnBucketAndThePointsBeforeItStay(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(2*time.Hour, IdleEvery, half)
	booted := sp.at
	sp.reboot()
	sp.run(time.Hour, IdleEvery, half)
	now := sp.at
	day := sp.s.History(now)
	if len(day) != 181 || !day[0].At.Equal(from) {
		t.Fatalf("the history across a reboot holds %d points from %v; want the 181 minutes from %v", len(day), day[0].At, from)
	}
	gaps := gapsOf(day)
	if len(gaps) != 2 || !gaps[0].Equal(from) || !gaps[1].Equal(booted.Add(IdleEvery)) {
		t.Errorf("the history's gaps are %v; want the first sample's and the reboot's minute alone, %v", gaps, booted.Add(IdleEvery))
	}
	for _, p := range day {
		if r := p.Rated; r != nil && !near(r.CPU.Mean, 50) {
			t.Errorf("the minute at %v rates %v %% CPU; want the 50 %% measured", p.At, r.CPU.Mean)
		}
	}
	graph := sp.s.Graph(24*time.Hour, now)
	before := 0
	for _, p := range graph {
		if p.At.Before(booted) {
			before++
		}
		if r := p.Rated; r != nil && !near(r.CPU.Mean, 50) {
			t.Errorf("the day's point at %v rates %v %% CPU; want 50", p.At, r.CPU.Mean)
		}
	}
	if before != 8 {
		t.Errorf("the day's graph keeps %d points before the reboot; want its 8 quarters", before)
	}
}

func TestAPauseIsAGapInThePointsNeverAZeroRate(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(time.Hour, IdleEvery, half)
	paused := sp.at
	sp.at = sp.at.Add(10 * time.Minute)
	sp.run(30*time.Minute, IdleEvery, half)
	now := sp.at
	day := sp.s.History(now)
	var around []Point
	for _, p := range day {
		if p.At.Equal(paused) || p.At.Equal(paused.Add(11*time.Minute)) {
			around = append(around, p)
		}
		if p.At.After(paused) && p.At.Before(paused.Add(11*time.Minute)) {
			t.Errorf("the history has a point at %v inside the pause; want none", p.At)
		}
	}
	if len(around) != 2 || around[0].Rated == nil || !near(around[0].Rated.CPU.Mean, 50) || around[1].Rated != nil {
		t.Fatalf("the points around the pause are %+v; want the minute before it at 50 %%, the first after it a gap", around)
	}
	for _, b := range ChartOf(day, time.Hour, now).Buckets {
		if b.Points > 0 && b.Mean == 0 {
			t.Errorf("the hour's bar at %v reads 0 %% over %d samples; want a gap or the 50 %% measured", b.At, b.Points)
		}
	}
}

func TestAMinutesPointCarriesTheLoadFreeSpaceAndBatteryItsSampleRead(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var s Series
	for i := range 30 {
		s.Add(Sample{
			At: from.Add(time.Duration(i) * time.Minute), CPUTotal: uint64(i) * 60_000, Load: [3]float64{4.8, 3.25, 1},
			DiskFree: 388_000_000_123 - int64(i), Battery: &Battery{Percent: 96, Charging: true, Health: 98},
		})
	}
	now := from.Add(29 * time.Minute)
	data, err := json.Marshal(s.Part(Want{History: true}, now))
	if err != nil {
		t.Fatal(err)
	}
	var again Series
	if err := json.Unmarshal(data, &again); err != nil {
		t.Fatal(err)
	}
	for _, kept := range []*Series{&s, &again} {
		day := kept.History(now)
		for i, p := range day {
			if p.Load != [3]float64{4.8, 3.25, 1} || p.DiskFree != 388_000_000_123-int64(i) || p.Battery == nil || *p.Battery != (Battery{Percent: 96, Charging: true}) {
				t.Fatalf("the minute at %v carries load %v, %d B free, battery %+v; want its sample's 4.8 3.25 1, %d B, 96 %% charging", p.At, p.Load, p.DiskFree, p.Battery, 388_000_000_123-int64(i))
			}
		}
		if len(day) != 30 {
			t.Errorf("the history holds %d minutes; want 30", len(day))
		}
	}
	for _, p := range s.Graph(LiveWindow, now) {
		if p.Load != ([3]float64{}) || p.DiskFree != 0 || p.Battery != nil {
			t.Errorf("a point of the 1 s tier carries %+v; want no gauge but T2's", p)
		}
	}
}

func TestTheMenuBarsDayAndTheEnginesAreOneFold(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(26*time.Hour, IdleEvery, func(at time.Time) float64 { return float64(at.Minute()%7) * 13 })
	sp.run(17*time.Minute, IdleEvery, func(time.Time) float64 { return 99 })
	now := sp.at.Add(17 * time.Second)
	for _, window := range []time.Duration{time.Hour, 24 * time.Hour} {
		engine, menubar := sp.s.Chart(window, now), ChartOf(sp.s.History(now), window, now)
		if len(engine.Buckets) != len(menubar.Buckets) || engine.Peak != menubar.Peak || !engine.PeakAt.Equal(menubar.PeakAt) {
			t.Fatalf("over %v the engine draws %d bars peaking at %d, the menu bar %d at %d; want the same", window, len(engine.Buckets), engine.Peak, len(menubar.Buckets), menubar.Peak)
		}
		for i, b := range engine.Buckets {
			m := menubar.Buckets[i]
			if !b.At.Equal(m.At) || b.Points != m.Points || !near(b.Mean, m.Mean) || b.Max != m.Max || !b.MaxAt.Equal(m.MaxAt) {
				t.Errorf("over %v bar %d is %+v for the engine, %+v for the menu bar; want the same bar", window, i, b, m)
			}
		}
	}
}
