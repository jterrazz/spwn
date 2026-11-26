package activity

import (
	"math"
	"time"
)

const (
	// value: SPEC §6.3's cadence: a machine a surface watches is sampled
	// every WatchedEvery, any other every IdleEvery.
	WatchedEvery = 2 * time.Second
	IdleEvery    = time.Minute
	// value: an interval between two samples longer than Pause is a pause
	// in the sampling: no rate is measured across it, the tiers keep a gap.
	Pause = 2 * IdleEvery
	// value: the shortest window a screen draws, its chart at 5 s a bar.
	LiveWindow = 6 * time.Minute
)

type Tier struct {
	// value: a tier's bucket spans Every; the tier keeps the last Kept.
	Every, Kept time.Duration
}

const (
	T0 = iota
	T1
	T2
	T3
)

var Tiers = [...]Tier{
	// value: ADR-107's tiers, finest first: 1 s for 10 min, 10 s for 3 h,
	// 1 min for 48 h, 15 min for 30 days.
	T0: {time.Second, 10 * time.Minute},
	T1: {10 * time.Second, 3 * time.Hour},
	T2: {time.Minute, 48 * time.Hour},
	T3: {15 * time.Minute, 30 * 24 * time.Hour},
}

type envelope struct{ mean, max float32 }

func (e *envelope) gauge(x float64, n uint16) {
	if n == 1 || float32(x) > e.max {
		e.max = float32(x)
	}
	e.mean = float32(float64(e.mean) + (x-float64(e.mean))/float64(n))
}

func (e *envelope) rate(x, span, dt float64) {
	if span == 0 || float32(x) > e.max {
		e.max = float32(x)
	}
	e.mean = float32((float64(e.mean)*span + x*dt) / (span + dt))
}

type bucket struct {
	// invariant: at is the unix millisecond of the bucket's newest sample,
	// 0 for a slot nothing was measured in; span the seconds its rates
	// measure, n the samples it counts; floor the least pressure they said.
	at    int64
	span  float32
	n     uint16
	floor uint8

	cpu, mem, swap, diskRead, diskWrite, netIn, netOut envelope
}

type ring struct {
	every int64
	slots []bucket
}

func (r *ring) of(at int64) *bucket { return &r.slots[(at/r.every)%int64(len(r.slots))] }

func (r *ring) add(at int64, x *Sample, rated *Rates, dt float64) {
	b := r.of(at)
	if b.at == 0 || b.at/r.every != at/r.every {
		*b = bucket{}
	}
	b.at = at
	if b.n < math.MaxUint16 {
		b.n++
	}
	rank := pressureRank(x.Pressure)
	if b.n == 1 || rank < b.floor {
		b.floor = rank
	}
	b.mem.gauge(float64(x.MemUsed), b.n)
	b.swap.gauge(float64(x.SwapUsed), b.n)
	if rated == nil {
		return
	}
	span := float64(b.span)
	b.cpu.rate(rated.CPU, span, dt)
	b.diskRead.rate(rated.DiskRead, span, dt)
	b.diskWrite.rate(rated.DiskWrite, span, dt)
	b.netIn.rate(rated.NetIn, span, dt)
	b.netOut.rate(rated.NetOut, span, dt)
	b.span = float32(span + dt)
}

func (r *ring) newest(at int64) []bucket {
	// value: the buckets of this ring from the slot of at back to its
	// oldest, newest first, those of a slot that is not theirs dropped.
	out := make([]bucket, 0, len(r.slots))
	slot := at / r.every
	for i := range int64(len(r.slots)) {
		if b := r.of((slot - i) * r.every); b.at != 0 && b.at/r.every == slot-i {
			out = append(out, *b)
		}
	}
	return out
}

func pressureRank(p Pressure) uint8 {
	switch p {
	case PressureNormal:
		return 1
	case PressureWarn:
		return 2
	case PressureCritical:
		return 3
	}
	return 0
}

func pressureOf(rank uint8) Pressure {
	return [...]Pressure{"", PressureNormal, PressureWarn, PressureCritical}[rank]
}

