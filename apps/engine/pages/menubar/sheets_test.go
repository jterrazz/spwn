package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/agents"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

var sheetNow = time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)

func fleetOfTwo() model.AllMachines {
	finished := &model.AgentView{State: model.AgentFinished, Ended: sheetNow.Add(-12 * time.Minute), Reason: "tests pass"}
	working := &model.AgentView{State: "working"}
	return model.AllMachines{Macs: []model.MacView{
		{Name: "pro", ThisMac: true, Reachable: true, VMs: []model.VM{
			{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running, Agent: working},
			{Name: "agent-7f", Lifetime: model.Ephemeral, State: model.Running, Agent: finished, SizeGB: 12},
			{Name: "docs", Lifetime: model.Persistent, State: model.Running, Agent: &model.AgentView{State: model.AgentFinished, Ended: sheetNow.Add(-time.Hour)}},
		}},
		{Name: "mini", Reachable: true, VMs: []model.VM{
			{Name: "old", Lifetime: model.Ephemeral, State: model.Stopped, SizeGB: 8},
			{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running},
		}},
		{Name: "air", Reachable: true, Asleep: true, VMs: []model.VM{{Name: "away", Lifetime: model.Ephemeral, State: model.Running}}},
	}}
}

// value: the fleet's agents, each named apart from its VM as the New agent
// sheet names them: a finished one on an ephemeral workspace, one finished beside a
// working one, one on a workspace, one on another Mac.
func sheetAgents() []agents.Agent {
	done := func(name, vm, mac string) agents.Agent {
		return agents.Agent{Name: name, VM: vm, Mac: mac, State: agents.Finished, Ended: sheetNow.Add(-12 * time.Minute), Reason: "tests pass"}
	}
	return []agents.Agent{
		{Name: "linux-1", VM: "linux-1", Mac: "pro", State: agents.Working},
		done("changelog-bot", "agent-7f", "pro"),
		done("docs-bot", "linux-1", "pro"),
		done("docs", "docs", "pro"),
		done("infra-2", "infra", "mini"),
	}
}

func fieldOf(a page.Action, key string) page.Field {
	for _, f := range a.Form.Fields {
		if f.Key == key {
			return f
		}
	}
	return page.Field{}
}

// invariant: SPEC A5 Clean up across every Mac: the empty ephemeral workspaces of
// each Mac deleted on that Mac, every finished agent listed and forgotten
// by its own name in its own VM, the one an ephemeral workspace carries too, should
// the owner keep the ephemeral workspace; the button worded from what is checked,
// red once an ephemeral workspace is; the note names what an agent still works in.
func TestCleanUpDeletesEachMacsEphemeralsAndForgetsTheAgentsLeft(t *testing.T) {
	all := fleetOfTwo()
	a := cleanupSheet(ActionCleanup, all, cleanupsOf(all.Macs, sheetAgents()), sheetNow, false)
	calls := make([]string, 0, 1+len(a.Run.And))
	calls = append(calls, a.Run.Call+" "+a.Run.In["VMs"])
	for _, r := range a.Run.And {
		on := ""
		if r.Target != nil {
			on = " on " + r.Target.Name + "/" + r.Target.On
		}
		calls = append(calls, r.Call+" "+r.In["VMs"]+r.In["Agent"]+on)
	}
	want := []string{`machines.cleanup ["agent-7f"]`, `agents.forget changelog-bot on agent-7f/`, `agents.forget docs-bot on linux-1/`, `agents.forget docs on docs/`,
		`machines.cleanup ["old"] on mini/`, `agents.forget infra-2 on infra/mini`}
	if !slices.Equal(calls, want) {
		t.Fatalf("Clean up runs %q, want %q", calls, want)
	}
	finished, vms := fieldOf(a, "Agent"), fieldOf(a, "VMs")
	if finished.Follows != "VMs" || finished.Options[0].Of != "agent-7f" || finished.Options[1].Of != "" || finished.Default != "docs-bot,docs,infra-2" {
		t.Errorf("the finished agents %+v: changelog-bot goes with its ephemeral workspace, the others are checked on their own", finished)
	}
	if o := finished.Options[1]; o.Label != "docs-bot" || o.Note != "on linux-1{sep}finished 12 min ago{sep}tests pass" {
		t.Errorf("the agent beside a working one is listed as %+v", o)
	}
	if vms.Default != "agent-7f,old" || vms.Options[1].Note != "mini{sep}stopped{sep}8 GB disk" {
		t.Errorf("the ephemeral workspaces %+v", vms)
	}
	if note := fieldOf(a, "stays").Note; note != "linux-1 has a working agent, so it stays. away is still running, so it stays." {
		t.Errorf("the note says %q", note)
	}
	words := make([]string, 0, len(a.Form.Primary))
	for _, p := range a.Form.Primary {
		words = append(words, p.Label)
		if p.Danger != (p.When.Key == "VMs") {
			t.Errorf("%q is red: %v", p.Label, p.Danger)
		}
	}
	if want := []string{"Delete 2 ephemeral workspaces", "Delete 1 ephemeral workspace", "Forget 4 agents", "Forget 3 agents", "Forget 2 agents", "Forget 1 agent"}; !slices.Equal(words, want) {
		t.Errorf("the button says %q, want %q", words, want)
	}
}

