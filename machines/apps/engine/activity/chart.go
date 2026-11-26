package activity

import "time"

type Bucket struct {
	// value: one span of a window as its chart draws it: when it starts,
	// the mean and the busiest of a measure in it (a CPU share 0 to 100,
	// bytes, or bytes a second) and when the busiest was; Points zero is a
	// gap, never an idle span.
	At     time.Time `json:"at"`
	Mean   float64   `json:"mean"`
	Max    float64   `json:"max"`
	MaxAt  time.Time `json:"max_at,omitzero"`
	Points int       `json:"points"`
}

type Chart struct {
	// value: SPEC B5.3: the window's CPU folded into buckets of Every, oldest
	// first, and its peak: the bucket whose busiest point is the busiest of
	// all and that point's time; Peak -1 when no bucket holds a point.
	Every   time.Duration `json:"every"`
	Buckets []Bucket      `json:"buckets"`
	Peak    int           `json:"peak"`
	PeakAt  time.Time     `json:"peak_at,omitzero"`
}

func BucketOf(window time.Duration) time.Duration {
	// value: the span a window's chart draws one bar of: 5 s up to
	// LiveWindow (72 over 6 min), a minute up to an hour (60), else 30 min
	// (48 over 24 h).
	switch {
	case window <= LiveWindow:
		return 5 * time.Second
	case window <= time.Hour:
		return time.Minute
	}
	return 30 * time.Minute
}

func peaked(w *Chart) {
	w.Peak = -1
	for i, b := range w.Buckets {
		if b.Points > 0 && (w.Peak < 0 || b.Max > w.Buckets[w.Peak].Max) {
			w.Peak = i
		}
	}
	if w.Peak >= 0 {
		w.PeakAt = w.Buckets[w.Peak].MaxAt
	}
}

type Measures struct {
	// value: ADR-107: the window's other measures, folded from the same
	// tier and into the same bars as its CPU chart, each bar the mean and
	// the busiest of what was measured in it: memory and swap used in
	// bytes, disk and network in bytes a second.
	Every     time.Duration `json:"every"`
	Memory    []Bucket      `json:"memory"`
	Swap      []Bucket      `json:"swap"`
	DiskRead  []Bucket      `json:"disk_read"`
	DiskWrite []Bucket      `json:"disk_write"`
	NetIn     []Bucket      `json:"net_in"`
	NetOut    []Bucket      `json:"net_out"`
}

type measure func(p *Point) (r Range, at time.Time, weight float64)

func rated(of func(r *Rated) Range) measure {
	// value: a rate, weighted by the time it spans; a point that rated
	// nothing weighs nothing.
	return func(p *Point) (Range, time.Time, float64) {
		if p.Rated == nil {
			return Range{}, time.Time{}, 0
		}
		return of(p.Rated), p.At, p.Rated.Span
	}
}

func gauged(of func(p *Point) Range) measure {
	// value: a gauge, weighted by the samples it counts.
	return func(p *Point) (Range, time.Time, float64) { return of(p), p.At, float64(p.Samples) }
}

func cpu(p *Point) (Range, time.Time, float64) {
	if p.Rated == nil {
		return Range{}, time.Time{}, 0
	}
	return p.Rated.CPU, p.Rated.CPUMaxAt, p.Rated.Span
}

func fold(points []Point, window time.Duration, now time.Time, m measure) []Bucket {
	// invariant: the one fold of a window, the engine's and the menu bar's:
	// the bars are aligned on their span and end on the first boundary at
	// or after now, so a now on a boundary ends a whole bar; a point falls
	// in the bar its newest sample is in, a bar's mean is its points' means
	// weighted as m weighs them, its busiest the busiest its points kept, at
	// its own sample's time for the CPU, at the point's for the others.
	every := BucketOf(window)
	n := max(int(window/every), 1)
	step := every.Milliseconds()
	end := (now.UnixMilli() + step - 1) / step * step
	start := end - int64(n)*step
	out := make([]Bucket, n)
	weights := make([]float64, n)
	for i := range out {
		out[i].At = time.UnixMilli(start + int64(i)*step).UTC()
	}
	for i := range points {
		p := &points[i]
		at := p.At.UnixMilli()
		if at < start || at >= end {
			continue
		}
		r, top, w := m(p)
		if w == 0 {
			continue
		}
		k := (at - start) / step
		o := &out[k]
		if o.Points == 0 || r.Max > o.Max {
			o.Max, o.MaxAt = r.Max, top
		}
		o.Mean = (o.Mean*weights[k] + r.Mean*w) / (weights[k] + w)
		weights[k] += w
		o.Points += p.Samples
	}
	return out
}

func ChartOf(points []Point, window time.Duration, now time.Time) Chart {
	// value: SPEC B5.3: the window's CPU, each bar the mean and the busiest
	// share measured in it, and its peak.
	w := Chart{Every: BucketOf(window), Buckets: fold(points, window, now, cpu)}
	peaked(&w)
	return w
}

func MeasuresOf(points []Point, window time.Duration, now time.Time) Measures {
	return Measures{
		Every:     BucketOf(window),
		Memory:    fold(points, window, now, gauged(func(p *Point) Range { return p.Memory })),
		Swap:      fold(points, window, now, gauged(func(p *Point) Range { return p.Swap })),
		DiskRead:  fold(points, window, now, rated(func(r *Rated) Range { return r.DiskRead })),
		DiskWrite: fold(points, window, now, rated(func(r *Rated) Range { return r.DiskWrite })),
		NetIn:     fold(points, window, now, rated(func(r *Rated) Range { return r.NetIn })),
		NetOut:    fold(points, window, now, rated(func(r *Rated) Range { return r.NetOut })),
	}
}

func (s *Series) charted(window time.Duration, now time.Time) []Point {
	// value: the points of the window's tier (TierOf) its bars can hold.
	return s.points(TierOf(window), now.Add(-window-BucketOf(window)), now)
}

func (s *Series) Chart(window time.Duration, now time.Time) Chart {
	return ChartOf(s.charted(window, now), window, now)
}

func (s *Series) Measures(window time.Duration, now time.Time) Measures {
	return MeasuresOf(s.charted(window, now), window, now)
}
