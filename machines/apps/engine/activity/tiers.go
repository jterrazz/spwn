package activity

import (
	"iter"
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

// value: the newest samples a series keeps whole, outside its tiers, so
// the means of the last RateWindow are the counters' own (Recent): the
// RateWindow and twice WatchedEvery of a sample a second.
const recentKept = 16

type envelope struct{ mean, max float32 }

func (e *envelope) rate(x, span, dt float64) {
	if span == 0 || float32(x) > e.max {
		e.max = float32(x)
	}
	e.mean = float32((float64(e.mean)*span + x*dt) / (span + dt))
}

type gauge struct {
	// invariant: bytes, exact: the sum and the most of the readings a
	// bucket counts, so its mean is their sum over its samples.
	sum, max int64
}

func (g *gauge) add(x int64, n int) {
	if n == 1 || x > g.max {
		g.max = x
	}
	g.sum += x
}

func (g gauge) mean(n int) int64 {
	if n == 0 {
		return 0
	}
	return int64(math.Round(float64(g.sum) / float64(n)))
}

const (
	// value: a bucket's meta, low bits first: 20 bits left for the time of
	// its CPU's busiest sample, then the samples it counts and the least
	// pressure they said.
	countShift = 20
	peakMask   = 1<<countShift - 1
	countBits  = 10
	floorShift = countShift + countBits
	// value: the samples a bucket counts at most: 15 min at a sample a
	// second is 900, and ADR-107 refuses sub-second sampling.
	maxCount = 1<<countBits - 1
)

type bucket struct {
	// invariant: at is the unix millisecond of the bucket's newest sample,
	// 0 for a slot nothing was measured in; span the seconds its rates
	// measure; meta the samples it counts and the least pressure they said.
	at   int64
	span float32
	meta uint32

	mem, swap                               gauge
	cpu, diskRead, diskWrite, netIn, netOut envelope
}

func (b *bucket) n() int { return int(b.meta >> countShift & maxCount) }

func (b *bucket) floor() uint8 { return uint8(b.meta >> floorShift) } //nolint:gosec // value: the top 2 bits.

func (b *bucket) count(n int, floor uint8) {
	b.meta = b.meta&peakMask | uint32(n)<<countShift | uint32(floor)<<floorShift //nolint:gosec // value: n is at most maxCount.
}

func (b *bucket) peak() int64 { return int64(b.meta & peakMask) }

func (b *bucket) peaked(offset int64) {
	// invariant: the milliseconds from the bucket's slot start to its CPU's
	// busiest sample: a 15 min slot is 900,000, under the 20 bits.
	b.meta = b.meta&^peakMask | uint32(min(max(offset, 0), peakMask)) //nolint:gosec // value: clamped to the mask.
}

func (b *bucket) peakAt(every int64) int64 { return b.at - b.at%every + b.peak() }

type gauges struct {
	// invariant: what a T2 bucket's newest sample read and no other tier
	// keeps: its free disk bytes, its load in hundredths, its battery a word
	// (0 none; else 1<<15, on battery 1<<9, charging 1<<8, the percent).
	free    int64
	load    [3]uint32
	battery uint16
}

const (
	batteryHeld     = 1 << 15
	batteryOn       = 1 << 9
	batteryCharging = 1 << 8
)

func gaugesOf(x *Sample) gauges {
	g := gauges{free: x.DiskFree}
	for i, l := range x.Load {
		g.load[i] = uint32(min(max(math.Round(l*100), 0), math.MaxUint32))
	}
	if b := x.Battery; b != nil {
		g.battery = batteryHeld | uint16(min(max(b.Percent, 0), 255)) //nolint:gosec // value: clamped to a byte.
		if b.OnBattery {
			g.battery |= batteryOn
		}
		if b.Charging {
			g.battery |= batteryCharging
		}
	}
	return g
}

func (g gauges) loads() [3]float64 {
	return [3]float64{float64(g.load[0]) / 100, float64(g.load[1]) / 100, float64(g.load[2]) / 100}
}

func (g gauges) charge() *Battery {
	if g.battery&batteryHeld == 0 {
		return nil
	}
	return &Battery{Percent: int(g.battery & 0xff), OnBattery: g.battery&batteryOn != 0, Charging: g.battery&batteryCharging != 0}
}

type ring struct {
	every int64
	slots []bucket
	// invariant: T2's alone, one per slot, nil on every other tier.
	gauges []gauges
}

func ringOf(tier, slots int) ring {
	r := ring{every: Tiers[tier].Every.Milliseconds(), slots: make([]bucket, slots)}
	if tier == T2 {
		r.gauges = make([]gauges, slots)
	}
	return r
}

func (r *ring) index(at int64) int64 { return (at / r.every) % int64(len(r.slots)) }

func (r *ring) of(at int64) *bucket { return &r.slots[r.index(at)] }

func (r *ring) gaugesAt(at int64) gauges {
	if r.gauges == nil {
		return gauges{}
	}
	return r.gauges[r.index(at)]
}

func (r *ring) add(at int64, x *Sample, rated *Rates, dt float64) {
	// invariant: a bucket that counts maxCount samples keeps none more.
	b := r.of(at)
	if b.at == 0 || b.at/r.every != at/r.every {
		*b = bucket{}
	}
	n := b.n()
	if n == maxCount {
		return
	}
	n++
	b.at = at
	if r.gauges != nil {
		r.gauges[r.index(at)] = gaugesOf(x)
	}
	floor := b.floor()
	if rank := pressureRank(x.Pressure); n == 1 || rank < floor {
		floor = rank
	}
	b.count(n, floor)
	b.mem.add(x.MemUsed, n)
	b.swap.add(x.SwapUsed, n)
	if rated == nil {
		return
	}
	span := float64(b.span)
	if span == 0 || float32(rated.CPU) > b.cpu.max {
		b.peaked(at % r.every)
	}
	b.cpu.rate(rated.CPU, span, dt)
	b.diskRead.rate(rated.DiskRead, span, dt)
	b.diskWrite.rate(rated.DiskWrite, span, dt)
	b.netIn.rate(rated.NetIn, span, dt)
	b.netOut.rate(rated.NetOut, span, dt)
	b.span = float32(span + dt)
}

func (r *ring) back(at int64) iter.Seq[*bucket] {
	// value: the buckets of this ring from the slot of at back to its
	// oldest, newest first, in place, those of a slot that is not theirs
	// skipped.
	return func(yield func(*bucket) bool) {
		if len(r.slots) == 0 {
			return
		}
		slot := at / r.every
		for i := range int64(len(r.slots)) {
			if b := r.of((slot - i) * r.every); b.at != 0 && b.at/r.every == slot-i && !yield(b) {
				return
			}
		}
	}
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
	// each tier's ring of fixed slots, indexed by the sample's own time, so a
	// tier keeps at most its Kept and a slot no sample reached is a gap.
	// recent holds the newest recentKept samples whole, count the samples
	// added.
	recent [recentKept]Sample
	count  int
	rings  [len(Tiers)]ring
}

func countedAgain(a, b *Sample) bool {
	// value: the machine counted again from zero between a and b: a
	// counter went back, or it booted again.
	return b.CPUTotal < a.CPUTotal || b.CPUBusy < a.CPUBusy || b.DiskRead < a.DiskRead || b.DiskWritten < a.DiskWritten ||
		b.NetIn < a.NetIn || b.NetOut < a.NetOut || (!a.Boot.IsZero() && !b.Boot.IsZero() && !a.Boot.Equal(b.Boot))
}

func (s *Series) newest() *Sample { return &s.recent[(s.count-1)%recentKept] }

func (s *Series) push(x Sample) {
	s.recent[s.count%recentKept] = x
	s.count++
}

func (s *Series) Add(x Sample) {
	// invariant: a sample no later than the newest is not kept; a rate is
	// measured between a sample and the one before it, unless a Pause or a
	// machine that counted again lies between them: the sample then counts
	// its gauges only.
	if s.count > 0 && !x.At.After(s.newest().At) {
		return
	}
	if s.rings[T0].slots == nil {
		for i, t := range Tiers {
			s.rings[i] = ringOf(i, int(t.Kept/t.Every))
		}
	}
	var rated *Rates
	var dt float64
	if s.count > 0 {
		last := s.newest()
		if gap := x.At.Sub(last.At); gap <= Pause && !countedAgain(last, &x) {
			r := Between(*last, x)
			rated, dt = &r, gap.Seconds()
		}
	}
	at := x.At.UnixMilli()
	for i := range s.rings {
		s.rings[i].add(at, &x, rated, dt)
	}
	s.push(x)
}

func (s *Series) Last() (Sample, bool) {
	if s.count == 0 {
		return Sample{}, false
	}
	return *s.newest(), true
}

func (s *Series) Clone() Series {
	// value: a series of its own: what is added to it leaves s as it was.
	out := *s
	for i := range out.rings {
		out.rings[i].slots = append([]bucket(nil), s.rings[i].slots...)
		if s.rings[i].gauges != nil {
			out.rings[i].gauges = append([]gauges(nil), s.rings[i].gauges...)
		}
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
	if s.count == 0 {
		return nil
	}
	var out []bucket
	for b := range s.rings[tier].back(now.UnixMilli()) {
		if b.at < from.UnixMilli() {
			break
		}
		out = append(out, *b)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
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

func (s *Series) Graph(window time.Duration, now time.Time) []Point {
	// invariant: SPEC §6.4: the points a graph of the last window draws,
	// oldest first: one a bucket of the window's tier (TierOf).
	now = s.read(now)
	return s.points(TierOf(window), now.Add(-window), now)
}

func (s *Series) History(now time.Time) []Point {
	// value: the last 24 h a script reads, one point a minute of T2.
	now = s.read(now)
	return s.points(T2, now.Add(-24*time.Hour), now)
}

func (s *Series) read(now time.Time) time.Time {
	// value: the time a read ends at: now, or the newest sample's when the
	// engine's clock is ahead of the reader's, so that sample is drawn.
	if s.count > 0 && s.newest().At.After(now) {
		return s.newest().At
	}
	return now
}

func (s *Series) points(tier int, from, now time.Time) []Point {
	// invariant: one point a bucket, built from that bucket alone, so a
	// pause or a counter that went back is a gap at its own bucket and the
	// points before it stay.
	buckets := s.within(tier, from, now)
	if len(buckets) == 0 {
		return nil
	}
	last := s.newest()
	r := &s.rings[tier]
	out := make([]Point, len(buckets))
	for i := range buckets {
		out[i] = buckets[i].point(r.every, r.gaugesAt(buckets[i].at), r.gauges != nil, last)
	}
	return out
}

func (s *Series) held(tier int, holds func(prev, x *bucket) bool) (run []bucket, held time.Duration) {
	// value: how long, back from the newest bucket of the tier, every
	// bucket held against the one before it, and the buckets that held,
	// newest first; a Pause between two buckets ends the run.
	if s.count == 0 {
		return nil, 0
	}
	var x *bucket
	var first, since int64
	for b := range s.rings[tier].back(s.newest().At.UnixMilli()) {
		if x == nil {
			x, first, since = b, b.at, b.at
			continue
		}
		if x.at-b.at > Pause.Milliseconds() || !holds(b, x) {
			break
		}
		run, since, x = append(run, *x), b.at, b
	}
	return run, time.Duration(first-since) * time.Millisecond
}

func Recent(s *Series, now time.Time) (Rates, time.Duration) {
	// invariant: SPEC §6.2: the mean of the last RateWindow, between the
	// newest sample of an engine a surface watches, no older than
	// WatchedEvery and a second of jitter, and the kept sample nearest
	// RateWindow before it, within WatchedEvery under and twice over (a
	// sample a cadence skipped), both read whole. Samples a minute apart, or
	// a newest gone stale, span no such window: no rate, and the reader
	// samples twice itself.
	if s.count == 0 {
		return Rates{}, 0
	}
	newest := s.newest()
	if now.Sub(newest.At) > WatchedEvery+time.Second {
		return Rates{}, 0
	}
	var base *Sample
	for i := 1; i < min(s.count, recentKept); i++ {
		x := &s.recent[(s.count-1-i)%recentKept]
		gap := newest.At.Sub(x.At)
		if gap < RateWindow-WatchedEvery {
			continue
		}
		if gap >= RateWindow+2*WatchedEvery {
			break
		}
		if base == nil || (gap-RateWindow).Abs() < (newest.At.Sub(base.At)-RateWindow).Abs() {
			base = x
		}
	}
	if base == nil {
		return Rates{}, 0
	}
	return Between(*base, *newest), newest.At.Sub(base.At)
}