// invariant: SPEC A5 Clean up on one Mac: its title names the Mac, its
// ephemeral workspaces say no Mac, and it deletes and forgets on that Mac alone.
func TestCleanUpOnAMacStaysOnThatMac(t *testing.T) {
	all := fleetOfTwo()
	a := cleanupSheet(ActionCleanup+":mini", all, cleanupsOf(all.Macs[1:2], sheetAgents()), sheetNow, true)
	if a.Form.Title != "Clean up on mini" || a.Run.Target.Name != "mini" || len(a.Run.And) != 1 || fieldOf(a, "VMs").Options[0].Note != "stopped{sep}8 GB disk" {
		t.Errorf("Clean up on mini: %+v, runs %+v", a.Form, a.Run)
	}
	if o := fieldOf(a, "Agent").Options; len(o) != 1 || o[0].Value != "infra-2" {
		t.Errorf("Clean up on mini offers the agents %+v", o)
	}
}

// invariant: SPEC A5 Clean up: a VM whose agent waits on the owner is not
// said working.
func TestCleanUpSaysAWaitingAgentIsLive(t *testing.T) {
	if got := stays([]string{"linux-1", "review-bot"}, true, nil); got != "linux-1 and review-bot have a live agent, so they stay." {
		t.Errorf("the note says %q", got)
	}
}

// invariant: SPEC A5 Grant a password: one sheet per machine a password
// may go to, To swapping between them; This Mac's passwords as a pick
// list, the one that cannot go there disabled with the engine's line; the
// button names the password, and the call is grants.add.
func TestGrantAPasswordOffersEachMachineThePasswordsThatFit(t *testing.T) {
	to := runningWorkspaces(fleetOfTwo())
	if want := []workspace{{VM: "linux-1", Mac: "pro"}, {VM: "agent-7f", Mac: "pro"}, {VM: "docs", Mac: "pro"}}; !slices.Equal(to, want) {
		t.Fatalf("the workspaces %v, want %v", to, want)
	}
	deploy := accounts.VaultEntry{ID: "pw-1", Label: "deploy token", Program: programFor(t, catalog.ProviderGitHub)}
	mail := accounts.VaultEntry{ID: "pw-2", Label: "mail token", Program: programFor(t, catalog.ProviderClaude)}
	target := func(vm string, fits accounts.TargetFit, line string) accounts.Target {
		return accounts.Target{ToKind: accounts.ToMachine, To: vm, Fit: fits, Line: line}
	}
	s := SignIns{
		Vault: &accounts.VaultRead{Mac: "pro", Entries: []accounts.VaultEntry{deploy, mail}},
		Offers: map[string]accounts.GrantOffer{
			"pw-1": {Targets: []accounts.Target{target("linux-1", accounts.TargetFits, "Signed out here"), target("docs", accounts.TargetFits, "Signed out here"), target("linux-3", accounts.TargetNo, "It is on mini: grant from there")}, Presence: "Touch ID will ask you"},
			"pw-2": {Targets: []accounts.Target{target("linux-1", accounts.TargetNo, "It has its own sign-in"), target("docs", accounts.TargetFits, "Signed out here")}},
		},
	}
	b := accountsBody{s: s, self: "pro"}
	sheets := b.grantSheets()
	ids := make([]string, 0, len(sheets))
	for _, a := range sheets {
		ids = append(ids, a.ID)
	}
	if want := []string{"grant", "grant:linux-1", "grant:docs"}; !slices.Equal(ids, want) {
		t.Fatalf("the sheets %q, want %q", ids, want)
	}
	first, docs := sheets[1], sheets[2]
	pick := first.Form.Fields[1]
	if first.Sheet != page.SheetGrant || first.Run.Call != CallGrant || first.Run.In["To"] != "linux-1" || first.Run.In["ToKind"] != string(accounts.ToMachine) || first.Form.Fields[0].Default != "grant:linux-1" {
		t.Errorf("linux-1's sheet: %+v runs %+v", first.Form, first.Run)
	}
	if len(pick.Options) != 2 || pick.Default != "pw-1" || pick.Options[1].Disabled != "It has its own sign-in" || pick.Options[0].Disabled != "" {
		t.Errorf("linux-1's passwords %+v", pick)
	}
	if got := docs.Form.Fields[1]; got.Default != "pw-1" || got.Options[1].Disabled != "" || docs.Run.In["To"] != "docs" {
		t.Errorf("docs' passwords %+v runs %+v", got, docs.Run)
	}
	if p := first.Form.Primary[0]; p.Label != "Grant {Entry}" || p.When.Key != "Entry" || !p.When.Set {
		t.Errorf("the button %+v", p)
	}
	if first.Form.Subtitle != "From the passwords of This Mac. It stays until you remove it. Touch ID will ask you." {
		t.Errorf("the subtitle %q", first.Form.Subtitle)
	}
}

