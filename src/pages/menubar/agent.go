package menubar

import (
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/src/agents"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
)

const (
	AgentID page.ID = "menubar.agent"

	SectionWait   = "wait"
	SectionNow    = "now"
	SectionRunsOn = "runs-on"
	RowHook       = "hook"

	ActionNewAgent   = "new-agent"
	ActionOpenPR     = "open-pr"
	ActionMerge      = "merge"
	ActionReply      = "reply"
	ActionFullLog    = "full-log"
	ActionAttachHere = "attach"
	ActionStopAgent  = "stop-agent"
	ActionForget     = "forget:"
	ActionApplyHook  = "apply-hook"

	CallAgentsStart  = "agents.start"
	CallAgentsReply  = "agents.reply"
	CallAgentsMerge  = "agents.merge"
	CallAgentsStop   = "agents.stop"
	CallAgentsForget = "agents.forget"

	// value: the steps an agent's Now card shows (SPEC A4).
	stepsShown = 3
)

type Agent struct {
	// value: the agent the page is of, by its name; the fleet's machines
	// and its agents, nil while they are still being read.
	Name   string
	All    *model.AllMachines
	Agents []agents.Agent
	Now    time.Time
	Asks   Levels
}

func agentPage(name string) page.ID { return page.ID(string(AgentID) + ":" + name) }

func AgentPage(r Agent) page.Page {
	// invariant: SPEC A4 An agent: what the agent says and asks, read from
	// its own report, never invented: its wait first when it waits on the
	// owner, its steps while it works, its last ones once it finished, then
	// the workspace it runs on; its terminal is its footer while it lives.
	if r.All == nil || r.Agents == nil {
		row := page.Row{ID: "reading", Label: r.Name, Mark: page.Awaiting}
		return page.Seal(page.Page{ID: AgentID, Command: "menubar", Context: page.Context{Machine: r.Name, Kind: kindAgent},
			Sections: []page.Section{{ID: SectionSummary, Title: r.Name, Role: page.Summary, Items: []page.Item{{Row: &row}}}}})
	}
	i := slices.IndexFunc(r.Agents, func(a agents.Agent) bool { return a.Name == r.Name })
	if i < 0 {
		return page.Seal(page.Page{ID: AgentID, Command: "menubar", Context: page.Context{Machine: r.Name, Kind: kindAgent},
			Error: &page.Error{Message: "no agent is named " + r.Name, Hint: "station agents"}})
	}
	a := r.Agents[i]
	h, vm, _ := vmOf(*r.All, a)
	p := page.Page{ID: AgentID, Command: "menubar", Context: page.Context{Machine: a.Name, Kind: kindAgent, On: a.VM}}
	p.Header = &page.Header{Back: "All machines", Glyph: page.Glyph{Kind: "agent", Signal: agentSignal(a)}, Buttons: []string{ActionMenu}}
	p.Sections = []page.Section{{ID: SectionSummary, Title: a.Name, Role: page.Summary, Caption: agentFacts(r, a, h)}}
	run := agentRun(h, a)
	menu := page.Action{ID: ActionMenu, Label: "More options for " + a.Name, Icon: "ellipsis", On: page.OnPage}
	if a.State == agents.Waiting && a.Ask != nil {
		p.Sections = append(p.Sections, waitCard(r, a))
		p.Actions = append(p.Actions, waitActions(a, run)...)
	} else {
		p.Sections = append(p.Sections, nowCard(r, a))
	}
	p.Sections = append(p.Sections, runsOn(r, h, vm, a))
	if agents.Live(a) {
		attach := attachAgent(a)
		attach.ID, attach.Label, attach.Rows, attach.On = ActionAttachHere, "Open in Terminal", nil, page.OnPage
		log := attachAgent(a)
		log.ID, log.Label, log.Rows, log.On, log.Key = ActionFullLog, "Full log", nil, page.OnPage, ""
		stop := page.Action{ID: ActionStopAgent, Label: "Stop " + a.Name, On: page.OnPage, Priority: 4, Mutates: true, Destructive: true,
			Confirm: stopAsk(r.Asks, a), Run: run(CallAgentsStop, nil), Hud: hud(a.Name + " stopped")}
		stop.Run.In = answered(stop.Confirm, stop.Run.In)
		p.Actions = append(p.Actions, attach, log, stop)
		menu.Submenu = append(menu.Submenu, ActionStopAgent)
		p.Footer = &page.Footer{Kind: page.FooterSSH, Copy: &page.Value{Kind: page.KindCommand, Text: "station agents attach " + a.Name}, Buttons: []string{ActionAttachHere}}
	} else {
		forget := forgetAgent(h, a)
		forget.Rows = []string{SectionNow}
		p.Actions = append(p.Actions, forget)
		menu.Submenu = append(menu.Submenu, forget.ID)
	}
	p.Actions = append(p.Actions, menu)
	return page.Seal(p)
}

const kindAgent = "agent"

