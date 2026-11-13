package activity

import (
	"encoding/json"
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
	return Kept{Last: k.Last, Series: k.Series.clone(), Told: slices.Clone(k.Told)}, true
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
		h.kept[name] = &Kept{Last: k.Last, Series: k.Series.clone(), Told: slices.Clone(k.Told)}
	}
}

func (s *Series) clone() Series {
	return Series{fine: ring{points: slices.Clone(s.fine.points), next: s.fine.next}, coarse: ring{points: slices.Clone(s.coarse.points), next: s.coarse.next}}
}

type seriesJSON struct {
	Fine   []Sample `json:"fine"`
	Coarse []Sample `json:"coarse"`
}

func (s Series) MarshalJSON() ([]byte, error) {
	return json.Marshal(seriesJSON{Fine: s.Fine(), Coarse: s.Coarse()})
}

func (s *Series) UnmarshalJSON(data []byte) error {
	var in seriesJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	*s = Series{}
	for _, x := range in.Fine {
		s.fine.add(x, FinePoints)
	}
	for _, x := range in.Coarse {
		s.coarse.add(x, CoarsePoints)
	}
	return nil
}
