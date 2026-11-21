package feeds

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
)

// value: the window's views, each named as the call whose answer it is:
// the machines list, This Mac's Overview, This Mac's with the other Macs' roll-up
// in its verdict, and one machine's (`overview.read:<machine>`, every
// machine's as `overview.read:*`).
const (
	ViewMachines          = "machines.list"
	ViewOverview          = "overview.read"
	ViewOverviewElsewhere = "overview.read+elsewhere"
	viewOverviewOf        = ViewOverview + ":"
)

type MachinesView struct {
	// value: the answer the machines.list view holds, the call's own out.
	All      machines.AllMachines
	Rooms    []machines.Room
	Roomiest string
}

type OverviewView struct {
	// value: the answer an overview.read view holds, the call's own out.
	Summary overview.Summary
}

func (o Pages) Known(name string) bool {
	switch name {
	case ViewMachines, ViewOverview, ViewOverviewElsewhere:
		return true
	}
	machine, ok := strings.CutPrefix(name, viewOverviewOf)
	if !ok {
		return false
	}
	_, err := store.ParseKey(string(store.Of(store.FamilyOverview, machine)))
	return machine == "*" || err == nil
}

func (o Pages) Project(s *store.Snapshot, asked func(name string) bool) map[string]store.View {
	// invariant: ADR-070: a view is a pure function of one snapshot, as a
	// page is, its words said as of the snapshot's instant; the instants it
	// carries are those its facts were read at, never the clock, so a
	// revision that only moves the clock projects the same bytes.
	v := o.of(s)
	out := map[string]store.View{}
	all := v.machines()
	own := s.Slices[store.Of(store.FamilyMac, v.this)]
	if asked(ViewMachines) {
		listed := *all
		listed.Read = own.ReadAt
		out[ViewMachines] = viewOf(own, MachinesView{All: listed, Rooms: machines.Rooms(listed), Roomiest: machines.Roomiest(listed)})
	}
	if sl, sum, ok := v.overviewOf(v.this); ok {
		if asked(ViewOverview) {
			out[ViewOverview] = viewOf(sl, OverviewView{Summary: sum})
		}
		if asked(ViewOverviewElsewhere) {
			others := slices.DeleteFunc(slices.Clone(all.Macs), func(h machines.MacView) bool { return h.ThisMac })
			roll := machines.RollUpOf(others)
			sum.Verdict = overview.VerdictOf(sum, &roll)
			out[ViewOverviewElsewhere] = viewOf(sl, OverviewView{Summary: sum})
		}
	}
	for k := range s.Slices {
		name := viewOverviewOf + k.Machine()
		if k.Family() != store.FamilyOverview || !asked(name) {
			continue
		}
		if sl, sum, ok := v.overviewOf(k.Machine()); ok {
			out[name] = viewOf(sl, OverviewView{Summary: sum})
		}
	}
	return out
}

// value: a machine's Overview as of its own reading: its uptime and the
// head said from it are those of that instant.
func (v view) overviewOf(name string) (store.Slice, overview.Summary, bool) {
	kept, sl, ok := value[overview.Kept](v.s, store.Of(store.FamilyOverview, name))
	if !ok {
		return sl, overview.Summary{}, false
	}
	return sl, kept.At(sl.ReadAt, sl.ReadAt), true
}

func viewOf(sl store.Slice, value any) store.View {
	state := sl.State
	if state == "" {
		state = store.Unread
	}
	return store.View{Value: value, ReadAt: sl.ReadAt, State: state, Why: sl.Why}
}