// value: the Mac and the workspace an agent runs on, as the fleet reads
// them; false for a workspace no Mac lists.
func vmOf(all model.AllMachines, a agents.Agent) (model.MacView, model.VM, bool) {
	for _, h := range all.Macs {
		if a.Mac != "" && h.Name != a.Mac {
			continue
		}
		for _, vm := range h.VMs {
			if vm.Name == a.VM {
				return h, vm, true
			}
		}
	}
	return model.MacView{Name: a.Mac}, model.VM{Name: a.VM}, false
}

// value: the agent's mark by its state: orange while it waits on the
// owner, done once finished, active otherwise.
func agentSignal(a agents.Agent) page.Signal {
	switch a.State {
	case agents.Waiting:
		return page.SignalWaits
	case agents.Finished:
		return page.SignalDone
	case agents.Starting, agents.Working, agents.Running:
	}
	return page.SignalActive
}

func appWord(app agents.App) string { return catalog.AgentAppName(string(app)) }

// value: SPEC A4's facts of an agent: its state and for how long, how it
// ended once finished, its app, the workspace it runs on, then its Mac when
// it is not This Mac. On a narrow line the Mac goes first, then the app,
// then the workspace: the state stays.
func agentFacts(r Agent, a agents.Agent, h model.MacView) []page.Value {
	state := page.Text(string(a.State))
	switch {
	case a.State == agents.Finished:
		state = page.Texts(string(a.State), cmpWord(a.Reason, "done"))
	case a.State == agents.Waiting && a.Ask != nil && !a.Ask.At.IsZero():
		state = page.Texts(string(a.State), minutes(r.Now.Sub(a.Ask.At)))
	case a.State != agents.Running && !a.Since.IsZero() && r.Now.After(a.Since):
		state = page.Texts(string(a.State), minutes(r.Now.Sub(a.Since)))
	}
	if a.State == agents.Waiting {
		state.Signal = page.SignalWaits
	}
	out := []page.Value{state, dropFirst(page.Text(appWord(a.App)), 2), dropFirst(page.Linked(page.Text("on "+a.VM), machinePage(a.VM)), 1)}
	if !h.ThisMac && h.Name != "" {
		out = append(out, dropFirst(page.Text(model.Label(*r.All, h)), 3))
	}
	return out
}

// value: SPEC A4's Waiting on you: the agent's own line, its detail, its PR
// and change, how long it has waited, and the reply it takes.
func waitCard(r Agent, a agents.Agent) page.Section {
	m := page.Message{Title: a.Ask.Title, Body: a.Ask.Detail, Reply: true}
	if !a.Ask.At.IsZero() {
		m.Since = page.Text(minutes(r.Now.Sub(a.Ask.At)))
	}
	if a.PR != nil {
		m.PR = &page.PR{Number: a.PR.Number, URL: a.PR.URL}
		if a.Change != nil {
			m.PR.Files, m.PR.Plus, m.PR.Minus = a.Change.Files, a.Change.Plus, a.Change.Minus
		}
	}
	return page.Section{ID: SectionWait, Title: "Waiting on you", Card: page.CardWait, Items: []page.Item{{Message: &m}}}
}

func waitActions(a agents.Agent, run func(string, map[string]string) *page.Run) []page.Action {
	reply := page.Action{ID: ActionReply, Label: "Reply to " + a.Name, Key: "enter", On: page.OnPage, Priority: 3, Mutates: true,
		Run:  run(CallAgentsReply, nil),
		Form: &page.Form{Title: "Reply to " + a.Name, Fields: []page.Field{{Key: "Text", Label: "Reply to " + a.Name, Kind: page.FieldText, Required: true}}},
		Hud:  hud("Sent to "+a.Name, "it works on it")}
	if a.PR == nil {
		return []page.Action{reply}
	}
	pr := "PR #" + strconv.Itoa(a.PR.Number)
	return []page.Action{
		{ID: ActionOpenPR, Label: "Open PR", Icon: "arrow.up.right", On: page.OnPage, Priority: 1, Default: true, Run: &page.Run{Open: a.PR.URL}, Hud: hud("Opens " + pr)},
		{ID: ActionMerge, Label: "Merge " + pr, On: page.OnPage, Priority: 2, Mutates: true, Run: run(CallAgentsMerge, nil), Hud: hud(pr+" merged", a.Name+" finished")},
		reply,
	}
}

// value: SPEC A4's Now: its task, then its last steps, the current one
// first and said as it is being done; once finished, its Last steps. An
// agent no hook reports says only that its app runs.
func nowCard(r Agent, a agents.Agent) page.Section {
	s := page.Section{ID: SectionNow, Title: "Now", Card: page.CardNow, Caption: []page.Value{page.Text(cmpWord(a.Task, "waits for a prompt"))}}
	if a.State == agents.Finished {
		s.Title = "Last steps"
	}
	if a.State == agents.Running {
		s.Caption = []page.Value{page.Text(appWord(a.App) + " running")}
		return s
	}
	var steps []page.Step
	for i := len(a.Steps) - 1; i >= 0 && len(steps) < stepsShown; i-- {
		st := a.Steps[i]
		current := i == len(a.Steps)-1 && a.State == agents.Working
		parts := agents.Done(st)
		at := page.Text(minutes(r.Now.Sub(st.At)) + " ago")
		if current {
			parts, at = agents.Words(a), page.Text("now")
		}
		step := page.Step{At: at, Text: parts[0], Current: current}
		if len(parts) > 1 {
			step.Diff = parts[1]
		}
		steps = append(steps, step)
	}
	if len(steps) > 0 {
		s.Items = []page.Item{{Steps: steps}}
	}
	return s
}