// invariant: SPEC A5 Share a port: nothing typed: each listener of the
// workspace with its Share, the action its row runs; none, the line that
// says so.
func TestShareAPortListsWhatListensInEachWorkspace(t *testing.T) {
	listener := func(key, verb string) page.Item {
		return page.Item{Row: &page.Row{ID: key, Label: key, Verb: verb}}
	}
	rows := []page.Item{listener("linux-1:4321", "share:linux-1:4321"), listener("linux-3:5432", "share:linux-3:5432")}
	sheets := sharePortSheets(rows, []workspace{{VM: "linux-1", Mac: "pro"}, {VM: "review-bot", Mac: "mini"}}, "pro")
	if len(sheets) != 3 || sheets[0].Run != nil {
		t.Fatalf("the sheets %+v", sheets)
	}
	first, empty := sheets[1].Form.Fields[1], sheets[2].Form.Fields[1]
	if len(first.Options) != 1 || first.Options[0].Value != "share:linux-1:4321" || first.Label != "Listening now in linux-1" {
		t.Errorf("linux-1's listeners %+v", first)
	}
	if len(empty.Options) != 0 || empty.Note != "Nothing listens in review-bot right now." {
		t.Errorf("review-bot's %+v", empty)
	}
	if of := sheets[0].Form.Fields[0]; of.Options[1].Note != "on mini" || !of.Swaps || of.Default != ActionSharePortSheet+"linux-1" {
		t.Errorf("Of %+v", of)
	}
}

// invariant: the Add a Mac hint copies the real command, naming the Mac to add.
func TestTheAddMacHintCopiesTheCommandThatAddsTheMac(t *testing.T) {
	if got := addMacCommands()[1]; got != "station machines add <mac>" {
		t.Errorf("hint command = %q", got)
	}
}

// invariant: a Mac's VMs come from the engine by name; a sheet picks live
// work first and a question names it first, a finished ephemeral workspace last.
func TestAFinishedEphemeralIsNeverWhatASheetPicksFirst(t *testing.T) {
	fleet := fleetOfTwo()
	this := &fleet.Macs[0]
	this.VMs[0], this.VMs[1] = this.VMs[1], this.VMs[0]
	if to := runningWorkspaces(fleet); len(to) < 2 || to[0].VM != "linux-1" || to[1].VM != "agent-7f" {
		t.Errorf("the workspaces %v, want linux-1 before agent-7f", to)
	}
	sheets := sharePortSheets(nil, runningWorkspaces(fleet), "pro")
	if of := sheets[0].Form.Fields[0]; of.Default != ActionSharePortSheet+"linux-1" || of.Options[0].Label != "linux-1" {
		t.Errorf("Of picks %q first of %+v", of.Default, of.Options)
	}
	stops := 0
	for _, a := range macAlerts(Machine{Asks: graded}, *this, page.ThisMac) {
		if a.ID == ActionStopAll {
			stops++
			if got := consequence(a.Confirm); got != "linux-1, agent-7f, docs stop and keep their disk. Awake ends." {
				t.Errorf("Stop everything says %q", got)
			}
		}
	}
	if stops != 1 {
		t.Errorf("%d Stop everything", stops)
	}
}

// invariant: one gesture, one surface: a Mac page's Clean up is Home's
// sheet scoped to that Mac, never a bare question.
func TestAMacPagesCleanUpIsTheSheetOfThatMac(t *testing.T) {
	all := fleetOfTwo()
	m := Machine{Name: "pro", All: &all, Agents: sheetAgents(), Now: sheetNow, Asks: graded}
	for _, a := range overviewActions(m, all.Macs[0], page.ThisMac) {
		if a.ID != ActionCleanup {
			continue
		}
		if a.Confirm != nil || a.Sheet != page.SheetCleanup || a.Form.Title != "Clean up on This Mac" || len(fieldOf(a, "VMs").Options) == 0 {
			t.Errorf("This Mac's Clean up %+v", a)
		}
		return
	}
	t.Fatal("This Mac's page has no Clean up")
}
