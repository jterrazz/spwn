package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	MachineID page.ID = "menubar.machine"

	ActionStop     = "stop"
	ActionDelete   = "delete"
	ActionRestart  = "restart"
	ActionShare    = "share-port"
	ActionTakeBack = "take-back"
	ActionUnlock   = "unlock"
	ActionAddMac   = "add-mac"
	ActionStartVM  = "start-vm"
	ActionMenu     = "menu"
	ActionAttach   = "attach:"
	ActionCleanup  = "cleanup"
	ActionWake     = "wake"
	ActionRemind   = "remind"

	CallCleanup = "machines.cleanup"
	CallWake    = "machines.wake"
	CallRemind  = "machines.remind"

	SectionMachine  = "machine"
	SectionSummary  = "summary"
	SectionActivity = "activity"
	SectionPorts    = "ports"
	SectionShared   = "shared"
	SectionUp       = "up"
	SectionHealth   = "health"
	SectionServer   = "server"
	SectionVMs      = "vms"
	SectionGauges   = "gauges"
	SectionCare     = "care"
	SectionAsleep   = "asleep"

	SegmentOverview = "overview"
	SegmentActivity = "activity"
	SegmentPorts    = "ports"
	SegmentAccounts = "accounts"
)

type Machine struct {
	// value: the machine the page is of, a VM or a Mac, by its name among
	// the machines; a nil part is still being read.
	Name    string
	All     *model.AllMachines
	Summary *overview.Summary
	Samples []activity.Reading
	// value: the machine's activity as its engine keeps it: the last 6 min
	// at 2 s (Graph) and the day at a minute (History), nil until read.
	Activity *activity.Report
	Ports    *ports.AllMachines
	Accounts *accounts.Report
	// value: a Mac's page: the copies every Mac shared, its Awake, and the
	// zone its times are said in.
	Shares *accounts.AllMachines
	Awake  *awake.Report
	Zone   *time.Location
	Now    time.Time
	Asks   Levels
	// value: a workspace's page: the fleet's agents, nil until they are read.
	Agents []agents.Agent
}

func MachinePage(m Machine) page.Page {
	// invariant: SPEC A4's page of one machine: a Mac's, a workspace's or a
	// server's, each with only the actions its kind takes, each naming its
	// call and asking at the level the call asks.
	if m.All == nil {
		row := page.Row{ID: "reading", Label: m.Name, Mark: page.Awaiting}
		return page.Seal(page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: m.Name}, Sections: []page.Section{{ID: SectionMachine, Title: m.Name, Role: page.Summary, Items: []page.Item{{Row: &row}}}}})
	}
	for _, h := range m.All.Macs {
		if h.Name == m.Name {
			return macPage(m, h)
		}
		for _, vm := range h.VMs {
			if vm.Name == m.Name {
				return vmPage(m, h, vm)
			}
		}
	}
	return page.Seal(page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: m.Name},
		Error: &page.Error{Message: "no machine is named " + m.Name, Hint: "station machines"}})
}

// value: where a VM's call runs: on the Mac that holds it, which reaches
// the VM itself; nil on This Mac.
func target(h model.MacView) *page.Target {
	if h.ThisMac {
		return nil
	}
	return &page.Target{Name: h.Name}
}

// value: a VM's page head: its state, its kind and lifetime, the Mac it
// is on (a server's size too), and the page it heads.
func vmHead(m Machine, h model.MacView, vm model.VM) (page.Page, page.Section) {
	kind := vm.Lifetime.Kind()
	on := model.Label(*m.All, h)
	head := page.Row{ID: "state", Label: "State", Mark: page.OK, Cells: []page.Value{pagemachines.State(vm.State)}}
	if vm.State != model.Running {
		head.Mark, head.Reason = page.Absent, page.Stopped
	}
	if model.Down(vm) {
		head.Mark, head.Reason = page.Broken, page.Down
	}
	caption := []page.Value{page.Text(model.KindWord(kind) + " VM"), pagemachines.Lifetime(vm.Lifetime), page.Text("on " + on)}
	if kind == machine.Server {
		r := vm.Resources
		caption = append(caption, page.Text(strconv.Itoa(r.CPUs)+" CPU"), pagemachines.Memory(r.MemoryBytes))
	}
	section := page.Section{ID: SectionMachine, Title: vm.Name, Role: page.Summary, Caption: caption, Columns: columns(), Items: []page.Item{{Row: &head}}}
	return page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: vm.Name, Kind: string(kind), On: on}}, section
}

func vmPage(m Machine, h model.MacView, vm model.VM) page.Page {
	if vm.Lifetime.Kind() == machine.Server {
		return vmFrame(m, h, vm, serverPage(m, h, vm))
	}
	return vmFrame(m, h, vm, workspacePage(m, h, vm))
}