// value: SPEC A4's Runs on: the agent's workspace, its Mac, its life and
// how long it has run, which opens its page.
func runsOn(r Agent, h model.MacView, vm model.VM, a agents.Agent) page.Section {
	row := page.Row{ID: a.VM, Label: a.VM, Mark: page.OK, Nav: machinePage(a.VM), Glyph: workspaceGlyph(vm, r.Agents)}
	var parts []page.Value
	if h.Name != "" {
		parts = append(parts, page.Text(model.Label(*r.All, h)))
	}
	if vm.Lifetime != "" {
		parts = append(parts, page.Text(lifeWord(vm)))
	}
	switch {
	case vm.State != "" && vm.State != model.Running:
		parts = append(parts, page.Text("stopped"))
	case a.State == agents.Finished && !a.Ended.IsZero() && r.Now.After(a.Ended):
		parts = append(parts, page.Text("idle "+minutes(r.Now.Sub(a.Ended))))
	case !vm.BootAt.IsZero() && r.Now.After(vm.BootAt):
		parts = append(parts, page.Text("up "+days(r.Now.Sub(vm.BootAt))))
	}
	row.Detail = page.List(parts...)
	return page.Section{ID: SectionRunsOn, Title: "Runs on", Card: page.CardFlush, Columns: columns(), Uncounted: true, Items: []page.Item{{Row: &row}}}
}

// value: a workspace's mark: the agent's while it hosts a live one, the
// box otherwise.
func workspaceGlyph(vm model.VM, all []agents.Agent) page.Glyph {
	for _, a := range all {
		if a.VM == vm.Name && agents.Live(a) {
			return page.Glyph{Kind: "agent", Signal: agentSignal(a)}
		}
	}
	return page.Glyph{Kind: "workspace", Signal: page.SignalQuiet}
}

// value: where an agent's call runs: inside its workspace, reached
// through the Mac that holds it.
func agentRun(h model.MacView, a agents.Agent) func(string, map[string]string) *page.Run {
	return func(call string, in map[string]string) *page.Run {
		if in == nil {
			in = map[string]string{}
		}
		in["Agent"] = a.Name
		t := &page.Target{Name: a.VM}
		if !h.ThisMac && h.Name != "" {
			t.On = h.Name
		}
		return &page.Run{Call: call, In: in, Target: t}
	}
}

// value: the agent's own terminal, opened from this Mac, which finds its
// workspace and its Mac by its name.
func attachAgent(a agents.Agent) page.Action {
	return page.Action{ID: ActionAttach + a.Name, Label: "Open " + a.Name + " in Terminal", Icon: "terminal", Key: "cmd+enter", On: page.OnRow, Rows: []string{a.Name},
		Priority: 2, Run: &page.Run{Terminal: true, Argv: []string{"agents", "attach", a.Name}}}
}

// value: SPEC B5.9: Clean up on a finished agent forgets it; its
// workspace stays.
func forgetAgent(h model.MacView, a agents.Agent) page.Action {
	return page.Action{ID: ActionForget + a.Name, Label: "Clean up", On: page.OnRow, Rows: []string{a.Name}, Priority: 3, Mutates: true,
		Run: agentRun(h, a)(CallAgentsForget, nil), Hud: hud(a.Name + " cleaned up")}
}

func stopAsk(levels Levels, a agents.Agent) *page.Confirm {
	return ask(levels, "stop-agent", machine.Workspace, a.Name, "Stop "+a.Name+"?", "stop", "Its terminal closes; "+a.VM+" keeps running.")
}

// value: SPEC B5.1: the hook that reports an agent is missing on its
// workspace while an agent runs there unreported: Apply installs it.
func hookMissing(vm string, as []agents.Agent) (agents.Agent, bool) {
	for _, a := range as {
		if a.VM == vm && a.State == agents.Running {
			return a, true
		}
	}
	return agents.Agent{}, false
}

func applyHook(h model.MacView, vm string) page.Action {
	t := &page.Target{Name: vm}
	if !h.ThisMac {
		t.On = h.Name
	}
	return page.Action{ID: ActionApplyHook, Label: "Apply", On: page.OnRow, Rows: []string{RowHook}, Priority: 3, Mutates: true,
		Run: &page.Run{Call: CallSettingsApply, In: map[string]string{"Names": `["` + catalog.AgentHooksSetting + `"]`, "Confirm": yes}, Target: t},
		Hud: hud("Agent hook installed on " + vm)}
}
