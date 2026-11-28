package machines

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

const (
	KeyCreate  = "n"
	KeyDelete  = "d"
	KeyStart   = "s"
	KeyStop    = "S"
	KeyRestart = "ctrl+r"
	KeyEnter   = "e"
	KeyCopy    = "c"
	// value: SSH's o: a new terminal runs the command.
	KeyOpenTerminal = "o"
)

func Columns() []page.Column {
	// invariant: the pipe keeps SPEC §10.8's eight fields (kind, host, lifetime,
	// and a fix riding in the resources field); the eye reads one class column (a
	// host's `host`, a VM's lifetime: the tree says the rest), the host's role,
	// the resources a short line drops first, and the detail that says why a row
	// is not ● (§13.5).
	return []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100},
		{Key: "kind", Kind: page.KindText, Show: page.PipeOnly},
		{Key: "mac", Kind: page.KindText, Show: page.PipeOnly},
		{Key: "lifetime", Kind: page.KindText, Show: page.PipeOnly},
		{Key: "class", Kind: page.KindText, Keep: 70, Show: page.EyeOnly},
		{Key: "role", Kind: page.KindText, Keep: 30, Show: page.EyeOnly},
		{Key: "state", Kind: page.KindText, Keep: 90},
		{Key: "ip", Kind: page.KindIdentity, Keep: 50},
		{Key: "record", Kind: page.KindList, Show: page.PipeOnly},
		{Key: "resources", Kind: page.KindList, Keep: 10, Whole: true, Show: page.EyeOnly},
		{Key: "detail", Kind: page.KindList, Keep: 5, Show: page.EyeOnly},
	}
}

func Section(f model.AllMachines, links func(mac, vm string) []page.Link) page.Section {
	// invariant: one column schema for host and VM rows, and no piped row ends on
	// an empty cell nor grows a hint field, so every record has the same eight
	// fields: a fix rides in the resources cell.
	thisVersion := ""
	for _, h := range f.Macs {
		if h.ThisMac {
			thisVersion = h.StationVersion
		}
	}
	items := make([]page.Item, 0, len(f.Macs))
	for _, h := range f.Macs {
		g := page.Group{Head: macHeadRow(h, thisVersion), Fold: page.Open}
		for _, vm := range h.VMs {
			row := vmRow(vm)
			var under []page.Item
			if links != nil {
				for _, l := range links(h.Name, vm.Name) {
					under = append(under, page.Item{Link: &l})
				}
			}
			if len(under) == 0 {
				g.Items = append(g.Items, page.Item{Row: &row})
				continue
			}
			g.Items = append(g.Items, page.Item{Group: &page.Group{Head: row, Items: under, Fold: page.Open}})
		}
		if len(h.VMs) == 0 && h.Reachable {
			hint := page.Cmd("station machines create")
			if !h.ThisMac {
				hint = page.Cmd("station machines create --on " + h.Name)
			}
			g.Items = append(g.Items, page.Item{Empty: &page.Empty{Noun: "VMs", Hint: hint}})
		}
		items = append(items, page.Item{Group: &g})
	}
	if len(items) == 0 {
		items = append(items, page.Item{Empty: &page.Empty{Noun: "Macs", Hint: page.Cmd("station machines add <name>")}})
	}
	return page.Section{ID: "all-machines", Title: "All machines", Columns: Columns(), Items: items}
}

func Actions() []page.Action {
	return []page.Action{
		{Key: KeyCreate, Label: "create", On: page.OnPage, Priority: 4, Mutates: true},
		{Key: KeyDelete, Label: "delete", On: page.OnRow, Priority: 1, Mutates: true, Confirm: &page.Confirm{Yes: "delete", Danger: true}},
		{Key: KeyStart, Label: "start", On: page.OnRow, Priority: 5, Mutates: true},
		{Key: KeyStop, Label: "stop", On: page.OnRow, Priority: 6, Mutates: true},
		{Key: KeyRestart, Label: "restart", On: page.OnRow, Priority: 2, Mutates: true, Confirm: &page.Confirm{Yes: "restart", Danger: true}},
		{Key: KeyEnter, Label: "enter", On: page.OnRow, Priority: 3, Mutates: true},
	}
}

func behind(h model.MacView, thisVersion string) bool {
	_, differs := model.Behind(h, thisVersion)
	return differs
}

