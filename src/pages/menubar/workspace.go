package menubar

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/overview"
	"github.com/jterrazz/studio-station/src/page"
	pagehome "github.com/jterrazz/studio-station/src/pages/home"
	pagemachines "github.com/jterrazz/studio-station/src/pages/machines"
)

const (
	SectionFootnote = "footnote"
	SectionStopped  = "stopped"
)

func workspacePage(m Machine, h model.MacView, vm model.VM) page.Page {
	// invariant: SPEC A4 A workspace: a machine whose agents are objects of
	// their own: its Overview lists them first, then what needs you there,
	// its gauges, its ports and what its life means; its Ports segment lists
	// its ports by reach, its Accounts its loans; a stopped one says so with
	// Start and has no segment and no footer.
	p, _ := vmHead(m, h, vm)
	label := model.Label(*m.All, h)
	mine := agentsOn(m, h, vm)
	p.Header = &page.Header{Back: "All machines", Glyph: workspaceGlyph(vm, mine), Buttons: []string{ActionMenu}}
	p.Sections = []page.Section{{ID: SectionSummary, Title: vm.Name, Role: page.Summary, Caption: workspaceFacts(m, h, vm, label, mine)}}
	menu := page.Action{ID: ActionMenu, Label: "More options for " + vm.Name, Icon: "ellipsis", On: page.OnPage}
	live, shared, loans := liveAgents(mine), len(workspacePorts(m, vm).Shares), len(sharedAccounts(m, vm.Name))
	stop := stopWorkspace(vm, label, live, shared).action(m.Asks, page.Action{ID: ActionStop, Label: "Stop", On: page.OnPage, Priority: 1, Mutates: true,
		Run: &page.Run{Call: CallStopVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}})
	running := vm.State == model.Running
	if !running && !model.Down(vm) {
		stop = page.Action{ID: ActionStartVM, Label: "Start " + vm.Name, On: page.OnPage, Priority: 1, Mutates: true,
			Run: &page.Run{Call: CallStartVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}, Hud: hud(vm.Name + " is starting")}
	}
	if vm.State != model.Running {
		live = nil
	}
	del := deleteWorkspace(vm, live, shared, loans).action(m.Asks, page.Action{ID: ActionDelete, Label: "Delete", On: page.OnPage, Priority: 2, Mutates: true,
		Run: &page.Run{Call: CallDeleteVM, In: map[string]string{"VM": vm.Name}, Target: target(h)}})
	p.Actions = []page.Action{stop, del}
	if running {
		menu.Submenu = append(menu.Submenu, ActionShare)
	}
	menu.Submenu = append(menu.Submenu, stop.ID, ActionDelete)
	if !running {
		p.Sections = append(p.Sections, stoppedCard(vm))
		p.Actions = append(p.Actions, menu)
		p.Header.Buttons = []string{ActionMenu}
		return page.Seal(p)
	}
	p.Segments = segments()
	p.Sections = append(p.Sections, agentsCard(vm, mine))
	if needs := vmNeeds(m, h, vm, label); needs != nil {
		p.Sections = append(p.Sections, *needs)
	}
	shares := workspacePorts(m, vm)
	p.Sections = append(p.Sections, vmGauges(m, vm), workspacePortsCard(shares), footnote(m, vm, label))
	p.Sections = append(p.Sections, shares.Sections...)
	copies := sharedSection(m, h, vm)
	copies.Segment = SegmentAccounts
	p.Sections = append(p.Sections, copies)
	p.Actions = append(p.Actions, agentActions(h, vm, mine)...)
	p.Actions = append(p.Actions, sharePortSheets(shares.Listeners, []workspace{{VM: vm.Name, Mac: h.Name}}, m.self())...)
	if n, ok := needsOfVM(m, h, vm, label); ok {
		for _, need := range n {
			p.Actions = append(p.Actions, need.Verb)
		}
	}
	p.Actions = append(p.Actions, takeBacks(m, h, vm)...)
	p.Actions = append(p.Actions, shares.Actions...)
	shell := openIn("--on", vm.Name)
	shell.Key, shell.Icon, shell.Hud = "cmd+enter", "terminal", opened(vm.Name)
	ssh := catalog.SSHCommand("", vm.Name)
	p.Actions = append(p.Actions, shell, copied(ActionCopySSH, "Copy "+ssh, ssh), menu)
	p.Footer = &page.Footer{Kind: page.FooterSSH, Copy: &page.Value{Kind: page.KindCommand, Text: ssh}, Buttons: []string{ActionOpenStation}}
	p.Live = pageLive(p)
	return page.Seal(p)
}

