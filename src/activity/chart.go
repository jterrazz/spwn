package activity

import "time"

type Bucket struct {
	// value: one span of a window as its CPU chart draws it: when it
	// starts, the mean and the busiest of the CPU shares measured in it (0 to
	// 100) and when the busiest was; Points zero is a gap, never an idle span.
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
	// value: the span a window's chart draws one bar of: 5 s over the 6 min
	// the fine ring holds (72), a minute up to an hour (60), else 30 min
	// (48 over 24 h).
	switch {
	case window <= FineEvery*FinePoints:
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
	w := Chart{Every: every, Buckets: make([]Bucket, n), Peak: -1}
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
	for i, b := range w.Buckets {
		if b.Points > 0 && (w.Peak < 0 || b.Max > w.Buckets[w.Peak].Max) {
			w.Peak = i
		}
	}
	if w.Peak >= 0 {
		w.PeakAt = w.Buckets[w.Peak].MaxAt
	}
	return w
}