func macHeadRow(h model.MacView, thisVersion string) page.Row {
	row := page.Row{ID: h.Name, Label: h.Name, Mark: page.OK}
	state, resources, detail := page.Text("online"), vmCount(len(h.VMs)), page.None()
	switch {
	case !h.Reachable:
		row.Mark, row.Reason, state, resources = page.Attention, page.Unreachable, page.Text("unreachable"), page.Text("-")
	case behind(h, thisVersion):
		row.Mark, row.Reason = page.Attention, page.Mismatch
		detail = page.Text("station " + h.StationVersion + ", This Mac " + thisVersion)
	case h.Engine.Known() && !h.Engine.Running():
		row.Mark, row.Reason = page.Attention, page.Stopped
		word, fix := "engine stopped", "station engine start"
		switch {
		case !h.ThisMac:
			ssh, err := catalog.SSHCommand("", h.Name)
			fix = ssh + " station engine start"
			if err != nil {
				fix = ""
			}
		case h.Engine.State == model.EngineNotInstalled:
			row.Reason, word, fix = page.Missing, "engine not installed", "station engine start"
		}
		detail = page.List(page.Text(word))
		if fix != "" {
			detail = page.List(page.Text(word), page.Command(fix))
		}
	}
	role := ""
	if h.ThisMac {
		role = "This Mac"
	}
	row.Cells = []page.Value{page.Text("mac"), page.Text(""), page.Text(""), page.Text("Mac"), page.Text(role), state, orDash(h.Node.IP), record(resources, detail), resources, detail}
	return row
}

func vmRow(vm model.VM) page.Row {
	row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.OK}
	resources, detail := Resources(vm), page.None()
	switch vm.State {
	case model.Running:
	case model.Stopped:
		row.Mark = page.Absent
		if vm.Detail != "" {
			detail = page.Text(vm.Detail)
		}
	case model.Suspended:
		row.Mark, row.Reason = page.Attention, page.Stopped
	case model.Stale:
		row.Mark, row.Reason = page.Attention, page.Outlived
		detail = page.List(page.Text("ended unexpectedly"), page.Text("start or delete it"))
	case model.Broken:
		row.Mark, row.Reason, detail = page.Broken, page.Failing, page.Text(vm.Detail)
	case model.UnknownState:
		row.Mark, row.Reason = page.NotKnown, page.Unreachable
	case model.Creating:
		row.Mark = page.Unjudged
		if vm.Birth != nil {
			detail = page.Text(vm.Birth.Doing())
		}
	}
	if vm.Birth != nil && vm.Birth.Failed {
		detail = page.Text(vm.Birth.Sentence)
	}
	if vm.Detail == model.HalfMade {
		detail = page.Text(vm.Detail)
	}
	row.Cells = []page.Value{page.Text(string(vm.Lifetime.Kind())), page.Text(vm.Mac), page.Text(string(vm.Lifetime)), page.Text(string(vm.Lifetime)), page.Text(""), page.Text(string(vm.State)), orDash(vm.IP), record(resources, detail), resources, detail}
	return row
}

func Resources(vm model.VM) page.Value {
	r := vm.Resources
	return page.List(plural(r.CPUs, "cpu"), Memory(r.MemoryBytes), page.Size(r.DiskBytes))
}

func Memory(n int64) page.Value {
	// invariant: memory reads in binary gigabytes, as macOS and tart count RAM, so the
	// 8 GB a VM was given prints as 8 GB.
	const gib = 1 << 30
	if n%gib == 0 {
		return page.Text(strconv.FormatInt(n/gib, 10) + " GB")
	}
	return page.Text(strconv.FormatFloat(float64(n)/gib, 'f', 1, 64) + " GB")
}

func plural(n int, noun string) page.Value {
	if n == 1 {
		return page.Count(1, noun)
	}
	return page.Count(int64(n), noun+"s")
}

func vmCount(n int) page.Value { return plural(n, "VM") }

// value: the pipe's last field: the fix when the row has one, else its resources.
func record(resources, detail page.Value) page.Value {
	if !detail.IsNone() {
		return detail
	}
	return resources
}

func orDash(s string) page.Value {
	if s == "" {
		return page.Text("-")
	}
	return page.Identity(s)
}

