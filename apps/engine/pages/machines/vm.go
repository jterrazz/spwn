package machines

import (
	"strconv"
	"strings"
	"time"

	coresection "github.com/jterrazz/studio-station/apps/engine/core/section"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pagehome "github.com/jterrazz/studio-station/apps/engine/pages/home"
)

type VMOverview struct {
	All model.AllMachines
	Mac model.MacView
	VM  model.VM
}

func (o VMOverview) on() string {
	if o.Mac.ThisMac {
		return "This Mac"
	}
	return model.Label(o.All, o.Mac)
}

func (o VMOverview) unreachable() bool { return !o.Mac.Reachable }

func VMPage(o VMOverview, s *overview.Summary, thisMac page.Context, all bool) page.Page {
	// invariant: a VM's Overview, read from its Mac: the line naming the
	// machine, its first line from All machines, then its sections as it reads
	// them inside (nil while it does not run or does not answer), then health.
	vm := o.VM
	on := page.OnMachine(vm.Name)
	if o.unreachable() {
		row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.Broken, Reason: page.Unreachable, Cells: []page.Value{page.Text("is on " + o.on() + ", which does not answer")}}
		return page.Seal(page.Page{ID: "overview", Command: "overview", Context: thisMac, Sections: []page.Section{on, {
			ID: "overview", Title: vm.Name, Role: page.Banner, Uncounted: true, Columns: vmColumns(), Items: []page.Item{{Row: &row}},
		}}})
	}
	sections := []page.Section{on, {ID: "overview", Title: vm.Name, Caption: vmCaption(o, s), Joined: true, Role: page.Banner, Uncounted: true}}
	state := page.Ready
	var hints []page.Hint
	if s != nil {
		sections = append(sections, pagehome.Body(*s, all)...)
		hints = pagehome.Hints(*s, all)
		if s.Pending {
			state = page.Loading
		}
	}
	sections = append(sections, page.Section{ID: "health", Title: "Health", Uncounted: true, Columns: vmColumns(), Items: healthItems(vm, readsSSH(s), o.All.Read)})
	return page.Seal(page.Page{ID: "overview", Command: "overview", Context: thisMac, State: state, Sections: sections, Hints: hints})
}

// value: SPEC §5.1's first line of a VM: its kind and its Mac, its
// lifetime, then how long it has run (a workspace) or its state (a server).
func vmCaption(o VMOverview, s *overview.Summary) []page.Value {
	vm := o.VM
	var head overview.Head
	if s != nil {
		head = s.Head
	}
	server := vm.Lifetime == model.AlwaysOn
	parts := []page.Value{page.Text(model.KindWord(vm.Lifetime.Kind()) + " on " + o.on()), page.Text(lifetimeWord(vm.Lifetime))}
	switch {
	case server:
		mark := page.OK
		if vm.State != model.Running {
			mark = page.Absent
		}
		parts = append(parts, page.Labeled(string(mark), page.Text(stateWord(vm.State))))
	case vm.State == model.Running && head.Up > 0:
		parts = append(parts, pagehome.Up(head.Up))
	default:
		parts = append(parts, page.Text(stateWord(vm.State)))
	}
	return parts
}

func Steps(vm model.VM) []page.Row {
	// value: SPEC §7.2's health steps of a VM, in their order: running, an
	// address, ssh answering, then a server's own check when it has one.
	health := model.HealthOf(vm)
	steps := []page.Row{
		step("running", model.StepRunning, health.Running, health.Running, ""),
		step("address", model.StepAddress, health.Address, health.Running, vm.IP),
		step(string(coresection.SSH), model.StepSSH, health.SSH, health.Running, ""),
	}
	if vm.Lifetime == model.AlwaysOn && vm.Check != "" {
		steps = append(steps, checkStep(vm, health.Running))
	}
	return steps
}

func healthItems(vm model.VM, sshRead bool, now time.Time) []page.Item {
	steps := Steps(vm)
	if vm.Lifetime == model.AlwaysOn {
		restarts := page.Row{ID: "restarts", Label: "Restarts", Mark: page.OK, Cells: []page.Value{page.Texts(vm.Restarts.Words(now)...)}}
		if vm.Restarts.Looping {
			restarts.Mark, restarts.Reason = page.Broken, page.Failing
		}
		steps = append(steps, restarts)
	}
	items := make([]page.Item, 0, len(steps)+1)
	for i := range steps {
		items = append(items, page.Item{Row: &steps[i]})
	}
	r := vm.Resources
	size := page.Row{ID: "size", Label: "Size", Mark: page.OK, Cells: []page.Value{page.List(page.Text(strconv.Itoa(r.CPUs)+" CPU"), Memory(r.MemoryBytes), DiskSize(r.DiskBytes))}}
	items = append(items, page.Item{Row: &size})
	if vm.Lifetime == model.AlwaysOn {
		items = append(items, serverRows(vm, sshRead)...)
	}
	return items
}

