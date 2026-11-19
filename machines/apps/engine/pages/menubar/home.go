package menubar

import (
	"cmp"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagemachines "github.com/jterrazz/studio-station/apps/engine/pages/machines"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	SectionNeedsRows = "needs"
	SectionServers   = "servers"
	SectionMacs      = "macs"
	SectionEmpty     = "empty"
	SectionShortcuts = "shortcuts"

	RowPorts    = "ports"
	RowAccounts = "accounts"

	RowAddMac = "add-mac"

	ActionNewVMOn = "new-vm:"

	searchPrompt = "Find a machine, a port, an action"
)

// value: what Home's sections read: the fleet, This Mac's name, the needs
// and the upkeep ranked, the clock.
type homeReading struct {
	r      Reading
	self   string
	needs  []Need
	upkeep []Need
	now    time.Time
	stale  string
}

// invariant: Home is the V3 panel, one kind per section, in one order:
// what needs you (only when something does), This Mac's measures under its
// head, the servers, the VMs counted, the Macs, then the Keep awake, Ports
// and Accounts rows; an empty fleet keeps its empty state and its Macs.
func homeSections(r Reading, self string) ([]page.Section, []page.Action) {
	ranked, upkeep := needsOf(r.needsReading(), self)
	h := homeReading{r: r, self: self, needs: ranked, upkeep: upkeep, now: r.now(), stale: staleMac(r, self)}
	var sections []page.Section
	actions := verbs(ranked)
	if s, ok := h.needsRows(); ok {
		sections = append(sections, s)
	}
	sections = append(sections, h.thisMac())
	if empty(*r.All) {
		sections = append(sections, page.Section{ID: SectionEmpty, Title: "No VMs yet", Card: page.CardEmpty, Items: []page.Item{{Empty: h.empty()}}})
	} else {
		if s, ok := h.servers(); ok {
			sections = append(sections, s)
		}
		if s, ok := h.vms(); ok {
			sections = append(sections, s)
		}
	}
	macs, more := h.macs()
	sections = append(sections, macs)
	actions = append(actions, more...)
	shortcuts, toggle := h.shortcuts()
	sections = append(sections, shortcuts)
	actions = append(actions, toggle)
	for _, m := range r.All.Macs {
		for _, vm := range m.VMs {
			if alive(vm) {
				actions = append(actions, attach(m, vm))
			}
		}
	}
	every, cleans := cleanupsOf(r.All.Macs), false
	for _, c := range every {
		if c.any() {
			actions = append(actions, cleanupSheet(ActionCleanup+":"+c.mac.Name, *r.All, []cleanupOf{c}, true))
		}
		cleans = cleans || c.any()
	}
	if cleans {
		actions = append(actions, cleanupSheet(ActionCleanup, *r.All, every, false))
	}
	return sections, actions
}

func alive(vm model.VM) bool { return vm.State == model.Running || vm.State == model.Creating }

