package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/activity"
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
	MachineID  page.ID = "menubar.machine"
	SwitcherID page.ID = "menubar.switcher"

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
	// value: a terminal on one family of a workspace (`open:apps`).
	ActionOpenFamily = "open:"

	CallCleanup = "machines.cleanup"
	CallWake    = "machines.wake"
	CallRemind  = "machines.remind"

	SectionMachine  = "machine"
	SectionSummary  = "summary"
	SectionActivity = "activity"
	SectionInside   = "inside"
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
	Name     string
	All      *model.AllMachines
	Summary  *overview.Summary
	Samples  []activity.Reading
	Ports    *ports.AllMachines
	Accounts *accounts.Report
	// value: a Mac's page: the copies every Mac shared, its Awake, and the
	// zone its times are said in.
	Shares *accounts.AllMachines
	Awake  *awake.Report
	Zone   *time.Location
	Now    time.Time
	Asks   Levels
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

// value: the samples a VM's page reads: none while it does not run.
func running(m Machine, vm model.VM) []activity.Reading {
	if vm.State != model.Running {
		return nil
	}
	return m.Samples
}

func vmPage(m Machine, h model.MacView, vm model.VM) page.Page {
	if vm.Lifetime.Kind() == machine.Server {
		return serverPage(m, h, vm)
	}
	return workspacePage(m, h, vm)
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
		p.Sections = append(p.Sections, macActivity(m, h), macPorts(m, h), macAccounts(m, h))
		p.Actions = append(p.Actions, overviewActions(m, h, label)...)
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

func Switcher(r Reading) page.Page {
	// invariant: SPEC §10.2's switcher: All machines as a tree, each Mac then its
	// Servers then its Workspaces, and Add a Mac…; Recent is the app's own. A
	// Mac's dot says what needs you there, as Home's does, and a server says
	// when its check last answered (vision panel b).
	s := Tree(r.All, r.Ports, func(h model.MacView) page.Value {
		if h.Engine.Running {
			return page.Text("Engine running")
		}
		return page.Text("Engine stopped")
	})
	markMacs(&s, r)
	if r.All != nil {
		checked(&s, *r.All)
	}
	s.Title = "All machines"
	add := page.Action{ID: ActionAddMac, Label: "Add a Mac", On: page.OnPage, Priority: 1, Mutates: true, Run: &page.Run{Call: CallAddMac},
		Form: &page.Form{Title: "Add a Mac", Fields: []page.Field{{Key: "Name", Label: "Name", Kind: page.FieldText, Required: true, Note: "as " + catalog.TailnetProduct + " names it"}}}}
	return page.Seal(page.Page{ID: SwitcherID, Command: "menubar", Sections: []page.Section{s}, Actions: []page.Action{add}})
}

// value: a running server's line in the switcher: its lifetime, its health
// and how long ago its check answered, in place of its time up and restarts.
func checked(s *page.Section, f model.AllMachines) {
	vms := map[string]model.VM{}
	for _, h := range f.Macs {
		for _, vm := range h.VMs {
			vms[vm.Name] = vm
		}
	}
	for _, mac := range s.Items {
		if mac.Group == nil {
			continue
		}
		for _, g := range mac.Group.Items {
			if g.Group == nil || g.Group.Head.Label != GroupServers {
				continue
			}
			for _, it := range g.Group.Items {
				vm, ok := vms[it.Row.ID]
				if !ok || vm.State != model.Running || model.Down(vm) || vm.CheckStatus.At.IsZero() || f.Read.IsZero() || len(it.Row.Cells) < 2 {
					continue
				}
				at := page.Age(vm.CheckStatus.At, f.Read)
				at.Label = "checked"
				it.Row.Cells[1] = page.List(pagemachines.Lifetime(vm.Lifetime), page.Text(health(vm)), at)
			}
		}
	}
}
