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

func Fold(points []Sample, window time.Duration, now time.Time) Chart {
	// invariant: each share is measured between two points and falls in the
	// bucket of the later one; a share outside the window is not counted.
	every := BucketOf(window)
	n := max(int(window/every), 1)
	start := now.Add(-time.Duration(n) * every)
	w := Chart{Every: every, Buckets: make([]Bucket, n)}
	for i := range w.Buckets {
		w.Buckets[i].At = start.Add(time.Duration(i) * every)
	}
	for i := 1; i < len(points); i++ {
		at := points[i].At
		if at.Before(start) || !at.Before(start.Add(time.Duration(n)*every)) {
			continue
		}
		b := &w.Buckets[int(at.Sub(start)/every)]
		cpu := Between(points[i-1], points[i]).CPU
		b.Mean = (b.Mean*float64(b.Points) + cpu) / float64(b.Points+1)
		b.Points++
		if b.Points == 1 || cpu > b.Max {
			b.Max, b.MaxAt = cpu, at
		}
	}
	peaked(&w)
	return w
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

type measure struct {
	// value: a measure's mean and busiest in a bucket, and whether it is a
	// rate, weighted by the time it spans, or a gauge, by its samples.
	of    func(b *bucket) (mean, top float64)
	rated bool
}

func rated(of func(b *bucket) envelope) measure {
	return measure{of: func(b *bucket) (float64, float64) { e := of(b); return float64(e.mean), float64(e.max) }, rated: true}
}

func gauged(of func(b *bucket) gauge) measure {
	return measure{of: func(b *bucket) (float64, float64) { g := of(b); return float64(g.mean(b.n())), float64(g.max) }}
}

func (s *Series) fold(window time.Duration, now time.Time, m measure) (time.Duration, []Bucket) {
	// invariant: the bars are aligned on their span and end on the first
	// boundary at or after now, so a now on a boundary ends a whole bar as
	// Fold's do; a tier bucket falls in the bar its slot starts in, a rate
	// weighted by the time it spans, a gauge by its samples, and the
	// busiest is the busiest the tier kept, at that bucket's newest sample.
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
	tier := TierOf(window)
	tierStep := Tiers[tier].Every.Milliseconds()
	for _, b := range s.within(tier, time.UnixMilli(start), now) {
		from := b.at - b.at%tierStep
		if from < start || from >= end {
			continue
		}
		w := float64(b.n())
		if m.rated {
			w = float64(b.span)
		}
		if w == 0 {
			continue
		}
		mean, top := m.of(&b)
		o := &out[(from-start)/step]
		if o.Points == 0 || top > o.Max {
			o.Max, o.MaxAt = top, time.UnixMilli(b.at).UTC()
		}
		k := &weights[(from-start)/step]
		o.Mean = (o.Mean**k + mean*w) / (*k + w)
		*k += w
		o.Points += b.n()
	}
	return every, out
}

func (s *Series) Chart(window time.Duration, now time.Time) Chart {
	// value: SPEC B5.3: the window's CPU from its tier (TierOf), each bar
	// the mean and the busiest share measured in it, and its peak.
	every, buckets := s.fold(window, now, rated(func(b *bucket) envelope { return b.cpu }))
	w := Chart{Every: every, Buckets: buckets}
	peaked(&w)
	return w
}

func (s *Series) Measures(window time.Duration, now time.Time) Measures {
	bars := func(m measure) []Bucket {
		_, out := s.fold(window, now, m)
		return out
	}
	return Measures{
		Every:     BucketOf(window),
		Memory:    bars(gauged(func(b *bucket) gauge { return b.mem })),
		Swap:      bars(gauged(func(b *bucket) gauge { return b.swap })),
		DiskRead:  bars(rated(func(b *bucket) envelope { return b.diskRead })),
		DiskWrite: bars(rated(func(b *bucket) envelope { return b.diskWrite })),
		NetIn:     bars(rated(func(b *bucket) envelope { return b.netIn })),
		NetOut:    bars(rated(func(b *bucket) envelope { return b.netOut })),
	}
}
