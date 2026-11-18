package activity

import "time"

const (
	FineEvery    = 2 * time.Second
	FinePoints   = 180
	CoarseEvery  = time.Minute
	CoarsePoints = 1440
)

type ring struct {
	points []Sample
	next   int
}

func (r *ring) add(x Sample, capacity int) {
	if len(r.points) < capacity {
		r.points = append(r.points, x)
		return
	}
	r.points[r.next] = x
	r.next = (r.next + 1) % capacity
}

func (r *ring) ordered() []Sample {
	out := make([]Sample, 0, len(r.points))
	out = append(out, r.points[r.next:]...)
	return append(out, r.points[:r.next]...)
}

func (r *ring) last() (Sample, bool) {
	if len(r.points) == 0 {
		return Sample{}, false
	}
	return r.points[(r.next+len(r.points)-1)%len(r.points)], true
}

type Series struct {
	// invariant: the engine's memory only: fine holds the last FinePoints
	// samples FineEvery apart (6 min, filled while a surface watches),
	// coarse the last CoarsePoints CoarseEvery apart (24 h, always).
	fine, coarse ring
}

func (s *Series) Add(x Sample) {
	// invariant: a sample closer to the last one than its ring's spacing
	// (less a second for the cadence's jitter) is not kept in that ring, so a
	// second surface asking does not shorten the window.
	keep := func(r *ring, every time.Duration) bool {
		last, ok := r.last()
		return !ok || x.At.Sub(last.At) >= every-time.Second
	}
	if keep(&s.fine, FineEvery) {
		s.fine.add(x, FinePoints)
	}
	if keep(&s.coarse, CoarseEvery) {
		s.coarse.add(x, CoarsePoints)
	}
}

func (s *Series) Fine() []Sample {
	// invariant: oldest first, and only the points of the fine window before
	// the newest: points left from a while unwatched are not the window.
	all := s.fine.ordered()
	if len(all) == 0 {
		return all
	}
	from := all[len(all)-1].At.Add(-FineEvery * FinePoints)
	for i, x := range all {
		if !x.At.Before(from) {
			return all[i:]
		}
	}
	return all
}

func (s *Series) Coarse() []Sample { return s.coarse.ordered() }

func (s *Series) Graph(window time.Duration, now time.Time) []Sample {
	// invariant: SPEC §6.4: the points a graph of the last window draws,
	// oldest first: the fine ring's 2 s points for a window it spans, else
	// the coarse ring's minutes.
	points := s.Coarse()
	if fine := s.Fine(); window <= FineEvery*FinePoints && len(fine) > 1 {
		points = fine
	}
	from := now.Add(-window)
	for i, x := range points {
		if !x.At.Before(from) {
			return points[i:]
		}
	}
	return nil
}

func (s *Series) Last() (Sample, bool) {
	fine, okFine := s.fine.last()
	coarse, okCoarse := s.coarse.last()
	if !okFine || (okCoarse && coarse.At.After(fine.At)) {
		return coarse, okCoarse
	}
	return fine, true
}