type Series struct {
	// invariant: the engine's memory only (ADR-107): every sample rolls into
	// each tier's ring of fixed slots, indexed by time, so a tier keeps at
	// most its Kept and a slot no sample reached is a gap. last is the
	// newest sample, ticks the CPU ticks a second its last interval counted.
	last  Sample
	has   bool
	ticks float64
	rings [len(Tiers)]ring
}

func countedAgain(a, b *Sample) bool {
	// value: the machine counted again from zero between a and b: a
	// counter went back, or it booted again.
	return b.CPUTotal < a.CPUTotal || b.CPUBusy < a.CPUBusy || b.DiskRead < a.DiskRead || b.DiskWritten < a.DiskWritten ||
		b.NetIn < a.NetIn || b.NetOut < a.NetOut || (!a.Boot.IsZero() && !b.Boot.IsZero() && !a.Boot.Equal(b.Boot))
}

func (s *Series) Add(x Sample) {
	// invariant: a sample no later than the newest is not kept; a rate is
	// measured between a sample and the one before it, unless a Pause or a
	// machine that counted again lies between them: the sample then counts
	// its gauges only.
	if s.has && !x.At.After(s.last.At) {
		return
	}
	if s.rings[T0].slots == nil {
		for i, t := range Tiers {
			s.rings[i] = ring{every: t.Every.Milliseconds(), slots: make([]bucket, int(t.Kept/t.Every))}
		}
	}
	var rated *Rates
	var dt float64
	if s.has {
		if gap := x.At.Sub(s.last.At); gap <= Pause && !countedAgain(&s.last, &x) {
			r := Between(s.last, x)
			rated, dt = &r, gap.Seconds()
			if x.CPUTotal > s.last.CPUTotal {
				s.ticks = float64(x.CPUTotal-s.last.CPUTotal) / dt
			}
		}
	}
	at := x.At.UnixMilli()
	for i := range s.rings {
		s.rings[i].add(at, &x, rated, dt)
	}
	s.last, s.has = x, true
}

func (s *Series) Last() (Sample, bool) { return s.last, s.has }

func (s *Series) Clone() Series {
	// value: a series of its own: what is added to it leaves s as it was.
	out := *s
	for i := range out.rings {
		out.rings[i].slots = append([]bucket(nil), s.rings[i].slots...)
	}
	return out
}

func SeriesOf(points []Sample) Series {
	// value: the series a list of samples, oldest first, rolls into.
	var s Series
	for _, x := range points {
		s.Add(x)
	}
	return s
}

func (s *Series) within(tier int, from, now time.Time) []bucket {
	// value: the tier's buckets from `from` to now, oldest first.
	if !s.has {
		return nil
	}
	r := &s.rings[tier]
	all := r.newest(now.UnixMilli())
	out := make([]bucket, 0, len(all))
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].at >= from.UnixMilli() {
			out = append(out, all[i])
		}
	}
	return out
}

func TierOf(window time.Duration) int {
	// value: the coarsest tier whose step divides the window's chart bar
	// and which keeps the whole window: T0 for 6 min, T2 for an hour, T3
	// for a day.
	every := BucketOf(window)
	pick := T0
	for i, t := range Tiers {
		if t.Every <= every && every%t.Every == 0 && t.Kept >= window {
			pick = i
		}
	}
	return pick
}

func (s *Series) Graph(window time.Duration, now time.Time) []Sample {
	// invariant: SPEC §6.4: the points a graph of the last window draws,
	// oldest first: one a bucket of the window's tier (TierOf).
	now = s.read(now)
	return s.points(TierOf(window), now.Add(-window), now)
}

func (s *Series) History(now time.Time) []Sample {
	// value: the last 24 h a script reads, one point a minute of T2.
	now = s.read(now)
	return s.points(T2, now.Add(-24*time.Hour), now)
}

func (s *Series) read(now time.Time) time.Time {
	// value: the time a read ends at: now, or the newest sample's when the
	// engine's clock is ahead of the reader's, so that sample is drawn.
	if s.has && s.last.At.After(now) {
		return s.last.At
	}
	return now
}