// value: a VM's delete warnings, one row each under its own.
func warned(name string, warnings []string) []page.Row {
	rows := make([]page.Row, 0, len(warnings))
	for i, w := range warnings {
		row := pages.Done(name+"/"+strconv.Itoa(i), catalog.StateDegraded, w)
		row.Label = name
		rows = append(rows, row)
	}
	return rows
}

func Change(verb, name, change string, thisMac page.Context) page.Page {
	return pages.Outcome("machines "+verb, thisMac, nil, []page.Row{pages.Done(name, catalog.StateOK, change)}, "machines")
}

func ServerStopped(name, start string, thisMac page.Context) page.Page {
	// value: a server stopped by hand stays stopped, at its Mac's boot too,
	// until start sets its wanted state back to Running.
	row := pages.Done(name, catalog.StateOK, "stopped")
	row.Detail = page.List(page.Text("stopped"), page.Text("wanted state: Stopped"))
	p := pages.Outcome("machines stop", thisMac, nil, []page.Row{row}, "machines")
	p.Hints = []page.Hint{*page.Cmd(start)}
	return p
}

func ServerStarted(name string, thisMac page.Context) page.Page {
	row := pages.Done(name, catalog.StateOK, "started")
	row.Detail = page.List(page.Text("started"), page.Text("wanted state: Running"))
	return pages.Outcome("machines start", thisMac, nil, []page.Row{row}, "machines")
}

func Added(a model.Added, thisMac page.Context) page.Page {
	// value: the Mac added, as ssh and its station answered; a station of
	// another version is one more row, with what brings them to one.
	rows := []page.Row{done(a.Label, catalog.StateOK, "added", strings.ToLower(model.StepSSH), "station "+a.Version)}
	if a.Update != "" {
		rows = append(rows, done(a.Label, catalog.StateDegraded, "runs "+a.Version, a.Update))
	}
	return pages.Outcome("machines add", thisMac, nil, rows, "machines")
}

func Forgot(name string, thisMac page.Context) page.Page {
	return pages.Outcome("machines forget", thisMac, nil, []page.Row{done(name, catalog.StateOK, "forgotten", "nothing changed on it")}, "machines")
}

func done(id string, state catalog.State, said ...string) page.Row {
	r := pages.Done(id, state, "")
	r.Detail = page.Texts(said...)
	return r
}

func Unlocked(u model.Unlocked, thisMac page.Context) page.Page {
	row := done(u.Name, catalog.StateOK, "unlocked", "on "+catalog.TailnetProduct+" again")
	switch {
	case u.Already:
		row = done(u.Name, catalog.StateOK, "already answers: its disk is unlocked")
	case !u.Answers:
		row = done(u.Name, catalog.StateDegraded, "unlocked", "not on "+catalog.TailnetProduct+" yet")
	}
	return pages.Outcome("machines unlock", thisMac, nil, []page.Row{row}, "machines")
}

func Woken(w model.Woken, thisMac page.Context) page.Page {
	row := done(w.Name, catalog.StateOK, "wake sent", "it answers in a minute or so")
	switch {
	case w.Already:
		row = done(w.Name, catalog.StateOK, "already awake: it answers")
	case w.Tell:
		row = done(w.Name, catalog.StateOK, "wake sent", "you are told once it answers")
	}
	return pages.Outcome("machines wake", thisMac, nil, []page.Row{row}, "machines")
}

func Reminded(r model.Reminded, thisMac page.Context) page.Page {
	row := done(r.Name, catalog.StateOK, "you are told once it answers")
	if r.Already {
		row = done(r.Name, catalog.StateOK, "already awake: it answers")
	}
	return pages.Outcome("machines remind", thisMac, nil, []page.Row{row}, "machines")
}

func Back(b model.Back, thisMac page.Context) page.Page {
	// value: the Mac back, and each VM it paused as it is now.
	said := make([]string, 0, 1+len(b.VMs))
	said = append(said, "back")
	for _, vm := range b.VMs {
		said = append(said, vm.Name+" "+strings.ToLower(vm.State.Word()))
	}
	return pages.Outcome("machines restart "+b.Name, thisMac, nil, []page.Row{done(b.Name, catalog.StateOK, said...)}, "machines")
}