// invariant: a server's Overview names its own tailnet device when its
// tooling installed one, which station never installs nor blocks, and the
// ssh that is its one door, for debugging (SPEC §7.7), here only when the
// server's own SSH line was not read inside it.
func serverRows(vm model.VM, sshRead bool) []page.Item {
	var rows []page.Row
	if vm.Device != "" {
		rows = append(rows, page.Row{ID: "device", Label: "Own " + catalog.TailnetProduct + " device", Mark: page.OK, Cells: []page.Value{page.Text(vm.Device)}})
	}
	if !sshRead {
		rows = append(rows, page.Row{ID: string(coresection.SSH), Label: "SSH", Mark: page.OK, Cells: []page.Value{page.List(page.Command(catalog.SSHCommand("", vm.Name)), page.Text("for debugging"))}})
	}
	items := make([]page.Item, 0, len(rows))
	for i := range rows {
		items = append(items, page.Item{Row: &rows[i]})
	}
	return items
}

func readsSSH(s *overview.Summary) bool {
	if s == nil {
		return false
	}
	for _, l := range s.Lines {
		if l.Section == coresection.SSH && l.Unread == "" {
			return true
		}
	}
	return false
}

func vmColumns() []page.Column {
	return []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "detail", Kind: page.KindList, Keep: 50}}
}

// value: SPEC §7.2's fourth step: a check that never passed since boot
// waits (not a failure), one that passed then fails says since when.
func checkStep(vm model.VM, running bool) page.Row {
	row := page.Row{ID: "check", Label: model.StepCheck, Mark: page.OK, Cells: []page.Value{page.Text(vm.Check)}}
	switch {
	case !running:
		row.Mark, row.Reason, row.Cells[0] = page.Absent, page.Stopped, page.None()
	case vm.CheckStatus.State == model.CheckFailing:
		row.Label, row.Mark, row.Reason = "Check failing", page.Attention, page.Failing
		row.Cells[0] = page.List(page.Text("since "+vm.CheckStatus.Since.Local().Format("15:04")), page.Text(vm.Check))
	case vm.CheckStatus.State != model.CheckPassing:
		row.Label, row.Mark, row.Reason = "Check waiting", page.Attention, page.Waiting
	}
	return row
}

// value: a step that did not pass fails while the VM runs; one of a VM
// that does not run is not taken.
func step(id, label string, passed, running bool, detail string) page.Row {
	row := page.Row{ID: id, Label: label, Mark: page.OK, Cells: []page.Value{page.None()}}
	if detail != "" {
		row.Cells[0] = page.Text(detail)
	}
	switch {
	case passed:
	case running:
		row.Mark, row.Reason, row.Cells[0] = page.Broken, page.Failing, page.Text("failing")
	default:
		row.Mark, row.Reason = page.Absent, page.Stopped
	}
	return row
}

func lifetimeWord(l model.Lifetime) string {
	if l == model.AlwaysOn {
		return l.Word()
	}
	return titled(string(l))
}

func Lifetime(l model.Lifetime) page.Value {
	// value: a VM's lifetime as a page says it (`Always on`, `Ephemeral`).
	return page.Text(titled(lifetimeWord(l)))
}

func State(s model.VMState) page.Value { return page.Text(stateWord(s)) }

func titled(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

func stateWord(s model.VMState) string {
	switch s {
	case model.Running:
		return "Running"
	case model.Stopped:
		return "Stopped"
	case model.Suspended:
		return "Suspended"
	case model.Stale:
		return "Ended unexpectedly"
	case model.Broken:
		return "In a loop"
	case model.Creating:
		return "Creating"
	case model.UnknownState:
	}
	return "Unknown"
}

func DiskSize(n int64) page.Value {
	if n%1e9 == 0 {
		return page.Text(strconv.FormatInt(n/1e9, 10) + " GB")
	}
	return page.Text(strconv.FormatFloat(float64(n)/1e9, 'f', 1, 64) + " GB")
}

type VMHealthJSON struct {
	Running bool `json:"running"`
	Address bool `json:"address"`
	SSH     bool `json:"ssh"`
	// value: a server's check (SPEC §7.2); a workspace has none.
	Check *VMCheckJSON `json:"check"`
}

type VMCheckJSON struct {
	URL    string `json:"url"`
	OK     bool   `json:"ok"`
	Status int    `json:"status,omitempty"`
	MS     int    `json:"ms,omitempty"`
	At     string `json:"at,omitempty"`
	// value: when a check that passed began to fail.
	Since string `json:"since,omitempty"`
}

type VMMachineJSON struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	On        string `json:"on"`
	Reachable bool   `json:"reachable"`
}