func (s *Series) points(tier int, from, now time.Time) []Sample {
	// invariant: one point a bucket, at its newest sample: its memory and
	// swap the bucket's means, its pressure the least it said; its
	// counters the newest sample's, less what each newer bucket's means
	// counted over the time they span, so two points apart rate what the
	// buckets measured. A point whose counter would go below zero is before
	// the machine counted: it and those older are dropped. A point carries
	// no load, thermal, battery nor free space: those are the newest
	// sample's alone.
	buckets := s.within(tier, from, now)
	if len(buckets) == 0 {
		return nil
	}
	out := make([]Sample, len(buckets))
	c := s.last
	cut := 0
	for i := len(buckets) - 1; i >= 0; i-- {
		b := &buckets[i]
		out[i] = Sample{
			At: time.UnixMilli(b.at).UTC(), CPUBusy: c.CPUBusy, CPUTotal: c.CPUTotal, CPUs: s.last.CPUs,
			MemUsed: int64(b.mem.mean), MemTotal: s.last.MemTotal, Pressure: pressureOf(b.floor), SwapUsed: int64(b.swap.mean),
			DiskRead: c.DiskRead, DiskWritten: c.DiskWritten, NetIn: c.NetIn, NetOut: c.NetOut,
			Thermal: ThermalUnknown, Boot: s.last.Boot,
		}
		span := float64(b.span)
		total := uint64(math.Round(span * s.ticks))
		busy := uint64(math.Round(float64(b.cpu.mean) / 100 * float64(total)))
		back := []struct {
			v *uint64
			d uint64
		}{
			{&c.CPUTotal, total}, {&c.CPUBusy, busy},
			{&c.DiskRead, uint64(math.Round(float64(b.diskRead.mean) * span))}, {&c.DiskWritten, uint64(math.Round(float64(b.diskWrite.mean) * span))},
			{&c.NetIn, uint64(math.Round(float64(b.netIn.mean) * span))}, {&c.NetOut, uint64(math.Round(float64(b.netOut.mean) * span))},
		}
		under := false
		for _, k := range back {
			if k.d > *k.v {
				under = true
			}
		}
		if under {
			cut = i
			break
		}
		for _, k := range back {
			*k.v -= k.d
		}
	}
	return out[cut:]
}

func (s *Series) held(tier int, holds func(prev, x *bucket) bool) (run []bucket, held time.Duration) {
	// value: how long, back from the newest bucket of the tier, every
	// bucket held against the one before it, and the buckets that held,
	// newest first; a Pause between two buckets ends the run.
	if !s.has {
		return nil, 0
	}
	all := s.rings[tier].newest(s.last.At.UnixMilli())
	if len(all) < 2 {
		return nil, 0
	}
	since := all[0].at
	for i := 0; i+1 < len(all); i++ {
		x, prev := &all[i], &all[i+1]
		if x.at-prev.at > Pause.Milliseconds() || !holds(prev, x) {
			break
		}
		run, since = all[:i+1], prev.at
	}
	return run, time.Duration(all[0].at-since) * time.Millisecond
}

func Recent(s *Series, now time.Time) (Rates, time.Duration) {
	// invariant: SPEC §6.2: the mean of the last RateWindow, from the T0
	// buckets of an engine a surface watches: the newest sample no older
	// than WatchedEvery and a second of jitter, the buckets after
	// RateWindow before it spanning RateWindow within WatchedEvery under
	// and twice over. Samples a minute apart, or a newest gone stale, span
	// no such window: no rate, and the reader samples twice itself.
	last, ok := s.Last()
	if !ok || now.Sub(last.At) > WatchedEvery+time.Second {
		return Rates{}, 0
	}
	var r Rates
	var span float64
	for _, b := range s.within(T0, last.At.Add(-RateWindow+time.Millisecond), last.At) {
		if b.span == 0 {
			continue
		}
		w := float64(b.span)
		r.CPU += float64(b.cpu.mean) * w
		r.DiskRead += float64(b.diskRead.mean) * w
		r.DiskWrite += float64(b.diskWrite.mean) * w
		r.NetIn += float64(b.netIn.mean) * w
		r.NetOut += float64(b.netOut.mean) * w
		span += w
	}
	window := time.Duration(span * float64(time.Second))
	if window < RateWindow-WatchedEvery || window >= RateWindow+2*WatchedEvery {
		return Rates{}, 0
	}
	return Rates{CPU: r.CPU / span, DiskRead: r.DiskRead / span, DiskWrite: r.DiskWrite / span, NetIn: r.NetIn / span, NetOut: r.NetOut / span}, window
}
