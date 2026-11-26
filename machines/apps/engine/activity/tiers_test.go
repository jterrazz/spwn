package activity

import (
	"encoding/json"
	"math"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// value: a machine sampled every gap from `from`, its CPU at the share
// busy(i) gives the interval ending at sample i, its disk reading 1 MB a
// second and its memory 1 GB more each sample.
type sampler struct {
	at   time.Time
	last Sample
	s    Series
}

func samplerAt(from time.Time) *sampler {
	first := Sample{At: from, CPUs: 4, MemTotal: 64 << 30, Pressure: PressureNormal}
	sp := &sampler{at: from, last: first}
	sp.s.Add(first)
	return sp
}

func (sp *sampler) step(gap time.Duration, share float64) {
	sp.at = sp.at.Add(gap)
	x := sp.last
	ticks := uint64(gap.Seconds() * 400)
	x.At, x.CPUTotal, x.CPUBusy = sp.at, x.CPUTotal+ticks, x.CPUBusy+uint64(share/100*float64(ticks))
	x.DiskRead += uint64(gap.Seconds() * 1e6)
	x.MemUsed = (x.MemUsed + 1<<30) % (32 << 30)
	sp.s.Add(x)
	sp.last = x
}

func (sp *sampler) run(d, gap time.Duration, share func(at time.Time) float64) {
	for end := sp.at.Add(d); sp.at.Before(end); {
		sp.step(gap, share(sp.at.Add(gap)))
	}
}

func near(got, want float64) bool { return math.Abs(got-want) <= 1e-4*math.Max(1, math.Abs(want)) }

func TestABurstOf3MinAt100PercentInADaySurvivesAsTheMaxOfEveryTier(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	now := from.Add(24 * time.Hour)
	burst := now.Add(-8 * time.Minute)
	sp := samplerAt(from)
	sp.run(24*time.Hour, WatchedEvery, func(at time.Time) float64 {
		if !at.Before(burst) && at.Before(burst.Add(3*time.Minute)) {
			return 100
		}
		return 10
	})
	for i, tier := range Tiers {
		top := float32(0)
		for _, b := range sp.s.within(i, burst, now) {
			top = max(top, b.cpu.max)
		}
		if top != 100 {
			t.Errorf("tier %v keeps a busiest share of %v over the burst; want 100", tier.Every, top)
		}
	}
	day := sp.s.Chart(24*time.Hour, now)
	peak := day.Buckets[day.Peak]
	if peak.Max != 100 || peak.Mean > 30 {
		t.Errorf("the day's peak bar is %+v; want the burst's 100 %% as its max over a mean the 30 min keep low", peak)
	}
	if !near(peak.Mean, (27*10+3*100)/30.0) {
		t.Errorf("the peak bar's mean is %v; want the 30 min's counter mean, 19 %%", peak.Mean)
	}
}

func TestEachTiersMeanIsTheCountersMean(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	share := func(at time.Time) float64 { return float64(at.Second()%7) * 13 }
	first := sp.last
	sp.run(58*time.Second, WatchedEvery, share)
	minute := sp.last
	sp.run(2*time.Minute, WatchedEvery, share)
	want := Between(first, minute)
	buckets := sp.s.within(T2, from, sp.at)
	b := buckets[0]
	if len(buckets) != 3 || !near(float64(b.cpu.mean), want.CPU) || !near(float64(b.diskRead.mean), want.DiskRead) || b.span != 58 {
		t.Errorf("the first minute is %+v over %d buckets; want its counter mean %.4f %% CPU, %v B/s read over the 58 s its 29 intervals span", b, len(buckets), want.CPU, want.DiskRead)
	}
	if mid := buckets[1]; mid.n != 30 || mid.span != 60 {
		t.Errorf("a whole minute holds %d samples over %v s; want 30 over 60 s", mid.n, mid.span)
	}
}

func TestAPauseInTheSamplingIsAGapNeverAZeroNorALine(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	busy := func(time.Time) float64 { return 50 }
	sp.run(time.Minute, WatchedEvery, busy)
	paused := sp.at
	sp.at = sp.at.Add(10 * time.Minute)
	sp.run(time.Minute, WatchedEvery, busy)
	now := sp.at
	if first := sp.s.within(T0, paused.Add(10*time.Minute), now)[0]; first.span != 0 || first.cpu != (envelope{}) {
		t.Errorf("the first sample after the pause rolls into %+v; want its gauges only, no rate across the 10 min", first)
	}
	for i := range Tiers[:T3] {
		for _, b := range sp.s.within(i, paused.Add(time.Second), paused.Add(10*time.Minute-time.Second)) {
			t.Errorf("tier %v keeps a bucket %+v inside the pause; want none", Tiers[i].Every, b)
		}
	}
	hour := sp.s.Chart(time.Hour, now)
	for _, b := range hour.Buckets {
		if b.At.After(paused) && b.At.Add(time.Minute).Before(paused.Add(10*time.Minute)) && b.Points != 0 {
			t.Errorf("the hour's bar at %v inside the pause holds %+v; want a gap", b.At, b)
		}
		if b.Points > 0 && b.Mean != 50 {
			t.Errorf("the hour's bar at %v reads %v %%; want 50: no share is spread across the pause", b.At, b.Mean)
		}
	}
	for _, x := range sp.s.Graph(time.Hour, now) {
		if x.At.After(paused) && x.At.Before(paused.Add(10*time.Minute)) {
			t.Errorf("the hour's graph has a point at %v inside the pause; want none", x.At)
		}
	}
}

func TestACounterThatWentBackIsAGapNeverANegativeRate(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var s Series
	s.Add(Sample{At: from, CPUBusy: 9e6, CPUTotal: 2e7, DiskRead: 182e9})
	s.Add(Sample{At: from.Add(2 * time.Second), CPUBusy: 300, CPUTotal: 1200, DiskRead: 1e8})
	b := s.within(T0, from, from.Add(2*time.Second))
	if len(b) != 2 || b[1].span != 0 || b[1].cpu != (envelope{}) || b[1].diskRead != (envelope{}) {
		t.Errorf("the sample after a reboot rolls into %+v; want its gauges only, no rate measured", b)
	}
}

func TestASampleAheadOfItsReadersClockIsStillDrawn(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	var s Series
	s.Add(Sample{At: from.Add(4 * time.Second), CPUTotal: 10})
	if got := s.Graph(LiveWindow, from); len(got) != 1 {
		t.Errorf("a reader 4 s behind the engine's newest sample graphs %d points; want that sample", len(got))
	}
}

func TestATiersRingWrapsAndKeepsOnlyItsHorizon(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(15*time.Minute, time.Second, func(time.Time) float64 { return 5 })
	got := sp.s.within(T0, from, sp.at)
	if len(got) != 600 || got[0].at != sp.at.Add(-599*time.Second).UnixMilli() {
		t.Fatalf("T0 holds %d buckets from %v; want the last 600 s, one a second", len(got), time.UnixMilli(got[0].at).UTC())
	}
	for _, b := range got {
		if b.n != 1 {
			t.Fatalf("a slot of T0 reused holds %d samples; want only the newest second's one", b.n)
		}
	}
	for i, r := range sp.s.rings {
		if want := int(Tiers[i].Kept / Tiers[i].Every); len(r.slots) != want {
			t.Errorf("tier %v holds %d slots; want %d", Tiers[i].Every, len(r.slots), want)
		}
	}
}

func TestTheTiersReadBackAsTheyWereWritten(t *testing.T) {
	from := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	sp := samplerAt(from)
	sp.run(3*time.Hour, 10*time.Second, func(at time.Time) float64 { return float64(at.Minute()) })
	data, err := json.Marshal(sp.s)
	if err != nil {
		t.Fatal(err)
	}
	var again Series
	if err := json.Unmarshal(data, &again); err != nil {
		t.Fatal(err)
	}
	if again.last != sp.s.last || again.ticks != sp.s.ticks {
		t.Errorf("the newest sample reads back as %+v at %v ticks/s; want %+v at %v", again.last, again.ticks, sp.s.last, sp.s.ticks)
	}
	for i := range Tiers {
		if want, got := sp.s.within(i, from, sp.at), again.within(i, from, sp.at); len(got) != len(want) || got[len(got)-1] != want[len(want)-1] || got[0] != want[0] {
			t.Errorf("tier %v reads back %d buckets; want the %d written, alike", Tiers[i].Every, len(got), len(want))
		}
	}
	if err := json.Unmarshal([]byte(`{"last": {"at": "2026-11-25T12:00:00Z"}, "tiers": [{"every": 2000000000, "buckets": []}]}`), &again); err == nil {
		t.Error("a tier of 2 s read back; want it refused, no tier has that step")
	}
}

func TestAMachinesTiersHoldAboutHalfAMegabyte(t *testing.T) {
	if size := reflect.TypeFor[bucket]().Size(); size != 72 {
		t.Errorf("a bucket is %d bytes; want 72", size)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var s Series
	s.Add(Sample{At: time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)})
	runtime.ReadMemStats(&after)
	bytes := after.TotalAlloc - before.TotalAlloc
	t.Logf("a machine's tiers: %d bytes", bytes)
	if bytes > 600_000 || bytes < 500_000 {
		t.Errorf("a machine's tiers hold %d bytes; want about 520 KB, never over 600 KB", bytes)
	}
	runtime.KeepAlive(&s)
}