// invariant: SPEC A4: a running VM's page carries its header and its
// segments (a server's Overview and Activity, a workspace's four), a
// workspace's loans under Accounts and every other section of its kind's
// page under Overview, then its Activity's; a header or a summary its
// kind's page drew is kept; a VM that does not run has none of these.
func vmFrame(m Machine, h model.MacView, vm model.VM, p page.Page) page.Page {
	if vm.State != model.Running {
		return p
	}
	if p.Header == nil {
		glyph := page.Glyph{Kind: "workspace"}
		switch {
		case vm.Lifetime.Kind() == machine.Server:
			glyph.Kind = "server"
		case vm.Agent != nil && !vm.Finished():
			glyph.Kind = "agent"
		}
		p.Header = &page.Header{Back: model.Label(*m.All, h), Glyph: glyph}
	}
	p.Segments = segments()
	if vm.Lifetime.Kind() == machine.Server {
		p.Segments = p.Segments[:2]
	}
	for i := range p.Sections {
		switch {
		case p.Sections[i].ID == SectionShared:
			p.Sections[i].Segment = SegmentAccounts
		case p.Sections[i].Segment == "" && p.Sections[i].ID != SectionMachine && p.Sections[i].ID != SectionSummary:
			p.Sections[i].Segment = SegmentOverview
		}
	}
	if !sectionHas(p, SectionSummary) {
		p.Sections = append(p.Sections, page.Section{ID: SectionSummary, Title: vm.Name, Role: page.Summary, Caption: vmFacts(m, vm)})
	}
	p.Sections = append(p.Sections, vmActivity(m, h, vm)...)
	if sectionHas(p, SectionGuestUpkeep) {
		p.Actions = append(p.Actions, updateGuest(h, vm))
	}
	p.Live = pageLive(p)
	return page.Seal(p)
}

func sectionHas(p page.Page, id string) bool {
	return slices.ContainsFunc(p.Sections, func(s page.Section) bool { return s.ID == id })
}

// value: a VM's facts in its header: its kind or lifetime, how long it
// has run.
func vmFacts(m Machine, vm model.VM) []page.Value {
	word := lifeWord(vm)
	if vm.Lifetime.Kind() == machine.Server {
		word = "server"
	}
	out := []page.Value{page.Text(word)}
	if !vm.BootAt.IsZero() && m.Now.After(vm.BootAt) {
		out = append(out, page.Text("up "+days(m.Now.Sub(vm.BootAt))))
	}
	return out
}

func items(rows []page.Row) []page.Item {
	out := make([]page.Item, 0, len(rows))
	for i := range rows {
		out = append(out, page.Item{Row: &rows[i]})
	}
	return out
}

func macPage(m Machine, h model.MacView) page.Page {
	// invariant: SPEC A4 A Mac: its header (the way back, its glyph by its
	// lid, New VM and its menu), its facts, then every segment's sections
	// in the one page, so a segment is picked on the client and asks
	// nothing; an asleep Mac has no segment and says so; a Mac not heard
	// from carries home's stale marking.
	label := model.Label(*m.All, h)
	p := page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: h.Name, Kind: string(machine.Mac)}}
	p.Header = &page.Header{Back: "All machines", Glyph: page.Glyph{Kind: lid(h)}}
	stale := model.Unheard(h, m.All.Read)
	p.Sections = []page.Section{{ID: SectionSummary, Title: label, Role: page.Summary, Caption: facts(m, h)}}
	menu := page.Action{ID: ActionMenu, Label: "More options for " + label, Icon: "ellipsis", On: page.OnPage}
	if model.Reach(h) == model.ReachAsleep {
		p.Sections = append(p.Sections, asleep(label)...)
		p.Actions = append(p.Actions, wake(h, label)...)
	} else {
		if stale {
			v := page.Texts("checking")
			if !h.Answered.IsZero() && m.All.Read.After(h.Answered) {
				v = page.Texts("seen "+ago(m.All.Read.Sub(h.Answered))+" ago", "checking")
			}
			p.Header.Stale = &v
		}
		p.Segments = segments()
		p.Sections = append(p.Sections, macOverviewSections(m, h, label, stale)...)
		shares := macPorts(m, h)
		p.Sections = append(p.Sections, macActivity(m, h)...)
		p.Sections = append(p.Sections, shares.Sections...)
		p.Actions = append(p.Actions, overviewActions(m, h, label)...)
		p.Actions = append(p.Actions, shares.Actions...)
		accountSections, accountActions := macAccounts(m, h)
		p.Sections = append(p.Sections, accountSections...)
		p.Actions = append(p.Actions, accountActions...)
	}
	if slices.ContainsFunc(p.Actions, func(a page.Action) bool { return a.ID == ActionNewVM }) && len(h.VMs) > 0 {
		p.Header.Buttons = append(p.Header.Buttons, ActionNewVM)
	}
	if locked(m, h) {
		p.Actions = append(p.Actions, page.Action{ID: ActionUnlock, Label: "Unlock the disk", On: page.OnPage, Priority: 1, Mutates: true, Run: &page.Run{Terminal: true, Argv: []string{"machines", "unlock", label}}})
		menu.Submenu = append(menu.Submenu, ActionUnlock)
	}
	if !h.ThisMac {
		p.Actions = append(p.Actions, openIn("--on", label))
	} else {
		p.Actions = append(p.Actions, openIn())
	}
	menu.Submenu = append(menu.Submenu, ActionOpenStation)
	p.Actions = append(p.Actions, menu)
	p.Header.Buttons = append(p.Header.Buttons, ActionMenu)
	p.Live = pageLive(p)
	return page.Seal(p)
}

