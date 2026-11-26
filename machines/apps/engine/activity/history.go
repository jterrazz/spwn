package activity

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
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

type Rollup [rollupLen]float64

// value: the numbers of a Rollup, one bucket of a tier as it is written:
// its newest sample's unix millisecond, the samples it counts, the seconds its
// rates span, the least pressure they said (0 none, 1 normal, 2 warn, 3
// critical); the CPU's mean and busiest share; the memory's and the swap's
// sum and busiest, in bytes; then the mean and the busiest of the disk
// read and write and the network in and out, in bytes a second; last the
// milliseconds from its slot's start to its CPU's busiest sample. A share
// or a rate is the float32 the tier holds, written in its shortest form.
const rollupLen = 19

type TierRollups struct {
	Every time.Duration `json:"every"`
	// value: the slots of a part's ring (Series.Part), none for a whole
	// tier's.
	Slots int `json:"slots,omitempty"`
	// value: the tier's buckets, oldest first, the slots nothing was
	// measured in left out.
	Buckets []Rollup `json:"buckets"`
	// invariant: T2's alone, one a bucket in the same order: what its
	// newest sample read that no other tier keeps.
	Gauges []Gauges `json:"gauges,omitempty"`
}

type Gauges struct {
	Load     [3]float64 `json:"load"`
	DiskFree int64      `json:"disk_free"`
	Battery  *Battery   `json:"battery,omitempty"`
}

type seriesJSON struct {
	// value: the newest samples the series keeps whole, oldest first.
	Recent []Sample      `json:"recent,omitempty"`
	Tiers  []TierRollups `json:"tiers"`
}

func (g gauges) wire() Gauges { return Gauges{Load: g.loads(), DiskFree: g.free, Battery: g.charge()} }

func (g Gauges) kept() gauges {
	return gaugesOf(&Sample{Load: g.Load, DiskFree: g.DiskFree, Battery: g.Battery})
}

func (b *bucket) row() Rollup {
	return Rollup{
		float64(b.at), float64(b.n()), float64(b.span), float64(b.floor()),
		float64(b.cpu.mean), float64(b.cpu.max),
		float64(b.mem.sum), float64(b.mem.max), float64(b.swap.sum), float64(b.swap.max),
		float64(b.diskRead.mean), float64(b.diskRead.max), float64(b.diskWrite.mean), float64(b.diskWrite.max),
		float64(b.netIn.mean), float64(b.netIn.max), float64(b.netOut.mean), float64(b.netOut.max),
		float64(b.peak()),
	}
}

func (r Rollup) bucket() (bucket, error) {
	if r[0] <= 0 || r[0] > math.MaxInt64 {
		return bucket{}, fmt.Errorf("activity: a bucket at %v", r[0])
	}
	f := func(i int) float32 { return float32(r[i]) }
	b := bucket{
		at: int64(r[0]), span: f(2),
		cpu: envelope{f(4), f(5)}, mem: gauge{int64(r[6]), int64(r[7])}, swap: gauge{int64(r[8]), int64(r[9])},
		diskRead: envelope{f(10), f(11)}, diskWrite: envelope{f(12), f(13)}, netIn: envelope{f(14), f(15)}, netOut: envelope{f(16), f(17)},
	}
	b.count(min(int(r[1]), maxCount), uint8(min(max(r[3], 0), 3)))
	b.peaked(int64(r[18]))
	return b, nil
}

// value: the fields of a Rollup a float32 holds; the others are integers.
var rollupFloats = [rollupLen]bool{2: true, 4: true, 5: true, 10: true, 11: true, 12: true, 13: true, 14: true, 15: true, 16: true, 17: true}

func (r Rollup) MarshalJSON() ([]byte, error) {
	out := make([]byte, 0, 16*rollupLen)
	out = append(out, '[')
	for i, x := range r {
		if i > 0 {
			out = append(out, ',')
		}
		bits := 64
		if rollupFloats[i] {
			bits = 32
		}
		out = strconv.AppendFloat(out, x, 'f', -1, bits)
	}
	return append(out, ']'), nil
}

func (s Series) MarshalJSON() ([]byte, error) {
	// invariant: the newest samples whole, then each tier's buckets, oldest
	// first, the slots nothing was measured in left out.
	out := seriesJSON{Tiers: []TierRollups{}}
	for i := max(0, s.count-recentKept); i < s.count; i++ {
		out.Recent = append(out.Recent, s.recent[i%recentKept])
	}
	for i, t := range Tiers {
		tier := TierRollups{Every: t.Every, Buckets: []Rollup{}}
		r := &s.rings[i]
		if n := len(r.slots); n > 0 && n < int(t.Kept/t.Every) {
			tier.Slots = n
		}
		if s.count > 0 {
			for _, b := range s.within(i, time.UnixMilli(0), s.newest().At) {
				tier.Buckets = append(tier.Buckets, b.row())
				if r.gauges != nil {
					tier.Gauges = append(tier.Gauges, r.gaugesAt(b.at).wire())
				}
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
	if len(in.Recent) == 0 {
		return nil
	}
	for _, x := range in.Recent {
		s.push(x)
	}
	var slots [len(Tiers)]int
	for i, t := range Tiers {
		slots[i] = int(t.Kept / t.Every)
	}
	of := make([]int, len(in.Tiers))
	for j, tier := range in.Tiers {
		i := slices.IndexFunc(Tiers[:], func(t Tier) bool { return t.Every == tier.Every })
		if i < 0 {
			return fmt.Errorf("activity: no tier of %v", tier.Every)
		}
		if tier.Slots > 0 {
			slots[i] = min(tier.Slots, slots[i])
		}
		of[j] = i
	}
	for i := range Tiers {
		s.rings[i] = ringOf(i, slots[i])
	}
	for j, tier := range in.Tiers {
		into := &s.rings[of[j]]
		if len(tier.Gauges) > 0 && (into.gauges == nil || len(tier.Gauges) != len(tier.Buckets)) {
			*s = Series{}
			return fmt.Errorf("activity: %d gauges for %d buckets of %v", len(tier.Gauges), len(tier.Buckets), tier.Every)
		}
		for k, r := range tier.Buckets {
			b, err := r.bucket()
			if err != nil {
				*s = Series{}
				return fmt.Errorf("activity: a tier of %v: %w", tier.Every, err)
			}
			*into.of(b.at) = b
			if len(tier.Gauges) > 0 {
				into.gauges[into.index(b.at)] = tier.Gauges[k].kept()
			}
		}
	}
	return nil
}