// value: the agents a workspace hosts, by name, as the fleet's agents say
// them; until they are read, the one its guest last reported, named by
// its workspace.
// value: the names of the agents of a workspace that still run, which a
// stop or a delete of it ends.
func liveAgents(as []agents.Agent) []string {
	var out []string
	for _, a := range as {
		if a.State != agents.Finished {
			out = append(out, a.Name)
		}
	}
	return out
}

func agentsOn(m Machine, h model.MacView, vm model.VM) []agents.Agent {
	if m.Agents == nil {
		if vm.Agent == nil {
			return nil
		}
		return []agents.Agent{{Name: vm.Name, VM: vm.Name, Mac: h.Name, State: agents.State(vm.Agent.State), Task: vm.Agent.Task, Reason: vm.Agent.Reason, Ended: vm.Agent.Ended}}
	}
	var out []agents.Agent
	for _, a := range m.Agents {
		if a.VM == vm.Name && (a.Mac == "" || a.Mac == h.Name) {
			out = append(out, a)
		}
	}
	slices.SortStableFunc(out, func(a, b agents.Agent) int { return agentOrder(a) - agentOrder(b) })
	return out
}

// value: SPEC A4's order of agents: waiting, working, then finished.
func agentOrder(a agents.Agent) int {
	switch a.State {
	case agents.Waiting:
		return 0
	case agents.Finished:
		return 2
	case agents.Starting, agents.Working, agents.Running:
	}
	return 1
}

// value: SPEC A4's facts of a workspace: its life, its Mac, how long it
// has run and how many agents it hosts; a stopped one says it stopped. On
// a narrow line its uptime goes first, then its Mac, then its life: what
// it runs stays.
func workspaceFacts(m Machine, h model.MacView, vm model.VM, label string, mine []agents.Agent) []page.Value {
	out := []page.Value{dropFirst(page.Text(lifeWord(vm)), 1), dropFirst(page.Linked(page.Text("on "+label), machinePage(h.Name)), 2)}
	if vm.State != model.Running {
		return append(out, page.Text("stopped"))
	}
	if !vm.BootAt.IsZero() && m.Now.After(vm.BootAt) {
		out = append(out, dropFirst(page.Text("up "+days(m.Now.Sub(vm.BootAt))), 3))
	}
	live := 0
	for _, a := range mine {
		if agents.Live(a) {
			live++
		}
	}
	if live == 0 {
		return append(out, page.Text("no agent running"))
	}
	return append(out, page.Text(counted(live, "agent", "agents")))
}

