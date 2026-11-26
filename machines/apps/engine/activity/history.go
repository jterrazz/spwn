package activity

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"
)

type History struct {
	mu    sync.Mutex
	since time.Time
	kept  map[string]*Kept
}

type Kept struct {
	Last   Reading `json:"last"`
	Series Series  `json:"series"`
	// value: the subjects of the broken findings already told the owner.
	Told []string `json:"told,omitempty"`
}

func NewHistory(since time.Time) *History {
	// invariant: the history of the engine that started at since, in its
	// memory alone: an engine that starts again starts it again.
	return &History{since: since, kept: map[string]*Kept{}}
}

func (h *History) Since() time.Time { return h.since }

func (h *History) Add(machine string, r Reading) {
	// invariant: a VM read from its Mac while its guest is silent knows none
	// of its own counters: its series keeps a gap there, never zeros.
	h.mu.Lock()
	defer h.mu.Unlock()
	k := h.kept[machine]
	if k == nil {
		k = &Kept{}
		h.kept[machine] = k
	}
	k.Last = r
	if r.Run == nil {
		k.Series.Add(r.Sample)
	}
}

func (h *History) Of(machine string) (Kept, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	k, ok := h.kept[machine]
	if !ok {
		return Kept{}, false
	}
	return Kept{Last: k.Last, Series: k.Series.Clone(), Told: slices.Clone(k.Told)}, true
}

func (h *History) Raise(machine string, broken []string) []string {
	// invariant: a broken finding is told once, until it clears: what it
	// answers are the subjects broken now that were not at the last call.
	h.mu.Lock()
	defer h.mu.Unlock()
	k := h.kept[machine]
	if k == nil {
		return nil
	}
	var fresh []string
	for _, subject := range broken {
		if !slices.Contains(k.Told, subject) {
			fresh = append(fresh, subject)
		}
	}
	k.Told = slices.Clone(broken)
	return fresh
}

func (h *History) Machines() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	names := make([]string, 0, len(h.kept))
	for name := range h.kept {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func (h *History) Hand() map[string]Kept {
	// value: what a stepped engine hands its successor in the same process
	// lifetime, as a real one keeps it between two ticks.
	out := map[string]Kept{}
	for _, name := range h.Machines() {
		out[name], _ = h.Of(name)
	}
	return out
}

func (h *History) Take(kept map[string]Kept) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for name, k := range kept {
		h.kept[name] = &Kept{Last: k.Last, Series: k.Series.Clone(), Told: slices.Clone(k.Told)}
	}
}

func (h *History) Read(machine string, read func(k *Kept)) bool {
	// value: what the engine keeps of a machine, read in place: no copy of
	// its tiers, for a reader that keeps nothing of it.
	h.mu.Lock()
	defer h.mu.Unlock()
	k, ok := h.kept[machine]
	if ok {
		read(k)
	}
	return ok
}

type Envelope struct {
	Mean float64 `json:"mean"`
	Max  float64 `json:"max"`
}

type Rollup struct {
	// value: one bucket of a tier as it is written: its newest sample's
	// time, the samples it counts, the seconds its rates span, the least
	// pressure they said, and each measure's mean and busiest.
	At        time.Time `json:"at"`
	Points    int       `json:"points"`
	Span      float64   `json:"span"`
	Pressure  Pressure  `json:"pressure,omitempty"`
	CPU       Envelope  `json:"cpu"`
	Memory    Envelope  `json:"memory"`
	Swap      Envelope  `json:"swap"`
	DiskRead  Envelope  `json:"disk_read"`
	DiskWrite Envelope  `json:"disk_write"`
	NetIn     Envelope  `json:"net_in"`
	NetOut    Envelope  `json:"net_out"`
}

type TierRollups struct {
	Every   time.Duration `json:"every"`
	Buckets []Rollup      `json:"buckets"`
}

type seriesJSON struct {
	Last  *Sample       `json:"last,omitempty"`
	Ticks float64       `json:"ticks,omitempty"`
	Tiers []TierRollups `json:"tiers"`
}

func (e envelope) out() Envelope { return Envelope{Mean: float64(e.mean), Max: float64(e.max)} }

func (e Envelope) in() envelope { return envelope{mean: float32(e.Mean), max: float32(e.Max)} }

func (s Series) MarshalJSON() ([]byte, error) {
	// invariant: each tier's buckets, oldest first, the slots nothing was
	// measured in left out.
	out := seriesJSON{Tiers: []TierRollups{}}
	if s.has {
		out.Last, out.Ticks = &s.last, s.ticks
	}
	for i, t := range Tiers {
		tier := TierRollups{Every: t.Every, Buckets: []Rollup{}}
		if s.has {
			buckets := s.rings[i].newest(s.last.At.UnixMilli())
			for j := len(buckets) - 1; j >= 0; j-- {
				b := buckets[j]
				tier.Buckets = append(tier.Buckets, Rollup{
					At: time.UnixMilli(b.at).UTC(), Points: int(b.n), Span: float64(b.span), Pressure: pressureOf(b.floor),
					CPU: b.cpu.out(), Memory: b.mem.out(), Swap: b.swap.out(), DiskRead: b.diskRead.out(), DiskWrite: b.diskWrite.out(), NetIn: b.netIn.out(), NetOut: b.netOut.out(),
				})
			}
		}
		out.Tiers = append(out.Tiers, tier)
	}
	return json.Marshal(out)
}

func (s *Series) UnmarshalJSON(data []byte) error {
	// invariant: a tier whose step is not one of Tiers is refused: the
	// buckets of another step would not fall in their slots.
	var in seriesJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	*s = Series{}
	if in.Last == nil {
		return nil
	}
	s.last, s.has, s.ticks = *in.Last, true, in.Ticks
	for i, t := range Tiers {
		s.rings[i] = ring{every: t.Every.Milliseconds(), slots: make([]bucket, int(t.Kept/t.Every))}
	}
	for _, tier := range in.Tiers {
		i := slices.IndexFunc(Tiers[:], func(t Tier) bool { return t.Every == tier.Every })
		if i < 0 {
			return fmt.Errorf("activity: no tier of %v", tier.Every)
		}
		for _, r := range tier.Buckets {
			at := r.At.UnixMilli()
			*s.rings[i].of(at) = bucket{
				at: at, span: float32(r.Span), n: uint16(min(r.Points, math.MaxUint16)), floor: pressureRank(r.Pressure), //nolint:gosec // value: capped to its range.
				cpu: r.CPU.in(), mem: r.Memory.in(), swap: r.Swap.in(), diskRead: r.DiskRead.in(), diskWrite: r.DiskWrite.in(), netIn: r.NetIn.in(), netOut: r.NetOut.in(),
			}
		}
	}
	return nil
}
