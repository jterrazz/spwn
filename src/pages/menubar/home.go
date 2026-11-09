package menubar

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/engine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

const (
	SectionTiles   = "tiles"
	SectionAgents  = "agents"
	SectionServers = "servers"
	SectionMacs    = "macs"

	TilePorts    = "ports"
	TileAccounts = "accounts"
	TileAwake    = "awake"

	RowMoreAgents = "more-agents"
	RowAddMac     = "add-mac"

	ActionNewVMOn = "new-vm:"

	// value: SPEC A4 Home 3: past three live agents, two stay and the
	// others fold.
	agentsFold = 3
	agentsKept = 2
)

// value: what Home's sections below the hero read: the fleet, This Mac's
// name, the upkeep ranked, the clock.
type homeReading struct {
	r      Reading
	self   string
	upkeep []Need
	now    time.Time
}

// invariant: SPEC A4 Home 2-5: under the hero, the three tiles, then the
// agents, the servers and the Macs, each one flush card; the empty fleet
// keeps only its Macs.
func homeSections(r Reading, self string, upkeep []Need) ([]page.Section, []page.Action) {
	if r.All == nil {
		return nil, nil
	}
	h := homeReading{r: r, self: self, upkeep: upkeep, now: r.now()}
	var sections []page.Section
	var actions []page.Action
	if !empty(*r.All) {
		tiles, toggle := h.tiles()
		sections = append(sections, tiles)
		actions = append(actions, toggle)
	}
	if s, ok := h.agents(); ok {
		sections = append(sections, s)
	}
	if s, ok := h.servers(); ok {
		sections = append(sections, s)
	}
	macs, more := h.macs()
	sections = append(sections, macs)
	actions = append(actions, more...)
	for _, m := range r.All.Macs {
		for _, vm := range m.VMs {
			if alive(vm) && !vm.Finished() {
				actions = append(actions, attach(m, vm))
			}
		}
	}
	for _, m := range r.All.Macs {
		if slices.ContainsFunc(m.VMs, model.VM.Finished) {
			actions = append(actions, h.cleanup(m))
		}
	}
	return sections, actions
}

func alive(vm model.VM) bool { return vm.State == model.Running || vm.State == model.Creating }

// value: the three tiles, Ports, Accounts and Awake: each its caption and
// glyph, one number and its word, the page it opens; Awake's glyph its
// switch.
func (h homeReading) tiles() (page.Section, page.Action) {
	toggle := h.awakeToggle()
	items := []page.Item{{Tile: h.portsTile()}, {Tile: h.accountsTile()}, {Tile: h.awakeTile(toggle)}}
	return page.Section{ID: SectionTiles, Title: "Tiles", Role: page.Summary, Card: page.CardTiles, Items: items}, toggle
}

