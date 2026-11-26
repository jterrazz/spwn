package feeds

import (
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

// value: the window's views, each named as the call whose answer it is:
// the machines list, This Mac's Overview, This Mac's with the other Macs' roll-up
// in its verdict, and one machine's (`overview.read:<machine>`, every
// machine's as `overview.read:*`); every Mac's ports, and one machine's
// (`ports.list:<machine>`); This Mac's sign-ins, and another machine's
// (`accounts.list:<machine>`); This Mac's Awake, and another Mac's
// (`awake.read:<mac>`).
const (
	ViewMachines          = "machines.list"
	ViewOverview          = "overview.read"
	ViewOverviewElsewhere = "overview.read+elsewhere"
	viewOverviewOf        = ViewOverview + ":"
	ViewPorts             = "ports.list+all-machines"
	viewPortsOf           = "ports.list:"
	ViewAccounts          = "accounts.list"
	viewAccountsOf        = ViewAccounts + ":"
	ViewAwake             = "awake.read"
	viewAwakeOf           = ViewAwake + ":"
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
	case ViewMachines, ViewOverview, ViewOverviewElsewhere, ViewPorts, ViewAccounts, ViewAwake, ViewGuides:
		return true
	}
	for _, kind := range []string{viewOverviewOf, viewPortsOf, viewAccountsOf, viewAwakeOf} {
		if machine, ok := strings.CutPrefix(name, kind); ok {
			_, err := store.ParseKey(string(store.Of(store.FamilyOverview, machine)))
			return machine == "*" || err == nil
		}
	}
	return false
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
	if asked(ViewGuides) {
		out[ViewGuides] = v.guides(all)
	}
	v.portsViews(all, asked, out)
	v.ownViews(asked, out)
	return out
}

// value: every Mac's ports as ports.list composes them with This Mac's
// node on the tailnet, and each machine's alone, as of their readings.
func (v view) portsViews(all *machines.AllMachines, asked func(name string) bool, out map[string]store.View) {
	sl := v.s.Slices[store.Of(store.FamilyPorts, v.this)]
	every := v.ports(all)
	if every == nil {
		return
	}
	node, _, _ := value[ports.Status](v.s, store.Of(store.FamilyNode, ""))
	// invariant: OwnDevices marks the ports it is given: each Mac's are
	// copied first, so the snapshot's values never move.
	for i := range every.Macs {
		every.Macs[i].Ports = slices.Clone(every.Macs[i].Ports)
	}
	listed := ports.OwnDevices(*every, node.Peers)
	if asked(ViewPorts) {
		out[ViewPorts] = viewOf(sl, ports.Listing{AllMachines: listed, Tailnet: node, Verdict: ports.VerdictOf(listed)})
	}
	for _, name := range machineNames(all) {
		if !asked(viewPortsOf + name) {
			continue
		}
		l := ports.Listing{AllMachines: ports.Of(listed, name), Tailnet: node}
		if len(l.AllMachines.Macs) == 0 || !l.AllMachines.Macs[0].ThisMac {
			l.Tailnet = ports.Status{}
		}
		l.Verdict = ports.VerdictOf(l.AllMachines)
		out[viewPortsOf+name] = viewOf(sl, l)
	}
}

// value: every Mac and every VM the fleet lists, by name.
func machineNames(all *machines.AllMachines) []string {
	var out []string
	for _, h := range all.Macs {
		out = append(out, h.Name)
		for _, vm := range h.VMs {
			out = append(out, vm.Name)
		}
	}
	return slices.Compact(slices.Sorted(slices.Values(out)))
}

// value: the sign-ins and the Awake of each machine the store holds, as
// accounts.list and awake.read answer them: This Mac's by the call's name,
// another machine's by `<call>:<machine>`. An Awake is said as of its
// reading.
func (v view) ownViews(asked func(name string) bool, out map[string]store.View) {
	for k, sl := range v.s.Slices {
		if read, _, ok := value[accounts.ProgramsRead](v.s, k); ok && k.Family() == store.FamilySignins {
			if name := v.named(ViewAccounts, k); asked(name) {
				out[name] = viewOf(sl, read)
			}
		}
		if r, _, ok := value[awake.Report](v.s, k); ok && k.Family() == store.FamilyAwake {
			if name := v.named(ViewAwake, k); asked(name) {
				out[name] = viewOf(sl, v.awakeOf(k.Machine(), r, sl.ReadAt))
			}
		}
	}
}

// value: the view of a slice by its call: This Mac's by the call's name,
// another machine's by `<call>:<machine>`.
func (v view) named(call string, k store.Key) string {
	if k.Machine() == v.this {
		return call
	}
	return call + ":" + k.Machine()
}

// value: a Mac's Awake as awake.read answers it at the instant it was read:
// the servers it hosts and the work that pauses while it sleeps from its
// machines slice, its choices and its verdict said then.
func (v view) awakeOf(mac string, r awake.Report, read time.Time) awake.Report {
	r.Mac, r.Now = mac, read
	if h, _, ok := value[machines.MacView](v.s, store.Of(store.FamilyMac, mac)); ok {
		r.Servers, r.Pauses = nil, machines.Pausing(h.VMs)
		for _, vm := range h.VMs {
			if vm.Lifetime == machines.AlwaysOn {
				r.Servers = append(r.Servers, vm.Name)
			}
		}
		slices.Sort(r.Servers)
	}
	r.Choices, r.Verdict = awake.Choices(r.Now, r.Zone, r.Keeping()), awake.VerdictOf(r)
	return r
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
