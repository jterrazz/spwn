package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/activity"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"

	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
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
	// value: a terminal on one family of a workspace (`open:apps`).
	ActionOpenFamily = "open:"

	SectionMachine  = "machine"
	SectionActivity = "activity"
	SectionInside   = "inside"
	SectionPorts    = "ports"
	SectionShared   = "shared"
	SectionUp       = "up"
	SectionHealth   = "health"
	SectionServer   = "server"
	SectionVMs      = "vms"
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
	Now      time.Time
	Asks     Levels
}

func MachinePage(m Machine) page.Page {
	// invariant: SPEC §10.2's page of one machine: a workspace's, a server's or
	// a Mac's, each with only the actions its kind takes, each naming its call
	// and asking at the level the call asks.
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

func vmPage(m Machine, h model.MacView, vm model.VM) page.Page {
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
	sections := []page.Section{{ID: SectionMachine, Title: vm.Name, Role: page.Summary, Caption: caption, Columns: columns(), Items: []page.Item{{Row: &head}}}}
	p := page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: vm.Name, Kind: string(kind), On: on}}
	samples := m.Samples
	if vm.State != model.Running {
		samples = nil
	}
	if kind == machine.Server {
		sections = append(sections, upSection(m, vm), healthSection(vm, m.Now), activitySection(samples, vm.State == model.Running, "Activity", vm.Name), serverSection(m))
		restart := page.Action{ID: ActionRestart, Label: "Restart", On: page.OnPage, Priority: 1, Mutates: true, Confirm: ask(m.Asks, "restart", kind, vm.Name, "Restart "+vm.Name+"?", "restart"),
			Run: &page.Run{Call: CallRestartVM, In: map[string]string{"Name": vm.Name}, Target: target(h)}}
		restart.Run.In = answered(restart.Confirm, restart.Run.In)
		stop := page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 2, Mutates: true,
			Confirm: ask(m.Asks, "stop", kind, vm.Name, "Stop "+vm.Name+"?", "stop", "It stays stopped until you start it: the engine no longer restarts it."),
			Run:     &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
		stop.Run.In = answered(stop.Confirm, stop.Run.In)
		p.Actions = []page.Action{restart, stop}
	} else {
		sections = append(sections, activitySection(samples, vm.State == model.Running, "Activity", vm.Name), insideSection(m.Summary), portsSection(m, h, vm), sharedSection(m, h, vm))
		stop := page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 1, Mutates: true,
			Confirm: ask(m.Asks, "stop", kind, vm.Name, "Stop "+vm.Name+"?", "stop", "Its disk is kept."),
			Run:     &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
		stop.Run.In = answered(stop.Confirm, stop.Run.In)
		if vm.State != model.Running && !model.Down(vm) {
			stop = page.Action{ID: ActionStartVM, Label: "Start", On: page.OnPage, Priority: 1, Mutates: true,
				Run: &page.Run{Call: CallStartVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
		}
		del := page.Action{ID: ActionDelete, Label: "Delete", On: page.OnPage, Priority: 2, Mutates: true,
			Confirm: ask(m.Asks, "delete", kind, vm.Name, "Delete "+vm.Name+"?", "delete", "Its disk is erased: nothing of it is kept."),
			Run:     &page.Run{Call: CallDeleteVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}}
		del.Run.In = answered(del.Confirm, del.Run.In)
		p.Actions = []page.Action{stop, del}
		if vm.State == model.Running {
			p.Actions = append(p.Actions, page.Action{ID: ActionShare, Label: "Share a port", On: page.OnPage, Priority: 3, Mutates: true,
				Run:  &page.Run{Call: CallSharePort, In: map[string]string{"VM": vm.Name}, Target: target(h)},
				Form: &page.Form{Title: "Share a port of " + vm.Name, Fields: []page.Field{{Key: "Port", Label: "Port", Kind: page.FieldText, Required: true, Note: "a port number, like 5173"}}}})
		}
		p.Actions = append(p.Actions, takeBacks(m, vm)...)
		if vm.State == model.Running {
			p.Actions = append(p.Actions, insideActions(vm.Name)...)
		}
		p.Actions = append(p.Actions, stopSharings(m, vm)...)
	}
	p.Actions = append(p.Actions, openIn("--on", vm.Name))
	p.Sections = sections
	p.Live = liveKeys(p)
	return page.Seal(p)
}

// value: Activity's three measures of the machine, as its last two
// readings give them (its disk as the share used, vision panels c and
// f), each led into its series over the readings kept (0 to 100: CPU's
// share, memory's and disk's use of their total), so a tile draws its bar
// meter; pending until they are read, none while the machine does not run.
func activitySection(readings []activity.Reading, running bool, title, machine string) page.Section {
	cpu, memory, disk := page.Pending(), page.Pending(), page.Pending()
	if !running {
		cpu, memory, disk = page.None(), page.None(), page.None()
	}
	m := measured(readings)
	cpus, memories, disks := m.cpus, m.memories, m.disks
	if len(readings) > 0 {
		memory, disk = page.None(), page.None()
		if m.cpu.Kind != "" {
			cpu = m.cpu
		}
		if m.memory.Kind != "" {
			memory = m.memory
		}
		if m.disk.Kind != "" {
			disk = m.disk
		}
	}
	if running {
		cpu, memory, disk = live(cpu, LiveKey(LiveCPU, machine)), live(memory, LiveKey(LiveMemory, machine)), live(disk, LiveKey(LiveDisk, machine))
	}
	measure := func(id, label string, now page.Value, series []int) page.Row {
		row := page.Row{ID: id, Label: label, Cells: []page.Value{now}}
		if len(series) > 1 {
			row.Cells = append(row.Cells, page.Spark(series, page.None()))
		}
		return row
	}
	rows := []page.Row{measure("cpu", "CPU", cpu, cpus), measure("memory", "Memory", memory, memories), measure("disk", "Disk", disk, disks)}
	cols := append(columns(), page.Column{Key: "graph", Kind: page.KindSpark, Keep: 50})
	return page.Section{ID: SectionActivity, Title: title, Role: page.Summary, Columns: cols, Items: items(rows)}
}

func items(rows []page.Row) []page.Item {
	out := make([]page.Item, 0, len(rows))
	for i := range rows {
		out = append(out, page.Item{Row: &rows[i]})
	}
	return out
}

// value: a workspace's Apps, Settings and Disk, as its Overview counts them
// inside it: station manages them there.
func insideSection(s *overview.Summary) page.Section {
	rows := []page.Row{{ID: "apps", Label: "Apps"}, {ID: "settings", Label: "Settings"}, {ID: "disk", Label: "Disk"}}
	for i := range rows {
		rows[i].Cells = []page.Value{page.Pending()}
		if s == nil {
			continue
		}
		rows[i].Cells = []page.Value{page.None()}
		for _, l := range s.Lines {
			if string(l.Section) == rows[i].ID {
				rows[i].Cells = []page.Value{inside(rows[i].ID, lineOf(*s, rows[i].ID))}
			}
		}
	}
	return page.Section{ID: SectionInside, Title: "Inside", Role: page.Summary, Columns: columns(), Items: items(rows)}
}

// value: an Inside line in the vision's few words (`38 installed`, `2
// need a look`, `12.4 GB free`): what is there and what needs you, never
// what station does not manage.
func inside(id string, v page.Value) page.Value {
	parts := v.Items
	if v.Kind != page.KindList {
		parts = []page.Value{v}
	}
	var out []page.Value
	look := 0
	for _, p := range parts {
		switch {
		case p.Label == string(page.Attention) || p.Label == string(page.Broken):
			switch {
			case p.Kind == page.KindCount:
				look += int(p.N)
			case id == "disk" && p.Kind == page.KindSize:
				out = append(out, p)
			default:
				look++
			}
		case id == "apps" && p.Label == string(page.OK):
			out = append(out, page.Count(p.N, "installed"))
		case id == "settings" && p.Label == string(page.Absent) && p.Kind == page.KindText:
			out = append(out, page.Text(p.Text+" set"))
		case id == "disk" && p.Kind == page.KindSize && p.Unit == "free":
			out = append(out, p)
		}
	}
	if look > 0 {
		out = append(out, page.Text(strconv.Itoa(look)+" "+pluralVerb(look, "needs", "need")+" a look"))
	}
	if len(out) == 0 {
		return v
	}
	return page.List(out...)
}

func pluralVerb(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// value: what opening Apps, Settings or Disk on a workspace's page does: a
// terminal on that family, on the workspace.
func insideActions(vm string) []page.Action {
	ids := []string{"apps", "settings", "disk"}
	out := make([]page.Action, 0, len(ids))
	for _, id := range ids {
		out = append(out, page.Action{ID: ActionOpenFamily + id, Label: "Open " + id, On: page.OnRow, Rows: []string{id}, Priority: 5, Run: &page.Run{Terminal: true, Argv: []string{id, "--on", vm}}})
	}
	return out
}

func lineOf(s overview.Summary, id string) page.Value {
	for _, body := range pagehome.Body(s, true) {
		for _, it := range body.Items {
			if body.ID == "sections" && it.Row != nil && it.Row.ID == id && len(it.Row.Cells) > 0 {
				return it.Row.Cells[0]
			}
		}
	}
	return page.None()
}

// value: the workspace's shared ports, each a link: private, on Tailscale;
// one with no profile is TCP, or Web when it has a web link.
func portsSection(m Machine, _ model.MacView, vm model.VM) page.Section {
	s := page.Section{ID: SectionPorts, Title: "Private", Caption: []page.Value{page.Text("shared on " + catalog.TailnetProduct)}, Uncounted: true}
	if m.Ports == nil {
		s.Items = []page.Item{{Row: &page.Row{ID: "reading", Label: "reading", Mark: page.Awaiting}}}
		return s
	}
	for _, h := range m.Ports.Macs {
		for _, sv := range h.Shared {
			if sv.VM != vm.Name {
				continue
			}
			value := page.Text(h.PublicName + ":" + strconv.Itoa(sv.Reserved))
			if sv.URLs.Public != "" {
				value = page.URL(sv.URLs.Public)
			}
			label := cmpWord(titled(sv.Profile), "TCP")
			if sv.Profile == "" && strings.HasPrefix(sv.URLs.Public, "http") {
				label = "Web"
			}
			s.Items = append(s.Items, page.Item{Link: &page.Link{Label: label, Lead: page.List(page.Text(label), page.Port(int64(sv.GuestPort))), Value: value, Row: strconv.Itoa(sv.GuestPort)}})
		}
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No port shared yet"}}}
	}
	return s
}

// value: Stop sharing on each of the workspace's shared ports, the row
// its link is.
func stopSharings(m Machine, vm model.VM) []page.Action {
	if m.Ports == nil {
		return nil
	}
	var out []page.Action
	for _, h := range m.Ports.Macs {
		for _, sv := range h.Shared {
			if sv.VM != vm.Name {
				continue
			}
			a := stopSharing(sv, h.Name)
			row := strconv.Itoa(sv.GuestPort)
			a.ID, a.Rows = ActionStopSharing+vm.Name+"/"+row, []string{row}
			out = append(out, a)
		}
	}
	return out
}

// value: the accounts This Mac shared with the workspace, each with what
// it may do, its time left and its Take back.
func sharedSection(m Machine, h model.MacView, vm model.VM) page.Section {
	from := model.Label(*m.All, h)
	s := page.Section{ID: SectionShared, Title: "Accounts shared from " + from, Columns: columns()}
	for _, a := range sharedAccounts(m, vm.Name) {
		left := a.Shared.Until.Sub(m.Now)
		words := make([]page.Value, 0, len(a.Scopes)+1)
		for _, scope := range a.Scopes {
			if !strings.Contains(scope, ":") {
				words = append(words, page.Text(scope))
			}
		}
		row := page.Row{ID: a.Provider, Label: accounts.Label(a.Provider), Mark: page.OK, Cells: []page.Value{page.List(append(words, page.Text(hours(left)+" left"))...)}}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if len(s.Items) == 0 {
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No account shared"}}}
	}
	return s
}

func sharedAccounts(m Machine, vm string) []accounts.Account {
	if m.Accounts == nil {
		return nil
	}
	var out []accounts.Account
	for _, mc := range m.Accounts.Machines {
		if mc.Name != vm {
			continue
		}
		for _, a := range mc.Accounts {
			if a.Shared != nil {
				out = append(out, a)
			}
		}
	}
	return out
}

func hours(d time.Duration) string {
	if d < time.Hour {
		return strconv.Itoa(max(int(d/time.Minute), 1)) + " min"
	}
	return strconv.Itoa(int(d/time.Hour)) + " h"
}

func takeBacks(m Machine, vm model.VM) []page.Action {
	shared := sharedAccounts(m, vm.Name)
	out := make([]page.Action, 0, len(shared))
	for _, a := range shared {
		label := accounts.Label(a.Provider)
		c := ask(m.Asks, "take-back", machine.Workspace, label, "Take back "+label+" from "+vm.Name+"?", "take back", "Its copy on "+vm.Name+" is deleted.")
		out = append(out, page.Action{ID: ActionTakeBack + "/" + a.Provider, Label: "Take back", On: page.OnRow, Rows: []string{a.Provider}, Priority: 4, Mutates: true, Confirm: c,
			Run: &page.Run{Call: CallTakeBack, In: answered(c, map[string]string{"Account": a.Provider, "From": vm.Name})}})
	}
	return out
}

// value: a server's Up as vision panel f's figure: days and hours (`12 d
// 4 h`), since it booted; its restarts beside it.
func upSection(m Machine, vm model.VM) page.Section {
	up := page.Row{ID: "up", Label: "Up", Cells: []page.Value{page.None()}}
	if m.Summary != nil && m.Summary.Head.Up > 0 {
		up.Cells = []page.Value{page.List(page.Text(span(m.Summary.Head.Up)), page.Text("since boot"))}
	}
	words := vm.Restarts.Words(m.Now)
	restarts := page.Row{ID: "restarts", Label: "Restarts", Cells: []page.Value{page.List(page.Count(int64(vm.Restarts.Count), plural(vm.Restarts.Count, "restart")), page.Text(words[len(words)-1]))}}
	return page.Section{ID: SectionUp, Title: "Overview", Role: page.Summary, Columns: columns(), Items: items([]page.Row{up, restarts})}
}

func span(d time.Duration) string {
	day := 24 * time.Hour
	switch {
	case d >= day && d%day >= time.Hour:
		return strconv.Itoa(int(d/day)) + " d " + strconv.Itoa(int(d%day/time.Hour)) + " h"
	case d >= day:
		return strconv.Itoa(int(d/day)) + " d"
	case d >= time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + " h"
	}
	return strconv.Itoa(max(int(d/time.Minute), 1)) + " min"
}

// value: a server's health steps, run after every boot, how long ago its
// check last ran, and what a passing check answered (vision panel f).
func healthSection(vm model.VM, now time.Time) page.Section {
	caption := []page.Value{page.Text("after every boot")}
	if at := vm.CheckStatus.At; !at.IsZero() && !now.IsZero() {
		caption = append(caption, page.Age(at, now))
	}
	steps := pagemachines.Steps(vm)
	for i := range steps {
		if steps[i].ID == "check" && steps[i].Mark == page.OK && vm.CheckStatus.Answer != "" {
			steps[i].Cells = []page.Value{page.List(page.Text(vm.Check), page.Text(vm.CheckStatus.Answer))}
		}
	}
	return page.Section{ID: SectionHealth, Title: "Health", Caption: caption, Columns: columns(), Items: items(steps)}
}

// value: what a server's page adds, as its Overview reads it: its ports'
// counts and the command that opens a shell on it, for debugging; station
// manages nothing inside it.
func serverSection(m Machine) page.Section {
	s := page.Section{ID: SectionServer, Title: "Ports and shell", Role: page.Summary, Columns: columns()}
	if m.Summary == nil {
		s.Items = []page.Item{{Row: &page.Row{ID: "reading", Label: "reading", Cells: []page.Value{page.Pending()}}}}
		return s
	}
	for _, body := range pagehome.Body(*m.Summary, true) {
		if body.ID != "sections" {
			continue
		}
		for _, it := range body.Items {
			if it.Row == nil || it.Row.ID != string(SectionActivity) {
				s.Items = append(s.Items, it)
			}
		}
	}
	return s
}

// value: a Mac's page: its readiness checklist, its VMs, its Needs you;
// Unlock… while it waits for its disk's password.
func macPage(m Machine, h model.MacView) page.Page {
	label := model.Label(*m.All, h)
	state := page.Row{ID: "state", Label: "State", Mark: page.OK, Cells: []page.Value{page.Text("Engine running")}}
	switch {
	case !h.Reachable && !h.ThisMac:
		state.Mark, state.Reason, state.Cells = page.Broken, page.Unreachable, []page.Value{page.Text("Unreachable")}
	case !h.Engine.Running:
		state.Mark, state.Reason, state.Cells = page.Attention, page.Stopped, []page.Value{page.Text("Engine stopped")}
	}
	sections := []page.Section{{ID: SectionMachine, Title: label, Role: page.Summary, Caption: []page.Value{page.Text(KindMac), page.Text(h.Name)}, Columns: columns(), Items: []page.Item{{Row: &state}}}}
	if m.Summary != nil {
		for _, body := range pagehome.Body(*m.Summary, true) {
			if body.ID == SectionNeeds || body.ID == "readiness" {
				sections = append(sections, body)
			}
		}
	}
	vms := Tree(&model.AllMachines{Macs: []model.MacView{h}, Read: m.All.Read}, m.Ports, func(model.MacView) page.Value { return page.None() })
	vms.ID, vms.Title = SectionVMs, "VMs"
	if len(vms.Items) == 1 && vms.Items[0].Group != nil {
		vms.Items = vms.Items[0].Group.Items
	}
	if len(vms.Items) == 0 {
		vms.Items = []page.Item{{Empty: &page.Empty{Noun: "VMs"}}}
	}
	sections = append(sections, vms)
	p := page.Page{ID: MachineID, Command: "menubar", Context: page.Context{Machine: h.Name, Kind: string(machine.Mac)}, Sections: sections}
	if h.ThisMac {
		for _, s := range sections {
			if s.ID != SectionNeeds {
				continue
			}
			p.Actions = append(p.Actions, updateAll(s, m.Summary)...)
		}
	}
	if locked(m, h) {
		p.Actions = append(p.Actions, page.Action{ID: ActionUnlock, Label: "Unlock the disk", On: page.OnPage, Priority: 1, Mutates: true, Run: &page.Run{Terminal: true, Argv: []string{"machines", "unlock", label}}})
	}
	if !h.ThisMac {
		p.Actions = append(p.Actions, openIn("--on", label))
	} else {
		p.Actions = append(p.Actions, openIn())
	}
	return page.Seal(p)
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
