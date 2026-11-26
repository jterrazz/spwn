package feeds

import (
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Pages struct {
	ThisMachine page.Context
	// value: the zone the pages say their times in; nil is the process's own
	// local zone.
	Zone *time.Location
}

func (o Pages) zone() *time.Location {
	if o.Zone == nil {
		return time.Local
	}
	return o.Zone
}

// value: one snapshot as the pages read it, at its own instant: a page
// never reads the clock, so one revision projects one page.
type view struct {
	s    *store.Snapshot
	this string
	now  time.Time
}

func (o Pages) of(s *store.Snapshot) view {
	// invariant: each page is a pure function of one snapshot of the store
	// (ADR-070): it reads nothing and asks no machine, so a page opens whole
	// from what the engine already holds.
	return view{s: s, this: o.ThisMachine.Machine, now: s.At}
}

// value: a slice's value as its family's reader put it, and the slice; ok
// false while it holds no value.
func value[T any](s *store.Snapshot, k store.Key) (T, store.Slice, bool) {
	sl, held := s.Slices[k]
	v, ok := sl.Value.(T)
	return v, sl, held && ok
}

// value: whether a slice could not be read for a reason other than its
// first reading under way, with that reason.
func failed(sl store.Slice) (string, bool) {
	return sl.Why, sl.Why != "" && sl.Why != store.WhyReading
}

// value: the confirmation rule of the calls, which the page only draws.
func levels(gesture string, kind machine.Kind, facts ...string) string {
	graded := make([]confirm.Fact, 0, len(facts))
	for _, f := range facts {
		graded = append(graded, confirm.Fact(f))
	}
	return string(confirm.Of(confirm.Action(gesture), kind, graded...))
}

// value: every other Mac the snapshot knows, by name: those of This Mac's
// records, and those whose slices it holds.
func (v view) others() []string {
	var out []string
	records, _, _ := value[map[string]machines.MacRecord](v.s, store.Of(store.FamilyRecords, ""))
	for n := range records {
		out = append(out, n)
	}
	for k := range v.s.Slices {
		if k.Family() == store.FamilyMac && k.Machine() != v.this {
			out = append(out, k.Machine())
		}
	}
	return slices.Compact(slices.Sorted(slices.Values(out)))
}

func (v view) machines() *machines.AllMachines {
	// invariant: All machines as machines.list composes it, from the slices
	// alone: This Mac's own, every other Mac's own answer while it is fresh;
	// one gone silent keeps the VMs it last said, unreachable, seen at its
	// last reading; one never heard is drawn from its record.
	thisMac, _, ok := value[machines.MacView](v.s, store.Of(store.FamilyMac, v.this))
	if !ok {
		thisMac = machines.MacView{Name: v.this, ThisMac: true, Reachable: true}
	}
	records, _, _ := value[map[string]machines.MacRecord](v.s, store.Of(store.FamilyRecords, ""))
	nodes, _, _ := value[map[string]machines.Node](v.s, store.Of(store.FamilyTailnet, ""))
	f := machines.AllMachines{Macs: []machines.MacView{thisMac}, Read: v.now}
	for _, n := range v.others() {
		far, sl, held := value[machines.MacView](v.s, store.Of(store.FamilyMac, n))
		fresh := held && sl.State == store.Fresh
		h := machines.ViewOf(n, records[n], nodes[n], far, fresh)
		if held && !fresh {
			h.VMs = nil
			for _, vm := range far.VMs {
				vm.Mac = n
				h.VMs = append(h.VMs, vm)
			}
			if sl.ReadAt.After(h.Answered) {
				h.Answered = sl.ReadAt
			}
		}
		f.Macs = append(f.Macs, h)
	}
	judged := machines.Judged(f)
	return &judged
}

// value: every Mac's ports, This Mac's first; nil while This Mac's are not
// read. Another Mac that does not answer lists none.
func (v view) ports(all *machines.AllMachines) *ports.AllMachines {
	own, _, ok := value[ports.MacPorts](v.s, store.Of(store.FamilyPorts, v.this))
	if !ok {
		return nil
	}
	out := ports.AllMachines{Schema: ports.ReservationsSchema, Macs: []ports.MacPorts{own}}
	for _, h := range all.Macs {
		if h.ThisMac {
			continue
		}
		far, sl, held := value[ports.MacPorts](v.s, store.Of(store.FamilyPorts, h.Name))
		if !held || sl.State != store.Fresh || !h.Reachable {
			out.Macs = append(out.Macs, ports.MacPorts{Name: h.Name, Shared: []ports.SharedPort{}})
			continue
		}
		far.Name, far.ThisMac, far.Reachable = h.Name, false, true
		if far.Shared == nil {
			far.Shared = []ports.SharedPort{}
		}
		out.Macs = append(out.Macs, far)
	}
	return &out
}

// value: a machine's Overview as of the snapshot's instant; nil while it is
// not read.
func (v view) summary(name string) *overview.Summary {
	kept, sl, ok := value[overview.Kept](v.s, store.Of(store.FamilyOverview, name))
	if !ok {
		return nil
	}
	s := kept.At(v.now, sl.ReadAt)
	return &s
}

// value: a Mac's Awake as of the snapshot's instant; nil while it is not
// read.
func (v view) awake(mac string) *awake.Report {
	r, _, ok := value[awake.Report](v.s, store.Of(store.FamilyAwake, mac))
	if !ok {
		return nil
	}
	r.Mac, r.Now = mac, v.now
	if r.Zone == nil {
		r.Zone = time.Local
	}
	return &r
}

// value: every other Mac's Awake and Overview, of each that answers.
func (v view) elsewhere(all *machines.AllMachines) (macs []awake.Report, away []overview.Summary) {
	for _, h := range all.Macs {
		if h.ThisMac || !h.Reachable {
			continue
		}
		if r := v.awake(h.Name); r != nil {
			macs = append(macs, *r)
		}
		if s := v.summary(h.Name); s != nil {
			away = append(away, *s)
		}
	}
	return macs, away
}

// value: the sign-ins and grants of every Mac and its running VMs, or of
// one Mac's, and This Mac's vault; nil while This Mac's are not read. A
// Mac that does not answer is named, never dropped.
func (v view) signIns(all *machines.AllMachines, only string) *pagemenubar.SignIns {
	if _, _, ok := value[accounts.ProgramsRead](v.s, store.Of(store.FamilySignins, v.this)); !ok {
		return nil
	}
	var out pagemenubar.SignIns
	if vo, _, ok := value[accounts.VaultOffers](v.s, store.Of(store.FamilyVault, "")); ok {
		out.Vault, out.Offers = &vo.Vault, vo.Offers
	}
	for _, h := range all.Macs {
		if only != "" && h.Name != only {
			continue
		}
		m := pagemenubar.MachineSignIns{Name: h.Name, Mac: h.Name, ThisMac: h.ThisMac}
		if !h.ThisMac && !h.Reachable {
			m.Silent = page.MacName(h.Name, v.this) + " does not answer"
			out.Machines = append(out.Machines, m)
			continue
		}
		out.Machines = append(out.Machines, v.programs(m))
		for _, vm := range h.VMs {
			if vm.State != machines.Running {
				continue
			}
			out.Machines = append(out.Machines, v.programs(pagemenubar.MachineSignIns{Name: vm.Name, Mac: h.Name, ThisMac: h.ThisMac, VM: true}))
		}
	}
	return &out
}

func (v view) programs(m pagemenubar.MachineSignIns) pagemenubar.MachineSignIns {
	read, sl, ok := value[accounts.ProgramsRead](v.s, store.Of(store.FamilySignins, m.Name))
	if !ok {
		if why, bad := failed(sl); bad {
			m.Silent = why
		}
		return m
	}
	m.Rows = read.Programs
	return m
}

// value: a machine's activity as its engine keeps it; a machine gone
// silent keeps the last it said.
func (v view) activity(name string) *activity.Report {
	r, _, ok := value[activity.Report](v.s, store.Of(store.FamilyActivity, name))
	if !ok {
		return nil
	}
	return &r
}

func (v view) reading() pagemenubar.Reading {
	all := v.machines()
	r := pagemenubar.Reading{Home: v.summary(v.this), All: all, Ports: v.ports(all), Awake: v.awake(v.this), SignIns: v.signIns(all, ""), Now: v.now}
	r.Macs, r.Away = v.elsewhere(all)
	return r
}

func (o Pages) Home(s *store.Snapshot, _ string) page.Page {
	r := o.of(s).reading()
	r.Zone, r.Asks = o.zone(), levels
	return pagemenubar.Page(r, o.ThisMachine)
}

func (o Pages) Find(s *store.Snapshot, _ string) page.Page {
	// invariant: SPEC B5.6: Find is drawn from home's own reading.
	r := o.of(s).reading()
	r.Zone = o.zone()
	return pagemenubar.Find(pagemenubar.FindReading{Reading: r, Now: s.At, Zone: o.zone(), Asks: levels}, o.ThisMachine)
}

func (o Pages) Machine(s *store.Snapshot, name string) page.Page {
	// invariant: SPEC §10.3: a machine's page, a Mac's or a VM's: another
	// Mac's is drawn from what that Mac's engine pushed; a Mac's page holds
	// every Mac's sign-ins, a VM's those of its Mac. Its moving numbers are
	// its live keys, never a projection.
	v := o.of(s)
	all := v.machines()
	m := pagemenubar.Machine{Name: name, Asks: levels, Zone: o.zone(), All: all, Summary: v.summary(name), Activity: v.activity(name), Ports: v.ports(all), Now: v.now}
	if h, isMac := macOf(*all, name); h.Name != "" {
		only := h.Name
		if isMac {
			only = ""
		}
		m.SignIns = v.signIns(all, only)
		m.Awake = v.awake(h.Name)
	}
	return pagemenubar.MachinePage(m)
}

// value: the Mac a page's machine is on: itself, or the Mac that runs it.
func macOf(all machines.AllMachines, name string) (machines.MacView, bool) {
	for _, h := range all.Macs {
		if h.Name == name {
			return h, true
		}
	}
	for _, h := range all.Macs {
		for _, vm := range h.VMs {
			if vm.Name == name {
				return h, false
			}
		}
	}
	return machines.MacView{}, false
}

func (o Pages) Ports(s *store.Snapshot, subject string) page.Page {
	// invariant: SPEC §10.2 Ports: the ports of the machine the page is of,
	// a Mac or a VM, kept by the Mac it is on, so a server down still lists
	// its ports unreachable; of every Mac when it names none.
	v := o.of(s)
	all := v.machines()
	listed := v.ports(all)
	if listed == nil {
		return o.unreadOr(s, store.Of(store.FamilyPorts, v.this), pagemenubar.PortsID, func() page.Page {
			return pagemenubar.Ports(ports.Listing{}, pagemenubar.PortsOf{Machine: subject, Now: v.now, Machines: all, Asks: levels}, o.ThisMachine)
		})
	}
	l := ports.Listing{AllMachines: *listed}
	if subject != "" {
		l.AllMachines = ports.Of(l.AllMachines, subject)
	}
	l.Verdict = ports.VerdictOf(l.AllMachines)
	return pagemenubar.Ports(l, pagemenubar.PortsOf{Machine: subject, Now: v.now, Machines: all, Asks: levels}, o.ThisMachine)
}

// value: the page a slice that could not be read draws, its why said, or
// else the page as drawn while it is read.
func (o Pages) unreadOr(s *store.Snapshot, k store.Key, id page.ID, reading func() page.Page) page.Page {
	if why, bad := failed(s.Slices[k]); bad {
		return pagemenubar.Unread(id, o.ThisMachine, unread(why))
	}
	return reading()
}

type unread string

func (u unread) Error() string { return string(u) }

func (o Pages) Accounts(s *store.Snapshot, subject string) page.Page {
	// invariant: SPEC B5.8: the sign-ins and grants of every answering Mac
	// and its VMs, each as its own engine read them; of one Mac, or of the
	// Mac a VM is on, when the page names one.
	v := o.of(s)
	all := v.machines()
	of := accountsOf(*all, subject)
	of.Asks = levels
	signed := v.signIns(all, of.Mac)
	if signed == nil {
		return o.unreadOr(s, store.Of(store.FamilySignins, v.this), pagemenubar.AccountsID, func() page.Page {
			return pagemenubar.Accounts(pagemenubar.SignIns{}, of, o.ThisMachine)
		})
	}
	return pagemenubar.Accounts(*signed, of, o.ThisMachine)
}

func (o Pages) Needs(s *store.Snapshot, subject string) page.Page {
	// invariant: SPEC B5.7: menubar.needs[:<machine>] is drawn from what the
	// needs are made of: the machines, every Mac's ports and every
	// answering Mac's Overview.
	v := o.of(s)
	all := v.machines()
	r := pagemenubar.NeedsReading{Now: v.now, Zone: o.zone(), All: all, Ports: v.ports(all), Asks: levels}
	for _, h := range all.Macs {
		if !h.ThisMac && !h.Reachable {
			continue
		}
		if sum := v.summary(h.Name); sum != nil {
			r.Overviews = append(r.Overviews, *sum)
		}
	}
	return pagemenubar.NeedsPage(r, needsOf(*all, subject), o.ThisMachine)
}

func (o Pages) Awake(s *store.Snapshot, _ string) page.Page {
	// invariant: SPEC §10.2 Awake: This Mac's Awake then each other Mac's,
	// as each engine keeps it; one that does not answer as it last did.
	v := o.of(s)
	all := v.machines()
	this := v.awake(v.this)
	if this == nil {
		return o.unreadOr(s, store.Of(store.FamilyAwake, v.this), pagemenubar.AwakeID, func() page.Page {
			return pagemenubar.Awake(pageawake.AllMachines{Now: v.now, Zone: o.zone(), This: v.this}, o.ThisMachine, levels)
		})
	}
	f := pageawake.AllMachines{Now: v.now, Zone: o.zone(), This: v.this, Reports: []awake.Report{*this}}
	records, _, _ := value[map[string]machines.MacRecord](v.s, store.Of(store.FamilyRecords, ""))
	for _, h := range all.Macs {
		if h.ThisMac {
			continue
		}
		if r := v.awake(h.Name); r != nil && h.Reachable {
			f.Reports = append(f.Reports, *r)
			continue
		}
		f.Silent = append(f.Silent, silentAwake(h, records[h.Name], v.this))
	}
	return pagemenubar.Awake(f, o.ThisMachine, levels)
}

// value: a Mac that does not answer, as it last did: asleep or not, when,
// and the servers it then hosted.
func silentAwake(h machines.MacView, rec machines.MacRecord, this string) pageawake.Silent {
	s := pageawake.Silent{Mac: h.Name, Why: page.MacName(h.Name, this) + " does not answer", Asleep: rec.Asleep, Answered: h.Answered}
	for _, vm := range rec.VMs {
		if vm.Lifetime == machines.AlwaysOn {
			s.Servers = append(s.Servers, vm.Name)
		}
	}
	return s
}

// value: the machine a needs page is of, by its full name: a Mac named as
// the interface names it, or a VM; empty is every machine.
func needsOf(all machines.AllMachines, subject string) string {
	if subject == "" {
		return ""
	}
	for _, h := range all.Macs {
		if h.Name == subject || strings.HasSuffix(h.Name, "-"+subject) {
			return h.Name
		}
	}
	return subject
}

// value: what an accounts page is of: a Mac named in full or as the
// interface names it, or a VM by the Mac it is on; empty is every Mac.
func accountsOf(all machines.AllMachines, subject string) pagemenubar.AccountsOf {
	if subject == "" {
		return pagemenubar.AccountsOf{}
	}
	for _, h := range all.Macs {
		if h.Name == subject || strings.HasSuffix(h.Name, "-"+subject) {
			return pagemenubar.AccountsOf{Mac: h.Name}
		}
	}
	for _, h := range all.Macs {
		for _, vm := range h.VMs {
			if vm.Name == subject {
				return pagemenubar.AccountsOf{Mac: h.Name, VM: vm.Name}
			}
		}
	}
	return pagemenubar.AccountsOf{Mac: subject}
}