// value: a Mac's glyph by its lid, a capability, never its model.
func lid(h model.MacView) string {
	if h.Laptop {
		return "laptop"
	}
	return "desktop"
}

func segments() []page.Segment {
	return []page.Segment{
		{ID: SegmentOverview, Label: "Overview", Picked: true},
		{ID: SegmentActivity, Label: "Activity", Key: "cmd+i"},
		{ID: SegmentPorts, Label: "Ports", Key: "cmd+p"},
		{ID: SegmentAccounts, Label: "Accounts", Key: "cmd+u"},
	}
}

// value: SPEC B5.12: what a Mac is, as it read itself ("MacBook Pro · M4
// Max · 48 GB"), then how long it has run, or when it was last seen
// while it sleeps or does not answer.
func facts(m Machine, h model.MacView) []page.Value {
	var out []page.Value
	hw := h.Hardware
	part := func(text string, pri int) {
		v := page.Text(text)
		v.Pri = pri
		out = append(out, v)
	}
	if hw.Model != "" {
		part(catalog.MacModelName(hw.Model), 1)
	}
	if chip := strings.TrimPrefix(hw.Chip, "Apple "); chip != "" {
		part(chip, 2)
	}
	if hw.Memory > 0 {
		part(strconv.FormatInt(hw.Memory>>30, 10)+" GB", 3)
	}
	switch {
	case model.Reach(h) == model.ReachAsleep:
		if !h.Answered.IsZero() {
			out = append(out, page.Text("seen "+ago(m.All.Read.Sub(h.Answered))+" ago"))
		}
	case m.Summary != nil && m.Summary.Head.Up > 0:
		part("up "+days(m.Summary.Head.Up), 4)
	}
	return out
}

// value: how long ago, as the panel says it: "40 s", "6 min", "2 hr".
func ago(d time.Duration) string {
	if d < time.Minute {
		return strconv.Itoa(max(int(d/time.Second), 1)) + " s"
	}
	return minutes(d)
}

// value: a time up as the panel says it: "3 days", "1 day", else as a wait.
func days(d time.Duration) string {
	day := 24 * time.Hour
	switch {
	case d >= 2*day:
		return strconv.Itoa(int(d/day)) + " days"
	case d >= day:
		return "1 day"
	}
	return minutes(d)
}

// value: every live key the page's values follow, each once, in order:
// its rows', its groups' and its gauges'.
func pageLive(p page.Page) []string {
	keys := liveKeys(p)
	add := func(v page.Value) {
		if v.Live != "" && !slices.Contains(keys, v.Live) {
			keys = append(keys, v.Live)
		}
	}
	var walk func(it page.Item)
	walk = func(it page.Item) {
		switch {
		case it.Row != nil:
			for _, v := range it.Row.Cells {
				add(v)
			}
		case it.Group != nil:
			for _, v := range it.Group.Head.Cells {
				add(v)
			}
			for _, c := range it.Group.Items {
				walk(c)
			}
		case it.Gauge != nil:
			add(it.Gauge.Value)
		}
	}
	for _, s := range p.Sections {
		for _, it := range s.Items {
			walk(it)
		}
	}
	return keys
}

// value: the Mac waits for its disk's password, as its readiness or its
// unreachable row says: Unlock… is its one action that helps.
func locked(m Machine, h model.MacView) bool {
	if m.Summary != nil && slices.ContainsFunc(m.Summary.Readiness, func(c overview.Check) bool {
		return c.Name == model.ItemDiskUnlock && c.State != catalog.StateOK
	}) {
		return true
	}
	return !h.Reachable && !h.ThisMac && h.LAN != ""
}