// value: Needs you, one row per thing that needs you, ranked, each its
// signal's dot, its subject, the Mac it is on, what it is and its verb;
// a Mac still answering dims the ones of it. No row, no section.
func (h homeReading) needsRows() (page.Section, bool) {
	s := page.Section{ID: SectionNeedsRows, Title: "Needs you", Columns: columns(), Uncounted: true}
	for _, n := range h.needs {
		row := page.Row{ID: n.Key, Label: n.Subject, Detail: page.Text(n.What), Glyph: page.Glyph{Dot: page.DotFill, Signal: n.Glyph.Signal}, Verb: n.Verb.ID, Nav: n.Nav, Mark: page.Broken, Reason: reasonOf(n.Finding), Signal: h.signal(n)}
		if len(n.Scope) > 0 && n.Subject != page.MacName(n.Scope[0], h.self) {
			row.Mac = page.MacName(n.Scope[0], h.self)
		}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	return s, len(s.Items) > 0
}

// value: a need of the Mac still answering is drawn stale.
func (h homeReading) signal(n Need) page.Signal {
	if h.stale == "" {
		return ""
	}
	for _, m := range h.r.All.Macs {
		if page.MacName(m.Name, h.self) == h.stale && n.of(m.Name) {
			return page.SignalStale
		}
	}
	return ""
}

// value: This Mac's head over the page's strip, its caption the memory a
// new VM finds there.
func (h homeReading) thisMac() page.Section {
	s := page.Section{ID: SectionThisMac, Title: page.ThisMac}
	for _, m := range h.r.All.Macs {
		if c, ok := model.CapacityOf(m); ok && m.ThisMac {
			s.Caption = []page.Value{page.Text(strconv.FormatInt(c.Free>>30, 10) + " GB free")}
		}
	}
	return s
}

func (h homeReading) empty() *page.Empty {
	e := &page.Empty{Text: "No VMs yet", Glyph: symbol("shippingbox", page.SignalQuiet), Actions: []string{ActionNewVM}}
	for _, m := range h.r.All.Macs {
		if c, ok := model.CapacityOf(m); ok && m.ThisMac {
			e.Sub = strconv.FormatInt(c.Free>>30, 10) + " GB free on " + page.ThisMac
		}
	}
	return e
}

// invariant: every server of the fleet, always first of the machines: a
// server is the kind that pages you, so a fleet of one server keeps its
// section of one row.
func (h homeReading) servers() (page.Section, bool) {
	s := page.Section{ID: SectionServers, Title: "Servers", Columns: columns(), Uncounted: true}
	for _, m := range h.r.All.Macs {
		for _, vm := range m.VMs {
			if vm.Lifetime == model.AlwaysOn {
				row := h.vmRow(m, vm)
				s.Items = append(s.Items, page.Item{Row: &row})
			}
		}
	}
	return s, len(s.Items) > 0
}

// value: every VM that is no server, the running first, its head counting
// them.
func (h homeReading) vms() (page.Section, bool) {
	s := page.Section{ID: SectionVMs, Title: "VMs", Columns: columns(), Uncounted: true}
	var running, others []page.Item
	for _, m := range h.r.All.Macs {
		for _, vm := range m.VMs {
			if vm.Lifetime == model.AlwaysOn {
				continue
			}
			row := h.vmRow(m, vm)
			if runs(vm.Standing) {
				running = append(running, page.Item{Row: &row})
			} else {
				others = append(others, page.Item{Row: &row})
			}
		}
	}
	s.Items = slices.Concat(running, others)
	s.Caption = []page.Value{page.Text(strconv.Itoa(len(s.Items)))}
	return s, len(s.Items) > 0
}

// value: a VM's standing counts as running, the one that lists live keys.
func runs(s model.VMStanding) bool { return s == model.StandingRunning || s == model.StandingHealthy }

// invariant: a VM's row is its standing: the dot its tone draws and its
// word first. A running or healthy VM adds the memory its process holds
// and ends in its CPU, a spark from its Mac's readings; every other
// standing says its lifetime, or what it waits on, and lists no key.
func (h homeReading) vmRow(m model.MacView, vm model.VM) page.Row {
	row := page.Row{ID: vm.Name, Label: vm.Name, Mac: page.MacName(m.Name, h.self), Mark: page.OK, Nav: machinePage(vm.Name),
		Glyph: page.Glyph{Dot: page.DotFill, Signal: toneSignal(string(vm.Standing.Tone()))}}
	word := page.Text(vm.Standing.Word())
	readings := h.r.Samples[m.Name]
	switch vm.Standing {
	case model.StandingRunning, model.StandingHealthy:
		row.Trail = "value"
		row.Cells = []page.Value{liveOf(engine.LiveKey(engine.LiveCPUMac, vm.Name), readings)}
		row.Detail = page.List(word, liveOf(engine.LiveKey(engine.LiveMemMac, vm.Name), readings))
	case model.StandingRestarting:
		if vm.Restart == nil {
			row.Detail = page.List(word)
			break
		}
		row.Glyph = page.Glyph{Dot: page.DotRing, Progress: (vm.Restart.Step - 1) * 1000 / model.RestartSteps, Signal: page.SignalActive}
		row.Detail = page.List(word, page.Text(vm.Restart.Counted()))
	case model.StandingDown:
		row.Mark, row.Reason = page.Broken, page.Down
		if since := vm.CheckStatus.Since; !since.IsZero() {
			word = page.Text(vm.Standing.Word() + " since " + (needSaid{r: NeedsReading{Zone: h.r.Zone}}).at(since))
		}
		word.Signal = page.SignalFailing
		row.Detail = page.List(word)
	case model.StandingCreating:
		row.Glyph.Dot = page.DotRing
		row.Detail = page.List(word, pagemachines.Lifetime(vm.Lifetime))
	case model.StandingStopped, model.StandingSuspended, model.StandingNotSeen:
		row.Mark, row.Reason = page.Absent, page.Stopped
		row.Glyph.Dot = page.DotRing
		row.Detail = page.List(word, pagemachines.Lifetime(vm.Lifetime))
	}
	if h.stale != "" && page.MacName(m.Name, h.self) == h.stale {
		row.Signal = page.SignalStale
	}
	return row
}

// value: a live key's value as a row draws it, pending until its machine
// is read.
func liveOf(key string, readings []activity.Reading) page.Value {
	return live(pendingOr(LiveValue(key, readings)), key)
}

// invariant: one row per Mac: its standing's dot and word, then, while it
// is online, how many VMs it holds and its disk used, the shortest part
// dropped first, and its CPU as a spark; an asleep or unseen Mac lists no
// key. A fleet of one Mac adds the row that adds another.
func (h homeReading) macs() (page.Section, []page.Action) {
	s := page.Section{ID: SectionMacs, Title: "Macs", Columns: columns(), Uncounted: true}
	var actions []page.Action
	for _, m := range h.r.All.Macs {
		row := h.macRow(m)
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if len(h.r.All.Macs) == 1 {
		s.Items = append(s.Items, page.Item{Row: &page.Row{ID: RowAddMac, Label: "Add a Mac to run VMs on it", Mark: page.OK, Glyph: symbol("plus", page.SignalActive), Verb: ActionAddMac}})
		actions = append(actions, addMacHint(page.Action{ID: ActionAddMac, Label: "Add a Mac", On: page.OnRow, Rows: []string{RowAddMac}, Priority: 4}))
	}
	return s, actions
}

func (h homeReading) macRow(m model.MacView) page.Row {
	glyph := "desktop"
	if m.Laptop {
		glyph = "laptop"
	}
	row := page.Row{ID: m.Name, Label: model.Label(*h.r.All, m), Mark: page.OK, Nav: machinePage(m.Name),
		Glyph: page.Glyph{Kind: glyph, Signal: toneSignal(string(m.Standing.Tone()))}}
	word := page.Text(m.Standing.Word())
	if m.Standing != model.MacOnline {
		row.Detail = page.List(word)
		if m.Standing == model.MacAsleep {
			row.Signal = page.SignalAsleep
			row.Detail = page.List(word, page.Text("seen "+minutes(h.now.Sub(m.Answered))+" ago"))
		}
		return row
	}
	readings := h.r.Samples[m.Name]
	row.Quick = ActionNewVMOn + m.Name
	row.Trail = "value"
	row.Cells = []page.Value{liveOf(engine.LiveKey(engine.LiveCPU, m.Name), readings)}
	disk := liveOf(engine.LiveKey(engine.LiveDisk, m.Name), readings)
	disk.Label = "Disk"
	row.Detail = page.List(append([]page.Value{word}, shortestFirst(page.Text(counted(len(m.VMs), "VM", "VMs")), disk)...)...)
	if h.stale != "" && page.MacName(m.Name, h.self) == h.stale {
		row.Signal = page.SignalStale
	}
	return row
}

// value: the parts after a standing's word, each dropped by its length,
// the shortest first: the highest priority goes to the shortest.
func shortestFirst(parts ...page.Value) []page.Value {
	order := slices.Clone(parts)
	slices.SortStableFunc(order, func(a, b page.Value) int { return cmp.Compare(len(partWords(b)), len(partWords(a))) })
	out := slices.Clone(parts)
	for pri, v := range order {
		for i := range out {
			if partWords(out[i]) == partWords(v) {
				out[i].Pri = pri + 1
			}
		}
	}
	return out
}

// value: a part's words as the line counts them: its label and its text,
// or its unit's room for a number ("Disk 83 %").
func partWords(v page.Value) string {
	if v.Kind == page.KindText {
		return v.Text
	}
	return v.Label + " 000 %"
}

// value: the rows under the machines, each its glyph, its word and the
// page it opens: Keep awake with its switch, then Ports and Accounts.
func (h homeReading) shortcuts() (page.Section, page.Action) {
	toggle := h.awakeToggle()
	awakeRow := page.Row{ID: RowAwake, Label: "Keep awake", Mark: page.OK, Glyph: symbol("sun.max", page.SignalQuiet), Nav: AwakeID, Verb: toggle.ID, Trail: "value"}
	awakeRow.Cells = []page.Value{page.Pending()}
	if r := h.r.Awake; r != nil {
		awakeRow.Cells = []page.Value{page.Text(cmpWord(awakeWords(*r), "off"))}
	}
	items := []page.Item{{Row: &awakeRow}, {Row: h.portsRow()}, {Row: h.accountsRow()}}
	return page.Section{ID: SectionShortcuts, Title: "Shortcuts", Columns: columns(), Uncounted: true, Items: items}, toggle
}

// value: the Ports row: how many ports are shared (each shared port, each
// declared public, each a server's own device carries); red what is
// exposed, or what a server down leaves unreachable; what a server
// restarting is starting.
func (h homeReading) portsRow() *page.Row {
	row := &page.Row{ID: RowPorts, Label: "Ports", Mark: page.OK, Glyph: symbol("network", page.SignalQuiet), Nav: PortsID, Trail: "value"}
	if h.r.Ports == nil {
		row.Cells = []page.Value{page.Pending()}
		return row
	}
	shared, exposed, unreachable, starting := 0, 0, 0, 0
	for _, m := range h.r.Ports.Macs {
		shared += len(m.Shared)
		for _, p := range m.Ports {
			switch {
			case p.Red():
				exposed++
			case p.Reach == ports.ThisMachine:
			case p.Declared != nil || p.Device:
				shared++
				if vm, ok := h.vm(p.Machine.Name); ok && vm.Restart != nil {
					starting++
				} else if ok && model.Down(vm) {
					unreachable++
				}
			}
		}
	}
	cell := page.Text(strconv.Itoa(shared) + " shared")
	switch {
	case exposed > 0:
		cell = page.Text(strconv.Itoa(exposed) + " exposed")
		cell.Signal = page.SignalFailing
	case unreachable > 0:
		cell = page.Text(strconv.Itoa(unreachable) + " unreachable")
		cell.Signal = page.SignalFailing
	case starting > 0:
		cell = page.Text(strconv.Itoa(starting) + " starting")
	}
	row.Cells = []page.Value{cell}
	return row
}

func (h homeReading) vm(name string) (model.VM, bool) {
	for _, m := range h.r.All.Macs {
		for _, vm := range m.VMs {
			if vm.Name == name {
				return vm, true
			}
		}
	}
	return model.VM{}, false
}

// value: the Accounts row: what is signed out on any Mac, as its upkeep
// says it, else how many passwords are granted to machines.
func (h homeReading) accountsRow() *page.Row {
	row := &page.Row{ID: RowAccounts, Label: "Accounts", Mark: page.OK, Glyph: symbol("key", page.SignalQuiet), Nav: AccountsID, Trail: "value"}
	out := 0
	for _, n := range h.upkeep {
		if n.Finding.Section == section.Accounts {
			out++
		}
	}
	switch {
	case out > 0:
		cell := page.Text(strconv.Itoa(out) + " signed out")
		cell.Signal = page.SignalWaits
		row.Cells = []page.Value{cell}
	case h.r.SignIns == nil:
		row.Cells = []page.Value{page.Pending()}
	default:
		row.Cells = []page.Value{page.Text(strconv.Itoa(h.r.SignIns.grants()) + " granted")}
	}
	return row
}

// invariant: the Keep awake switch is one click: it lets This Mac sleep
// while a keep-awake holds, else keeps it awake for the default term; the
// other terms are its menu's.
func (h homeReading) awakeToggle() page.Action {
	a := awakeAction(h.r.Awake)
	a.Label, a.Rows = "Keep "+page.ThisMac+" awake", []string{RowAwake}
	if a.ID == ActionAwakeOff {
		a.Picked = true
		return a
	}
	a.Form = nil
	a.Run.In = map[string]string{"For": strconv.FormatInt(int64(4*time.Hour), 10)}
	return a
}