// value: the Ports tile: how many ports are shared (each shared port,
// each declared public, each a server's own device carries); red what is
// exposed, or what a server down leaves unreachable; what a server
// restarting is starting.
func (h homeReading) portsTile() *page.Tile {
	t := &page.Tile{Caption: "Ports", Glyph: symbol("network", page.SignalQuiet), Nav: PortsID, Tip: page.Texts("Ports"), Key: keyPorts}
	if h.r.Ports == nil {
		t.Number = page.Pending()
		return t
	}
	shared, exposed, unreachable, starting := 0, 0, 0, 0
	for _, m := range h.r.Ports.Macs {
		shared += len(m.Shared)
		for _, p := range m.Ports {
			switch {
			case p.Red():
				exposed++
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
	t.Number, t.Word = page.Count(int64(shared), ""), "shared"
	switch {
	case exposed > 0:
		t.Number, t.Word, t.Signal = page.Count(int64(exposed), ""), "exposed", page.SignalFailing
	case unreachable > 0:
		t.Number, t.Word, t.Signal = page.Count(int64(unreachable), ""), "unreachable", page.SignalFailing
	case starting > 0:
		t.Number, t.Word = page.Count(int64(starting), ""), "starting"
	}
	t.Tip = page.Texts(strconv.FormatInt(t.Number.N, 10) + " " + t.Word)
	return t
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

// value: the Accounts tile: what is signed out on any Mac, as its upkeep
// says it, else how many accounts are shared with VMs.
func (h homeReading) accountsTile() *page.Tile {
	t := &page.Tile{Caption: "Accounts", Glyph: symbol("key", page.SignalQuiet), Nav: AccountsID, Tip: page.Texts("Accounts", "All machines"), Key: keyAccounts}
	out := 0
	for _, n := range h.upkeep {
		if n.Finding.Section == section.Accounts {
			out++
		}
	}
	if out > 0 {
		t.Number, t.Word, t.Signal = page.Count(int64(out), ""), "signed out", page.SignalWaits
		return t
	}
	if h.r.Accounts == nil {
		t.Number = page.Pending()
		return t
	}
	shared := 0
	for _, m := range h.r.Accounts.Macs {
		shared += len(m.Shares)
	}
	t.Number, t.Word = page.Count(int64(shared), ""), "shared"
	return t
}

// value: the Awake tile: the time This Mac stays awake, or Off; its glyph
// the switch, the sun while it holds, the moon while it may sleep.
func (h homeReading) awakeTile(toggle page.Action) *page.Tile {
	t := &page.Tile{Caption: "Awake", Toggle: toggle.ID, Nav: AwakeID, Glyph: symbol("moon", page.SignalQuiet), Number: page.Text("Off"), Tip: page.Texts("Awake", durations), Key: keyAwake}
	r := h.r.Awake
	if r != nil && r.Session != nil {
		t.Tip = page.Texts("Awake "+awakeWords(*r), durations)
	}
	switch {
	case r == nil:
		t.Number = page.Pending()
	case r.Session != nil:
		t.Glyph = symbol("sun.max", page.SignalWaits)
		t.Number, t.Word = page.Text(hoursLeft(r.Session.Until.Sub(h.now))), "left"
	case r.NeverSleeps:
		t.Glyph = symbol("sun.max", page.SignalWaits)
		t.Number = page.Text("Always")
	}
	return t
}

// value: a time left as the tile says it, in whole hours from one hour
// on: "4 hr", "35 min".
func hoursLeft(d time.Duration) string {
	if d >= time.Hour {
		return strconv.Itoa(int(d/time.Hour)) + " hr"
	}
	return minutes(d)
}

// invariant: the Awake tile's switch is one click: it lets This Mac sleep
// while a keep-awake holds, else keeps it awake for the default term; the
// other terms are its menu's.
func (h homeReading) awakeToggle() page.Action {
	a := awakeAction(h.r.Awake)
	a.Label, a.Rows = "Keep "+page.ThisMac+" awake", []string{TileAwake}
	if a.ID == ActionAwakeOff {
		a.Picked = true
		return a
	}
	a.Form = nil
	a.Run.In = map[string]string{"For": strconv.FormatInt(int64(4*time.Hour), 10)}
	return a
}

// value: an agent of the fleet as Home lists it: the VM it runs on, and
// that VM's Mac.
type homeAgent struct {
	vm  model.VM
	mac model.MacView
}

func (a homeAgent) state() string {
	switch {
	case a.vm.Finished():
		return model.AgentFinished
	case a.vm.State == model.Creating:
		return string(agents.Starting)
	}
	return a.vm.Agent.State
}

func agentRank(a homeAgent) int {
	switch a.state() {
	case string(agents.Waiting):
		return 0
	case string(agents.Starting):
		return 2
	case model.AgentFinished:
		return 3
	}
	return 1
}

// invariant: SPEC A4 Home 3: one row per agent, never per VM, ordered
// waiting, working, starting, then finished; past three live agents the
// first two stay, then the finished ones, the other live ones folded
// behind "N more" which names them.
func (h homeReading) agents() (page.Section, bool) {
	var all []homeAgent
	for _, m := range h.r.All.Macs {
		for _, vm := range m.VMs {
			if vm.Agent != nil {
				all = append(all, homeAgent{vm: vm, mac: m})
			}
		}
	}
	if len(all) == 0 {
		return page.Section{}, false
	}
	slices.SortStableFunc(all, func(a, b homeAgent) int { return agentRank(a) - agentRank(b) })
	var livers, done []homeAgent
	working := 0
	for _, a := range all {
		switch a.state() {
		case model.AgentFinished:
			done = append(done, a)
		case string(agents.Working), string(agents.Running):
			working++
			livers = append(livers, a)
		default:
			livers = append(livers, a)
		}
	}
	s := page.Section{ID: SectionAgents, Title: "Agents", Card: page.CardFlush, Columns: vmColumns(), Uncounted: true}
	if working > 0 {
		s.Caption = []page.Value{page.Text(strconv.Itoa(working) + " working")}
	}
	shown, folded := livers, []homeAgent(nil)
	if len(livers) > agentsFold {
		shown, folded = livers[:agentsKept], livers[agentsKept:]
	}
	for _, a := range append(append(slices.Clone(shown), done...), folded...) {
		row := h.agentRow(a)
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if len(folded) > 0 {
		names := make([]string, 0, len(folded))
		for _, a := range folded {
			names = append(names, a.vm.Name)
		}
		s.Top = len(shown) + len(done)
		s.Rest = &page.Item{Row: &page.Row{ID: RowMoreAgents, Label: strconv.Itoa(len(folded)) + " more", Detail: page.Text("Show less"), Cells: []page.Value{page.Text(strings.Join(names, ", "))}}}
	}
	return s, true
}

func (h homeReading) agentRow(a homeAgent) page.Row {
	vm := a.vm
	row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.OK, Mac: model.Label(*h.r.All, a.mac), Nav: machinePage(vm.Name)}
	if alive(vm) && !vm.Finished() {
		row.Quick = ActionAttach + vm.Name
	}
	switch a.state() {
	case model.AgentFinished:
		row.Mac, row.Detail = "", page.Text(cmpWord(vm.Agent.Reason, "done"))
		row.Glyph = page.Glyph{Kind: "symbol:checkmark", Signal: page.SignalDone}
		row.Signal, row.Verb, row.Nav = page.SignalDone, ActionCleanup+":"+a.mac.Name, ""
	case string(agents.Waiting):
		row.Glyph = page.Glyph{Kind: "agent", Signal: page.SignalWaits}
		row.Detail = page.Text(cmpWord(vm.Agent.Task, "waits for you"))
	case string(agents.Starting):
		progress := 0
		if vm.Birth != nil {
			progress = vm.Birth.Progress
		}
		row.Glyph = page.Glyph{Dot: page.DotRing, Progress: progress, Signal: page.SignalActive}
		row.Detail = page.Texts("starting", cmpWord(vm.Agent.Task, "a new task"))
	default:
		row.Glyph = page.Glyph{Kind: "agent", Signal: page.SignalActive}
		row.Detail = page.Text(cmpWord(vm.Agent.Task, "working"))
		if len(vm.Agent.Step) > 0 {
			row.Detail = page.Texts(vm.Agent.Step...)
		}
		row.Trail, row.Cells = "cpu", []page.Value{h.cpu(vm)}
	}
	if h.stale(a.mac) {
		row.Signal = page.SignalStale
	}
	return row
}

// value: a VM's CPU as a share of its Mac, This Mac's from its last
// reading, every one read live under its key.
func (h homeReading) cpu(vm model.VM) page.Value {
	v := page.None()
	if n := len(h.r.Samples); n > 0 {
		for _, u := range h.r.Samples[n-1].VMs {
			if u.VM == vm.Name {
				v = page.Percent(int64(u.CPU * 10))
			}
		}
	}
	return live(v, engine.LiveKey(engine.LiveCPUMac, vm.Name))
}

func (h homeReading) stale(m model.MacView) bool {
	return !m.Asleep && model.Unheard(m, h.now)
}

// value: Clean up on a finished agent's row: the finished throwaways of
// its Mac, asked as that Mac's page asks it.
func (h homeReading) cleanup(m model.MacView) page.Action {
	label := model.Label(*h.r.All, m)
	c := &page.Confirm{Question: "Delete the finished throwaways on " + label + "?", Lines: []page.Value{page.Text("Each goes with its disk, its ports and the accounts shared with it.")}, Yes: "Delete"}
	return page.Action{ID: ActionCleanup + ":" + m.Name, Label: "Clean up", On: page.OnRow, Priority: 3, Mutates: true, Confirm: c,
		Run: &page.Run{Call: CallCleanup, In: answered(c, map[string]string{}), Target: target(m)}}
}

// invariant: SPEC A4 Home 4: one row per server: healthy with its time up
// and its ports, red while down, its ring while it restarts, hollow while
// its Mac is checked; no uptime strip.
func (h homeReading) servers() (page.Section, bool) {
	s := page.Section{ID: SectionServers, Title: "Servers", Card: page.CardFlush, Columns: columns(), Uncounted: true}
	for _, m := range h.r.All.Macs {
		for _, vm := range m.VMs {
			if vm.Lifetime != model.AlwaysOn {
				continue
			}
			row := h.serverRow(m, vm)
			s.Items = append(s.Items, page.Item{Row: &row})
		}
	}
	return s, len(s.Items) > 0
}

func (h homeReading) serverRow(m model.MacView, vm model.VM) page.Row {
	row := page.Row{ID: vm.Name, Label: vm.Name, Mark: page.OK, Nav: machinePage(vm.Name)}
	switch {
	case h.stale(m):
		row.Glyph, row.Signal = page.Glyph{Dot: page.DotRing, Signal: page.SignalStale}, page.SignalStale
		row.Detail = page.Texts("seen "+ago(h.now.Sub(m.Answered))+" ago", "checking")
	case vm.Restart != nil:
		row.Glyph = page.Glyph{Dot: page.DotRing, Progress: (vm.Restart.Step - 1) * 1000 / model.RestartSteps, Signal: page.SignalActive}
		row.Detail = page.Texts("restarting", vm.Restart.Counted())
	case model.Down(vm):
		row.Mark, row.Reason = page.Broken, page.Down
		row.Glyph = page.Glyph{Dot: page.DotFill, Signal: page.SignalFailing}
		down := page.Text("down")
		if since := vm.CheckStatus.Since; !since.IsZero() {
			down = page.Text("down since " + (needSaid{r: NeedsReading{Zone: h.r.Zone}}).at(since))
		}
		down.Signal = page.SignalFailing
		row.Detail = page.List(down)
	case vm.State != model.Running:
		row.Mark, row.Reason = page.Absent, page.Stopped
		row.Glyph = page.Glyph{Dot: page.DotSquare, Signal: page.SignalQuiet}
		row.Detail = page.Texts("stopped")
	default:
		row.Glyph = page.Glyph{Dot: page.DotFill, Signal: page.SignalWell}
		parts := []string{health(vm)}
		if !vm.BootAt.IsZero() && h.now.After(vm.BootAt) {
			parts = append(parts, "up "+days(h.now.Sub(vm.BootAt)))
		}
		if n := h.ports(vm.Name); n > 0 {
			parts = append(parts, counted(n, "port", "ports"))
		}
		row.Detail = page.Texts(parts...)
	}
	return row
}

func (h homeReading) ports(vm string) int {
	if h.r.Ports == nil {
		return 0
	}
	n := 0
	for _, m := range h.r.Ports.Macs {
		for _, p := range m.Ports {
			if p.Machine.Name == vm {
				n++
			}
		}
	}
	return n
}

// invariant: SPEC A4 Home 5: one row per Mac: its CPU ring, what runs and
// waits there and its upkeep (the upkeep dropped first), its memory free
// for a new VM, New VM in its quick slot; an asleep Mac dimmed; a fleet
// of one Mac adds the row that adds another.
func (h homeReading) macs() (page.Section, []page.Action) {
	s := page.Section{ID: SectionMacs, Title: "Macs", Card: page.CardFlush, Columns: columns(), Uncounted: true}
	var actions []page.Action
	for _, m := range h.r.All.Macs {
		row := h.macRow(m)
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	if len(h.r.All.Macs) == 1 {
		s.Items = append(s.Items, page.Item{Row: &page.Row{ID: RowAddMac, Label: "Add a Mac to run VMs on it", Mark: page.OK, Glyph: symbol("plus", page.SignalActive), Verb: ActionAddMac}})
		actions = append(actions, page.Action{ID: ActionAddMac, Label: "Add a Mac", On: page.OnRow, Rows: []string{RowAddMac}, Priority: 4, Sheet: page.SheetHint + RowAddMac})
	}
	return s, actions
}

func (h homeReading) macRow(m model.MacView) page.Row {
	label := model.Label(*h.r.All, m)
	glyph := "desktop"
	if m.Laptop {
		glyph = "laptop"
	}
	row := page.Row{ID: m.Name, Label: label, Mark: page.OK, Nav: machinePage(m.Name), Trail: "value",
		Glyph: page.Glyph{Kind: glyph, Ring: engine.LiveKey(engine.LiveCPU, m.Name)}}
	switch {
	case m.Asleep:
		row.Glyph.Signal, row.Signal = page.SignalAsleep, page.SignalAsleep
		row.Detail = page.Texts("asleep", "seen "+minutes(h.now.Sub(m.Answered))+" ago")
		row.Trail = ""
		return row
	case h.stale(m):
		row.Signal = page.SignalStale
		row.Detail = page.Texts("seen "+ago(h.now.Sub(m.Answered))+" ago", "checking")
	default:
		row.Detail = h.macFacts(m)
	}
	row.Quick = ActionNewVMOn + m.Name
	if c, ok := model.CapacityOf(m); ok {
		row.Cells = []page.Value{page.Text(strconv.FormatInt(c.Free>>30, 10) + " GB free")}
	}
	return row
}

// value: a Mac's sub line, by priority: what is down there, its agents at
// work and waiting, its upkeep (dropped first); with none of them, how
// many VMs it holds.
func (h homeReading) macFacts(m model.MacView) page.Value {
	down, working, waiting := 0, 0, 0
	for _, vm := range m.VMs {
		switch {
		case vm.Lifetime == model.AlwaysOn && model.Down(vm):
			down++
		case vm.Agent == nil || vm.Finished() || vm.State == model.Creating:
		case vm.Agent.State == string(agents.Waiting):
			waiting++
		default:
			working++
		}
	}
	var parts []page.Value
	if down > 0 {
		v := page.Text(strconv.Itoa(down) + " down")
		v.Signal = page.SignalFailing
		parts = append(parts, v)
	}
	if working > 0 {
		parts = append(parts, page.Text(strconv.Itoa(working)+" working"))
	}
	if waiting > 0 {
		parts = append(parts, page.Text(strconv.Itoa(waiting)+" waiting"))
	}
	if n := len(scoped(h.upkeep, m.Name)); n > 0 {
		parts = append(parts, dropFirst(page.Text(strconv.Itoa(n)+" upkeep"), 1))
	}
	if len(parts) == 0 {
		return page.Text(counted(len(m.VMs), "VM", "VMs"))
	}
	return page.List(parts...)
}