func Already(verb, name string, running bool, thisMac page.Context) page.Page {
	// value: a start or stop that found the VM already so: its mark is the state
	// it is in, running or stopped, and nothing changed.
	state, said := catalog.StateAbsent, "is already stopped"
	if running {
		state, said = catalog.StateOK, "is already running"
	}
	return pages.Outcome("machines "+verb, thisMac, nil, []page.Row{pages.Done(name, state, said)}, "machines")
}

type CreatedJSON struct {
	Schema      string            `json:"schema"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Mac         string            `json:"mac"`
	Lifetime    string            `json:"lifetime"`
	Check       string            `json:"check,omitempty"`
	Image       string            `json:"image"`
	Version     string            `json:"version"`
	CPUs        int               `json:"cpus"`
	MemoryBytes int64             `json:"memory_bytes"`
	DiskBytes   int64             `json:"disk_bytes"`
	IP          string            `json:"ip"`
	Network     string            `json:"network"`
	Health      CreatedHealth     `json:"health"`
	Ports       []CreatedPort     `json:"ports"`
	Env         map[string]string `json:"env"`
	// value: the Machinefile it was made from, and its first start's boot
	// phase; absent for a workspace made from an image alone.
	Template string       `json:"template,omitempty"`
	Boot     *CreatedBoot `json:"boot,omitempty"`
}

type CreatedBoot struct {
	// value: why the boot phase's apply did not end well; empty when it did.
	Failed string `json:"failed,omitempty"`
	// value: each package the apply of its whole Machinefile installed or
	// set, and each it left unfinished, with why.
	Applied    []CreatedPackage `json:"applied"`
	Unfinished []CreatedPackage `json:"unfinished"`
	// value: why that apply answered for no package; absent when it did.
	Unapplied string `json:"unapplied,omitempty"`
	// value: why that apply did not write the machine's `spwn` skill;
	// absent when it did.
	SkillFailed string          `json:"skill_failed,omitempty"`
	Grants      []CreatedGrant  `json:"grants"`
	SignIns     []CreatedSignIn `json:"signins"`
	Drift       []string        `json:"drift"`
}

type CreatedPackage struct {
	Name    string `json:"name"`
	Setting bool   `json:"setting,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Log     string `json:"log,omitempty"`
}

type CreatedGrant struct {
	Program string `json:"program"`
	// value: why it was not granted, its refusal's code and subject; absent
	// when it was granted.
	Code    string `json:"code,omitempty"`
	Subject string `json:"subject,omitempty"`
}

type CreatedSignIn struct {
	Program string `json:"program"`
	Command string `json:"command"`
}

type CreatedHealth struct {
	Running bool `json:"running"`
	Address bool `json:"address"`
	SSH     bool `json:"ssh"`
}

type CreatedPort struct {
	GuestPort int    `json:"guest_port"`
	Reserved  int    `json:"reserved"`
	Profile   string `json:"profile"`
	URL       string `json:"url"`
}

func CreatedContract(schema string, c model.Created) CreatedJSON {
	// invariant: a VM is created once it runs, has its address and answers
	// ssh: the three steps its page passes.
	vm, r := c.VM, c.VM.Resources
	body := CreatedJSON{
		Schema: schema, Health: CreatedHealth{Running: true, Address: vm.IP != "", SSH: true}, Name: vm.Name, Kind: string(vm.Lifetime.Kind()), Mac: vm.Mac, Lifetime: string(vm.Lifetime), Check: vm.Check, Image: vm.Image,
		Version: vm.Version, CPUs: r.CPUs, MemoryBytes: r.MemoryBytes, DiskBytes: r.DiskBytes, IP: vm.IP,
		Network: vm.Network, Ports: []CreatedPort{}, Env: c.Env,
	}
	if body.Env == nil {
		body.Env = map[string]string{}
	}
	for _, s := range c.Ports {
		body.Ports = append(body.Ports, CreatedPort{GuestPort: s.GuestPort, Reserved: s.MacPort, Profile: s.Profile, URL: s.URL})
	}
	if b := c.Boot; b != nil {
		body.Template = b.Template
		body.Boot = &CreatedBoot{Failed: b.Failed, Applied: createdPackages(b.Applied), Unfinished: createdPackages(b.Unfinished), Unapplied: b.Unapplied, SkillFailed: b.SkillFailed, Grants: []CreatedGrant{}, SignIns: []CreatedSignIn{}, Drift: append([]string{}, b.Drift...)}
		for _, g := range b.Grants {
			body.Boot.Grants = append(body.Boot.Grants, CreatedGrant{Program: g.Program, Code: string(g.Code), Subject: g.Subject})
		}
		for _, s := range b.SignIns {
			body.Boot.SignIns = append(body.Boot.SignIns, CreatedSignIn{Program: s.Program, Command: s.Command})
		}
	}
	return body
}