// value: SPEC A4's Agents card: one row per agent, its state's words, its
// page behind it and its terminal in the quick slot; a finished one is
// quiet with its Clean up; none, the line that offers the first; the
// hook's absence where an agent runs unreported.
func agentsCard(vm model.VM, mine []agents.Agent) page.Section {
	s := page.Section{ID: SectionAgents, Title: "Agents", Segment: SegmentOverview, Card: page.CardFlush, Columns: columns(), Uncounted: true}
	if len(mine) == 0 {
		s.Card = page.CardEmpty
		s.Items = []page.Item{{Empty: &page.Empty{Text: "No agent on " + vm.Name, Actions: []string{ActionNewAgent}}}}
		return s
	}
	counts := map[agents.State]int{}
	for _, a := range mine {
		counts[a.State]++
		row := agentRow(a)
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	var words []page.Value
	for _, st := range []agents.State{agents.Working, agents.Running, agents.Waiting, agents.Finished} {
		if counts[st] > 0 {
			v := page.Text(strconv.Itoa(counts[st]) + " " + string(st))
			if st == agents.Waiting {
				v.Signal = page.SignalWaits
			}
			words = append(words, v)
		}
	}
	s.Caption = []page.Value{page.List(words...)}
	if a, ok := hookMissing(vm.Name, mine); ok {
		detail := page.Text("station only sees that " + appWord(a.App) + " runs")
		row := page.Row{ID: RowHook, Label: "Agent hook not installed", Mark: page.OK, Detail: detail, Glyph: symbol("exclamationmark.triangle", page.SignalWaits), Verb: ActionApplyHook}
		s.Items = append(s.Items, page.Item{Row: &row})
	}
	return s
}

// value: an agent's row on its workspace's page: home's agent row
// without its Mac.
func agentRow(a agents.Agent) page.Row {
	row := page.Row{ID: a.Name, Label: a.Name, Mark: page.OK, Nav: agentPage(a.Name), Glyph: page.Glyph{Kind: "agent", Signal: agentSignal(a)}}
	switch a.State {
	case agents.Waiting:
		ask := page.Text(cmpWord(a.Task, "waits for you"))
		if a.Ask != nil && a.Ask.Title != "" {
			ask = page.Text(a.Ask.Title)
		}
		ask.Signal = page.SignalWaits
		row.Detail = page.List(ask)
	case agents.Finished:
		row.Detail, row.Signal = page.Texts("finished", cmpWord(a.Reason, "done")), page.SignalDone
		row.Verb = ActionForget + a.Name
	case agents.Running:
		row.Detail = page.Text(appWord(a.App) + " running")
	default:
		if w := agents.Words(a); len(w) > 0 {
			row.Detail = page.Texts(w...)
		} else {
			row.Detail = page.Text(cmpWord(a.Task, "working"))
		}
	}
	if agents.Live(a) {
		row.Quick = ActionAttach + a.Name
	}
	return row
}

func agentActions(h model.MacView, vm model.VM, mine []agents.Agent) []page.Action {
	start := page.Action{ID: ActionNewAgent, Label: "New agent", Icon: "plus", Key: "shift+cmd+n", On: page.OnPage, Priority: 2, Mutates: true,
		Run:  &page.Run{Call: CallAgentsStart, In: map[string]string{"On": vm.Name}, Target: target(h)},
		Form: &page.Form{Title: "New agent on " + vm.Name, Fields: []page.Field{{Key: "Task", Label: "Task", Kind: page.FieldText, Note: "empty, it waits for a prompt"}}}}
	out := []page.Action{start}
	for _, a := range mine {
		if agents.Live(a) {
			out = append(out, attachAgent(a))
		} else {
			out = append(out, forgetAgent(h, a))
		}
	}
	if _, ok := hookMissing(vm.Name, mine); ok {
		out = append(out, applyHook(h, vm.Name))
	}
	return out
}

// value: what needs you on the workspace, its agents' waits left to its
// Agents card.
func needsOfVM(m Machine, h model.MacView, vm model.VM, label string) ([]Need, bool) {
	o := overviewOf(m, h, label, false)
	var out []Need
	for _, n := range scoped(o.needs, vm.Name) {
		if !strings.HasPrefix(n.Key, "waiting:") {
			out = append(out, n)
		}
	}
	return out, len(out) > 0
}

func vmNeeds(m Machine, h model.MacView, vm model.VM, label string) *page.Section {
	needs, ok := needsOfVM(m, h, vm, label)
	if !ok {
		return nil
	}
	o := overviewOf(m, model.MacView{Name: vm.Name}, label, false)
	o.needs = needs
	s := o.needsCard()
	return &s
}

// value: the workspace's gauges, read inside it.
func vmGauges(m Machine, vm model.VM) page.Section {
	s := macOverview{m: m, h: model.MacView{Name: vm.Name}}.gauges()
	return s
}

// value: what a workspace's life means, and whether it pauses when its
// Mac sleeps.
func footnote(m Machine, vm model.VM, label string) page.Section {
	life := "Persistent: kept until you delete it"
	if vm.Lifetime == model.Ephemeral {
		life = "Ephemeral: deleted when it powers off from inside"
	}
	sleeps := "pauses when " + label + " sleeps"
	if m.Awake != nil && m.Awake.Mode() == awake.NeverSleeps {
		sleeps = "never paused"
	}
	note := page.Note{Text: life + "; " + sleeps + "."}
	return page.Section{ID: SectionFootnote, Title: "Life", Role: page.Banner, Segment: SegmentOverview, Uncounted: true, Items: []page.Item{{Note: &note}}}
}

// value: SPEC A4's stopped workspace: what it keeps and the memory it
// takes again, with Start.
func stoppedCard(vm model.VM) page.Section {
	sub := []string{"Disk kept"}
	if b := vm.Resources.MemoryBytes; b > 0 {
		sub = append(sub, "starts with "+pagemachines.Memory(b).Text)
	}
	empty := page.Empty{Glyph: page.Glyph{Kind: "workspace", Signal: page.SignalQuiet}, Text: vm.Name + " is stopped", Sub: strings.Join(sub, ", "), Actions: []string{ActionStartVM}}
	return page.Section{ID: SectionStopped, Title: vm.Name + " is stopped", Card: page.CardEmpty, Items: []page.Item{{Empty: &empty}}}
}

func pluralVerb(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
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

// value: a workspace's Overview Ports card: its shares, each opening its
// link with Copy and QR beside, then its first listener offered with
// Share; nothing shared nor listening, the one line Nothing shared yet.
func workspacePortsCard(b portsBody) page.Section {
	items := slices.Clone(b.Shares)
	if len(b.Listeners) > 0 {
		items = append(items, b.Listeners[0])
	}
	if len(items) == 0 {
		items = []page.Item{{Empty: &page.Empty{Text: "Nothing shared yet"}}}
	}
	return page.Section{ID: SectionPorts, Title: TitlePorts, Segment: SegmentOverview, Caption: []page.Value{count(len(b.Shares))}, Columns: columns(), Items: items}
}
