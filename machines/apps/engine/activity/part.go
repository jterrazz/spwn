package activity

import (
	"slices"
	"time"
)

type Want struct {
	// value: what a reader of a machine's history reads of it: the window
	// its graphs draw (none, no graph) and whether it prints the 24 h a
	// script reads.
	Graph   time.Duration
	History bool
}

func (s *Series) Part(w Want, now time.Time) Series {
	// invariant: the part reads as the whole for w at now: its newest
	// samples, the buckets of each tier a graph, the 24 h and the findings
	// read, and room for the samples a reader adds up to a Pause past now;
	// each ring only as long as that, so nothing else is copied.
	out := Series{recent: s.recent, count: s.count, ticks: s.ticks}
	if s.count == 0 {
		return out
	}
	newest := s.newest().At.UnixMilli()
	from := [len(Tiers)]int64{}
	for i := range from {
		from[i] = newest
	}
	reach := func(tier int, at int64) { from[tier] = min(from[tier], at) }
	if w.Graph > 0 {
		reach(TierOf(w.Graph), now.UnixMilli()-(w.Graph+BucketOf(w.Graph)).Milliseconds())
	}
	if w.History {
		reach(T2, now.UnixMilli()-(24*time.Hour).Milliseconds())
	}
	reach(T0, s.reach(T0, pressureHeld))
	reach(T2, s.reach(T2, swapGrowing))
	reach(T2, s.reach(T2, busy))
	end := max(now.UnixMilli(), newest) + Pause.Milliseconds()
	for i := range s.rings {
		r := &s.rings[i]
		n := min(int64(len(r.slots)), (end-from[i])/r.every+2)
		part := ring{every: r.every, slots: make([]bucket, n)}
		for b := range r.back(newest) {
			if b.at < from[i] {
				break
			}
			*part.of(b.at) = *b
		}
		out.rings[i] = part
	}
	return out
}

func (s *Series) reach(tier int, holds func(prev, x *bucket) bool) int64 {
	// value: the oldest bucket of the tier held reads with holds: its
	// run's, and the one that ended it.
	oldest := s.newest().At.UnixMilli()
	var x *bucket
	for b := range s.rings[tier].back(oldest) {
		oldest = b.at
		if x != nil && (x.at-b.at > Pause.Milliseconds() || !holds(b, x)) {
			break
		}
		x = b
	}
	return oldest
}

func (h *History) Part(machine string, w Want, now time.Time) (Kept, bool) {
	// value: Of, but only the part of the series w reads at now, copied
	// under the lock from the series in place.
	h.mu.Lock()
	defer h.mu.Unlock()
	k, ok := h.kept[machine]
	if !ok {
		return Kept{}, false
	}
	return Kept{Last: k.Last, Series: k.Series.Part(w, now), Told: slices.Clone(k.Told)}, true
}