func createdPackages(of []model.FirstStartApp) []CreatedPackage {
	out := make([]CreatedPackage, 0, len(of))
	for _, a := range of {
		out = append(out, CreatedPackage{Name: a.App, Setting: a.Setting, Reason: a.Reason, Log: a.Log})
	}
	return out
}

func Created(c model.Created, thisMac page.Context, refusal func(model.FirstStartGrant) string) page.Page {
	vm, r := c.VM, c.VM.Resources
	rows := []page.Row{passed(model.StepRunning, ""), passed(model.StepAddress, vm.IP), passed(model.StepSSH, "")}
	if vm.Lifetime == model.AlwaysOn {
		return serverCreated(c, rows, thisMac)
	}
	off := ""
	for _, s := range c.Ports {
		row := pages.Done(s.Profile+"/"+strconv.Itoa(s.GuestPort), catalog.StateOK, "")
		row.Label = s.Profile
		row.Cells = []page.Value{page.Port(int64(s.GuestPort)), page.URL(s.URL)}
		rows = append(rows, row)
		if s.Off != "" {
			off = s.Off
		}
	}
	if off != "" {
		row := page.Row{ID: "ports", Label: "ports", Mark: page.Attention, Reason: page.Degraded, Cells: []page.Value{page.Text(off)}}
		rows = append(rows, row)
	}
	rows = append(rows, bootRows(vm.Name, c.Boot, refusal)...)
	head := pages.Done(vm.Name, catalog.StateOK, "")
	head.Cells = []page.Value{page.Text("Created"), page.List(page.Text(titled(string(vm.Lifetime))), page.Text(strconv.Itoa(r.CPUs)+" CPU"), Memory(r.MemoryBytes), DiskSize(r.DiskBytes))}
	rows = append(rows, head)
	columns := []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "what", Kind: page.KindText, Keep: 60},
		{Key: "detail", Kind: page.KindList, Keep: 20},
	}
	p := pages.Outcome("machines create", thisMac, columns, rows, "VMs")
	if ssh, err := catalog.SSHCommand("", vm.Name); err == nil {
		p.Hints = []page.Hint{*page.Cmd(ssh)}
	}
	if b := c.Boot; b != nil && len(b.Drift) > 0 {
		p.Hints = append(p.Hints, *page.Cmd("station images build " + b.Template))
	}
	return p
}

// invariant: a first start's boot phase says what it did of its
// Machinefile and what it left: how many apps its apply installed and
// settings it set, each package it left with why, then how many are left
// and the apply that finishes them, then the `spwn` skill when its apply
// could not write it, counted among none of them; a boot phase, a
// package, a skill or a grant that failed is attention, never a failed
// create, a refused grant in the
// words of its refusal, and the image's drift names each item with the
// build that carries it.
func bootRows(vm string, b *model.FirstStart, refusal func(model.FirstStartGrant) string) []page.Row {
	if b == nil {
		return nil
	}
	attention := func(id, label, detail string) page.Row {
		return page.Row{ID: id, Label: label, Mark: page.Attention, Reason: page.Degraded, Cells: []page.Value{page.Text(detail), page.None()}}
	}
	var rows []page.Row
	if b.Failed != "" {
		rows = append(rows, attention("boot", model.StepBoot, b.Failed))
	} else {
		row := passed(model.StepBoot, b.Template)
		if said := appliedWords(b.Applied); len(said) > 0 {
			row.Cells[1] = page.List(said...)
		}
		rows = append(rows, row)
	}
	for _, u := range b.Unfinished {
		verb := "not installed: "
		if u.Setting {
			verb = "not set: "
		}
		rows = append(rows, attention("unfinished/"+u.App, u.App, verb+u.Reason))
	}
	switch finish := "station machinefile apply --on " + vm; {
	case b.Unapplied != "":
		rows = append(rows, attention("unapplied", b.Template, "not applied: "+b.Unapplied), attention("finish", "Apps to finish", finish))
	case len(b.Unfinished) > 0:
		rows = append(rows, attention("finish", packagesLeft(b.Unfinished)+" to finish", finish))
	}
	if b.SkillFailed != "" {
		rows = append(rows, attention("skill/spwn", "spwn", "skill not written: "+b.SkillFailed))
	}
	for _, g := range b.Grants {
		if g.Code != "" {
			rows = append(rows, attention("grant/"+g.Program, g.Program, "not granted: "+refusal(g)))
			continue
		}
		row := passed(g.Program, "granted")
		row.ID = "grant/" + g.Program
		rows = append(rows, row)
	}
	for _, s := range b.SignIns {
		rows = append(rows, attention("signin/"+s.Program, s.Program, "not signed in: "+s.Command))
	}
	if len(b.Drift) > 0 {
		row := attention("drift", "Drift", "its image is behind its Machinefile")
		items := make([]page.Value, 0, len(b.Drift))
		for _, d := range b.Drift {
			items = append(items, page.Text(d))
		}
		row.Cells[1] = page.List(items...)
		rows = append(rows, row)
	}
	return rows
}

