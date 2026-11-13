package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/agents"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

func fleetAgents() []agents.Agent {
	return []agents.Agent{
		{Name: "review-bot", VM: "review-bot", Mac: miniMac, App: "claudecode", State: agents.Waiting,
			Ask: &agents.Ask{Title: "PR #42 is ready", At: macNow.Add(-6 * time.Minute)}, PR: &agents.PR{Number: 42, URL: "https://github.com/jterrazz/app/pull/42"}},
		{Name: "review-bot" + agents.Nth + "2", VM: "review-bot", Mac: miniMac, App: "claudecode", State: agents.Finished, Reason: "tests pass", Ended: macNow.Add(-time.Minute)},
		{Name: "linux-3", VM: "linux-3", Mac: miniMac, App: "claudecode", State: agents.Running},
	}
}

func agentPageOf(t *testing.T, name string) page.Page {
	t.Helper()
	page.Strict(true)
	defer page.Strict(false)
	return AgentPage(Agent{Name: name, All: macFleet(), Agents: fleetAgents(), Now: macNow, Asks: func(string, coremachine.Kind, ...string) string { return LevelYesNo }})
}

func agentActionOf(p page.Page, id string) *page.Action {
	for i := range p.Actions {
		if p.Actions[i].ID == id {
			return &p.Actions[i]
		}
	}
	return nil
}

// invariant: what the owner does on an agent's page runs inside its
// workspace, reached through the Mac that holds it, and addresses the
// agent by its name.
func TestAnAgentsGesturesRunInsideItsWorkspaceByItsName(t *testing.T) {
	p := agentPageOf(t, "review-bot")
	for _, id := range []string{ActionMerge, ActionReply, ActionStopAgent} {
		a := agentActionOf(p, id)
		if a == nil || a.Run.Target == nil || a.Run.Target.Name != "review-bot" || a.Run.Target.On != miniMac || a.Run.In["Agent"] != "review-bot" {
			t.Fatalf("%s runs %+v", id, a)
		}
	}
	if open := agentActionOf(p, ActionOpenPR); open == nil || open.Run.Open != "https://github.com/jterrazz/app/pull/42" || !open.Default {
		t.Fatalf("Open PR is %+v", open)
	}
}

// invariant: a workspace lists every agent it hosts, waiting first; a
// finished one is cleaned up on its own, its workspace untouched.
func TestAWorkspaceListsItsAgentsAndCleansUpAFinishedOneAlone(t *testing.T) {
	p := macPageOf(t, "review-bot", func(m *Machine) { m.Agents = fleetAgents() })
	card := sectionOf(p, SectionAgents)
	if card == nil || len(card.Items) != 2 || card.Items[0].Row.ID != "review-bot" || card.Items[1].Row.ID != "review-bot"+agents.Nth+"2" {
		t.Fatalf("agents card %+v", card)
	}
	done := card.Items[1].Row
	clean := agentActionOf(p, done.Verb)
	if clean == nil || clean.Run.Call != CallAgentsForget || clean.Run.In["Agent"] != "review-bot"+agents.Nth+"2" {
		t.Fatalf("Clean up runs %+v", clean)
	}
	if slices.ContainsFunc(p.Sections, func(s page.Section) bool { return s.ID == SectionNeeds }) {
		t.Fatal("the agent's wait is said twice: in the Agents card and in Needs you")
	}
}

// invariant: SPEC A4: a workspace's Agents card says the agent that waits
// on the owner first, then the ones at work, then the finished, whatever
// order the fleet reads them in.
func TestAWorkspaceListsTheWaitingAgentBeforeTheWorkingOnes(t *testing.T) {
	fleet := fleetAgents()
	working := agents.Agent{Name: "review-bot" + agents.Nth + "3", VM: "review-bot", Mac: miniMac, App: "claudecode", State: agents.Working, Task: "fixes the docs"}
	fleet = append([]agents.Agent{fleet[1], working}, append(fleet[:1:1], fleet[2:]...)...)
	p := macPageOf(t, "review-bot", func(m *Machine) { m.Agents = fleet })
	card := sectionOf(p, SectionAgents)
	got := make([]string, 0, len(card.Items))
	for _, it := range card.Items {
		got = append(got, it.Row.ID)
	}
	want := []string{"review-bot", working.Name, "review-bot" + agents.Nth + "2"}
	if !slices.Equal(got, want) {
		t.Fatalf("agents card lists %v, want %v", got, want)
	}
}

// invariant: SPEC A4: once a workspace's only agent has finished, its
// Runs on row wears the box, whatever its guest last reported.
func TestAWorkspaceWhoseAgentFinishedWearsTheBox(t *testing.T) {
	done := fleetAgents()[1]
	done.Name = "review-bot"
	page.Strict(true)
	defer page.Strict(false)
	p := AgentPage(Agent{Name: "review-bot", All: macFleet(), Agents: []agents.Agent{done}, Now: macNow, Asks: func(string, coremachine.Kind, ...string) string { return LevelYesNo }})
	runs := sectionOf(p, SectionRunsOn)
	if runs == nil || runs.Items[0].Row.Glyph.Kind != "workspace" {
		t.Fatalf("Runs on %+v", runs)
	}
}

// invariant: SPEC B5.1: an agent no hook reports says only that its app
// runs, and its workspace offers to install the hook.
func TestAnAgentNoHookReportsOffersTheHook(t *testing.T) {
	p := macPageOf(t, "linux-3", func(m *Machine) { m.Agents = fleetAgents() })
	var hook *page.Row
	for _, it := range sectionOf(p, SectionAgents).Items {
		if it.Row != nil && it.Row.ID == RowHook {
			hook = it.Row
		}
	}
	if hook == nil || hook.Label != "Agent hook not installed" || hook.Detail.Text != "station only sees that Claude runs" {
		t.Fatalf("hook row %+v", hook)
	}
	if a := agentActionOf(p, hook.Verb); a == nil || a.Run.Call != CallSettingsApply || a.Run.Target.Name != "linux-3" {
		t.Fatalf("Apply runs %+v", a)
	}
}

// invariant: SPEC A4: an agent's facts name its workspace as a link that
// opens it, and a workspace's facts name its Mac as a link that opens it.
func TestAnAgentsFactsOpenItsWorkspaceAndAWorkspacesOpenItsMac(t *testing.T) {
	linked := func(p page.Page, text string) page.ID {
		for _, v := range sectionOf(p, SectionSummary).Caption {
			if v.Text == text {
				return v.Nav
			}
		}
		t.Fatalf("no %q in the facts of %s", text, p.ID)
		return ""
	}
	if nav := linked(agentPageOf(t, "review-bot"), "on review-bot"); nav != machinePage("review-bot") {
		t.Fatalf("on review-bot opens %q", nav)
	}
	if nav := linked(macPageOf(t, "review-bot", func(m *Machine) { m.Agents = fleetAgents() }), "on mac-mini"); nav != machinePage(miniMac) {
		t.Fatalf("on mac-mini opens %q", nav)
	}
}