type VMOverviewJSON struct {
	Schema   string        `json:"schema"`
	Machine  VMMachineJSON `json:"machine"`
	Lifetime string        `json:"lifetime"`
	State    string        `json:"state"`
	Wanted   string        `json:"wanted"`
	Health   VMHealthJSON  `json:"health"`
	Restarts int           `json:"restarts"`
	// value: the seconds a server waited before its last restart.
	RestartedAfter int            `json:"restarted_after_s,omitempty"`
	Size           model.ListSize `json:"size"`
	// value: a server's own tailnet device; empty for none.
	Device string `json:"device,omitempty"`
	Detail string `json:"detail,omitempty"`
	// value: what the VM reads of itself inside: its findings and one count
	// set per section, as its own Overview carries them.
	Findings []finding.JSON                `json:"findings"`
	Sections []pagehome.SummarySectionJSON `json:"sections"`
}

func VMContract(o VMOverview, s *overview.Summary, schema string) VMOverviewJSON {
	vm, h := o.VM, model.HealthOf(o.VM)
	inside := pagehome.SummaryJSON{Machine: pagehome.SummaryMachineJSON{Kind: "workspace"}, Findings: []finding.JSON{}, Sections: []pagehome.SummarySectionJSON{}}
	if s != nil {
		inside = pagehome.SummaryContract(*s, schema)
	}
	return VMOverviewJSON{
		Schema: schema, Machine: VMMachineJSON{Name: vm.Name, Kind: string(vm.Lifetime.Kind()), On: o.Mac.Name, Reachable: o.Mac.Reachable},
		Findings: inside.Findings, Sections: inside.Sections,
		Lifetime: string(vm.Lifetime), State: string(vm.State), Wanted: model.Wanted(vm),
		Health: VMHealthJSON{Running: h.Running, Address: h.Address, SSH: h.SSH, Check: checkJSON(vm)}, Restarts: vm.Restarts.Count, RestartedAfter: int(vm.Restarts.Wait.Seconds()),
		Size:   model.ListSize{CPUs: vm.Resources.CPUs, MemoryBytes: vm.Resources.MemoryBytes, DiskBytes: vm.Resources.DiskBytes},
		Device: vm.Device, Detail: vm.Detail,
	}
}

func checkJSON(vm model.VM) *VMCheckJSON {
	if vm.Lifetime != model.AlwaysOn || vm.Check == "" {
		return nil
	}
	c := &VMCheckJSON{URL: vm.Check, OK: vm.State == model.Running && vm.CheckStatus.State == model.CheckPassing}
	if vm.CheckStatus.State == model.CheckFailing {
		c.Since = vm.CheckStatus.Since.UTC().Format(time.RFC3339)
	}
	return c
}

func SSHPage(o VMOverview, target model.SSHTarget, thisMac page.Context) page.Page {
	// invariant: a server's SSH section (SPEC §3.2, §7.7): its one door, the
	// path that reaches it from This Mac, and what station leaves to the
	// owner's tooling; station opens it for debugging only.
	vm, mac := o.VM, o.Mac
	door := page.Row{ID: "command", Label: target.Command, Mark: page.OK, Cells: []page.Value{page.Text("from any Mac: station's SSH block hops through " + o.on())}}
	editor := page.Row{ID: "editor", Label: "In an editor", Mark: page.OK, Cells: []page.Value{page.Text("Remote SSH to " + vm.Name)}}
	reached := []page.Row{
		{ID: "to-mac", Label: "This Mac to " + o.on(), Mark: page.OK, Cells: []page.Value{page.List(page.Text("over "+catalog.TailnetProduct), page.Text("as "+mac.User))}},
		{ID: "to-vm", Label: o.on() + " to " + vm.Name, Mark: page.OK, Cells: []page.Value{page.List(page.Text(vm.IP), page.Text("its Mac's private network"))}},
	}
	if mac.ThisMac {
		reached = reached[1:]
	}
	if vm.State != model.Running {
		door.Mark, door.Reason = page.Absent, page.Stopped
		door.Cells = []page.Value{page.Text(vm.Name + " is stopped: station machines start " + vm.Name)}
	}
	rows := func(rs ...page.Row) []page.Item {
		items := make([]page.Item, 0, len(rs))
		for i := range rs {
			items = append(items, page.Item{Row: &rs[i]})
		}
		return items
	}
	columns := []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "detail", Kind: page.KindList, Keep: 50}}
	return page.Seal(page.Page{ID: page.ID(coresection.SSH), Command: "machines ssh " + vm.Name, Context: thisMac, Sections: []page.Section{
		{ID: "door", Title: "SSH", Caption: []page.Value{page.Text(vm.Name)}, Uncounted: true, Columns: columns, Items: append(rows(door, editor), page.Item{Note: &page.Note{Text: "No password: your key. Files travel through the same name; no folder of a Mac is shared."}})},
		{ID: "reached", Title: "How it is reached", Uncounted: true, Columns: columns, Items: rows(reached...)},
		{ID: "inside", Title: "Inside", Uncounted: true, Items: []page.Item{{Note: &page.Note{Text: "Station opens it for debugging only; your tooling owns what runs inside."}}}},
	}})
}