// value: what a create's apply carried out, counted: the apps installed,
// then the settings set; none when it carried out nothing.
func appliedWords(applied []model.FirstStartApp) []page.Value {
	apps, settings := 0, 0
	for _, a := range applied {
		if a.Setting {
			settings++
		} else {
			apps++
		}
	}
	var out []page.Value
	if apps > 0 {
		out = append(out, page.Text(counted(apps, "app")+" installed"))
	}
	if settings > 0 {
		out = append(out, page.Text(counted(settings, "setting")+" set"))
	}
	return out
}

// value: how many packages a create's apply left, named apps when every
// one is an app.
func packagesLeft(left []model.FirstStartApp) string {
	noun := "app"
	for _, a := range left {
		if a.Setting {
			noun = "package"
		}
	}
	return counted(len(left), noun)
}

func passed(label, detail string) page.Row {
	row := pages.Done(label, catalog.StateOK, "")
	row.Cells = []page.Value{page.None(), page.None()}
	if detail != "" {
		row.Cells[0] = page.Text(detail)
	}
	return row
}

// invariant: a server's create says what station will not do inside it; a
// check that does not answer yet waits, never fails, as what answers it
// comes with the owner's tooling (SPEC §7.1).
func serverCreated(c model.Created, rows []page.Row, thisMac page.Context) page.Page {
	vm, r := c.VM, c.VM.Resources
	if vm.Check != "" {
		rows = append(rows, page.Row{ID: "check", Label: "Check waiting", Mark: page.Attention, Reason: page.Waiting, Cells: []page.Value{page.Text("normal: what answers comes with your tooling")}})
	}
	if c.OnALaptop {
		mac := vm.Mac
		if mac == thisMac.Machine {
			mac = "This Mac"
		}
		rows = append(rows, page.Row{ID: "laptop", Label: mac + " is a laptop", Mark: page.Attention, Reason: page.Degraded, Cells: []page.Value{page.List(page.Text(vm.Name+" stops whenever it sleeps"), page.Text("see Mac readiness"))}})
	}
	head := pages.Done(vm.Name, catalog.StateOK, "")
	head.Cells = []page.Value{page.Text("is a server"), page.List(page.Text("always on"), page.Text(strconv.Itoa(r.CPUs)+" CPU"), Memory(r.MemoryBytes), DiskSize(r.DiskBytes))}
	rows = append(rows, head)
	note := page.Row{ID: "inside", Label: "Machines sets up nothing inside", Mark: page.OK, Cells: []page.Value{page.Text("no apps, settings or accounts: set it up with your tooling")}}
	rows = append(rows, note)
	columns := []page.Column{
		{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "what", Kind: page.KindText, Keep: 60},
		{Key: "detail", Kind: page.KindList, Keep: 20},
	}
	p := pages.Outcome("machines create", thisMac, columns, rows, "VMs")
	p.Hints = []page.Hint{*page.Cmd("station machines " + vm.Name)}
	if ssh, err := catalog.SSHCommand("", vm.Name); err == nil {
		p.Hints = []page.Hint{*page.Cmd(ssh), p.Hints[0]}
	}
	return p
}

func Deleted(r model.DeleteReport, thisMac page.Context) page.Page {
	rows := make([]page.Row, 0, 1+len(r.Warnings))
	rows = append(rows, pages.Done(r.Name, catalog.StateOK, "deleted"))
	rows = append(rows, warned(r.Name, r.Warnings)...)
	return pages.Outcome("machines delete "+r.Name, thisMac, nil, rows, "VMs")
}

func Restarted(r model.RestartReport, thisMac page.Context) page.Page {
	rows := make([]page.Row, 0, 4)
	if r.VM {
		rows = append(rows, passed(model.StepRunning, ""), passed(model.StepAddress, r.IP), passed(model.StepSSH, ""))
	}
	rows = append(rows, pages.Done(r.Name, catalog.StateOK, "restarted"))
	return pages.Outcome("machines restart "+r.Name, thisMac, nil, rows, "VMs")
}

type MacChoice struct {
	Name, Detail string
	ThisMac      bool
}

func CreateForm(lifetime string, macs []MacChoice) page.Form {
	var f page.Form
	if lifetime == "" {
		f.Fields = append(f.Fields, page.Field{Key: "lifetime", Label: "Will it last?", Kind: page.FieldSelect, Options: []page.Option{
			{Value: "ephemeral", Label: "Ephemeral: deleted when it powers off from inside"},
			{Value: "persistent", Label: "Persistent: kept until you delete it"},
		}})
	}
	if len(macs) >= 2 {
		field := page.Field{Key: "mac", Label: "On which Mac?", Kind: page.FieldSelect}
		for _, h := range macs {
			field.Options = append(field.Options, page.Option{Value: h.Name, Label: h.Name, Note: h.Detail})
			if h.ThisMac {
				field.Default = h.Name
			}
		}
		f.Fields = append(f.Fields, field)
	}
	return f
}

type SSHJSON struct {
	Schema  string `json:"schema"`
	Name    string `json:"name"`
	User    string `json:"user"`
	Address string `json:"address"`
	Command string `json:"command"`
}

func SSHContract(name string, t model.SSHTarget) SSHJSON {
	return SSHJSON{Schema: "station.machines.ssh/v1", Name: name, User: t.User, Address: t.Address, Command: t.Command}
}

func SSH(name string, t model.SSHTarget, thisMac page.Context) page.Page {
	row := pages.Done(name, catalog.StateOK, "")
	row.Cells = []page.Value{page.Command(t.Command)}
	p := pages.Outcome("machines ssh "+name, thisMac, []page.Column{{Key: "name", Kind: page.KindText, Keep: 100}, {Key: "command", Kind: page.KindText, Keep: 90}}, []page.Row{row}, "machines")
	p.Actions = []page.Action{
		{Key: KeyCopy, Label: "copy", On: page.OnRow, Priority: 1},
		{Key: KeyOpenTerminal, Label: "open in a new terminal", On: page.OnRow, Priority: 2},
	}
	return p
}

func Edited(e model.Edited, thisMac page.Context) page.Page {
	// value: the sizes an edit changed, and when they apply; a running VM is
	// told the restart that applies them now.
	var parts []string
	if n := e.Changed.CPUs; n == 1 {
		parts = append(parts, "1 cpu")
	} else if n != 0 {
		parts = append(parts, strconv.Itoa(n)+" cpus")
	}
	if e.Changed.MemoryBytes != 0 {
		parts = append(parts, Memory(e.Changed.MemoryBytes).Text)
	}
	if e.Changed.DiskBytes != 0 {
		parts = append(parts, strconv.FormatFloat(float64(e.Changed.DiskBytes)/1e9, 'f', 1, 64)+" GB disk")
	}
	var said string
	if len(parts) > 0 {
		said = strings.Join(parts, ", ") + ", applies when " + e.Name + " next starts"
	}
	switch check := ""; {
	case e.NoCheck:
		check = "no health check: health has three steps"
		said = strings.TrimPrefix(said+"; "+check, "; ")
	case e.Check != "":
		check = "health check " + e.Check + ", from its next health run"
		said = strings.TrimPrefix(said+"; "+check, "; ")
	}
	row := pages.Done(e.Name, catalog.StateOK, said)
	if e.Running && len(parts) > 0 {
		row.Detail = page.List(page.Text(said), page.Command("station machines restart "+e.Name))
	}
	return pages.Outcome("machines edit "+e.Name, thisMac, nil, []page.Row{row}, "machines")
}
