package menubar

import (
	"slices"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

var sheetNow = time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)

func fleetOfTwo() model.AllMachines {
	return model.AllMachines{Macs: []model.MacView{
		{Name: "pro", ThisMac: true, Reachable: true, VMs: []model.VM{
			{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running},
			{Name: "agent-7f", Lifetime: model.Ephemeral, State: model.Stopped, SizeGB: 12},
			{Name: "docs", Lifetime: model.Persistent, State: model.Running},
		}},
		{Name: "mini", Reachable: true, VMs: []model.VM{
			{Name: "old", Lifetime: model.Ephemeral, State: model.Stopped, SizeGB: 8},
			{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running},
		}},
		{Name: "air", Reachable: true, Asleep: true, VMs: []model.VM{{Name: "away", Lifetime: model.Ephemeral, State: model.Running}}},
	}}
}

func fieldOf(a page.Action, key string) page.Field {
	for _, f := range a.Form.Fields {
		if f.Key == key {
			return f
		}
	}
	return page.Field{}
}

// invariant: SPEC A5 Clean up across every Mac: the stopped ephemeral
// workspaces of each Mac deleted on that Mac, each checked; a persistent
// one is never offered; the button worded from what is checked, red; the
// note names the ephemeral workspaces that run, which stay.
func TestCleanUpDeletesEachMacsStoppedEphemerals(t *testing.T) {
	all := fleetOfTwo()
	a := cleanupSheet(ActionCleanup, all, cleanupsOf(all.Macs), false)
	calls := make([]string, 0, 1+len(a.Run.And))
	calls = append(calls, a.Run.Call+" "+a.Run.In["VMs"])
	for _, r := range a.Run.And {
		on := ""
		if r.Target != nil {
			on = " on " + r.Target.Name + "/" + r.Target.On
		}
		calls = append(calls, r.Call+" "+r.In["VMs"]+on)
	}
	if want := []string{`machines.cleanup ["agent-7f"]`, `machines.cleanup ["old"] on mini/`}; !slices.Equal(calls, want) {
		t.Fatalf("Clean up runs %q, want %q", calls, want)
	}
	vms := fieldOf(a, "VMs")
	if vms.Default != "agent-7f,old" || vms.Options[1].Note != "mini{sep}stopped{sep}8 GB disk" {
		t.Errorf("the ephemeral workspaces %+v", vms)
	}
	if note := fieldOf(a, "stays").Note; note != "linux-1 and away are still running, so they stay." {
		t.Errorf("the note says %q", note)
	}
	words := make([]string, 0, len(a.Form.Primary))
	for _, p := range a.Form.Primary {
		words = append(words, p.Label)
		if !p.Danger {
			t.Errorf("%q is not red", p.Label)
		}
	}
	if want := []string{"Delete 2 ephemeral workspaces", "Delete 1 ephemeral workspace"}; !slices.Equal(words, want) {
		t.Errorf("the button says %q, want %q", words, want)
	}
}

// invariant: SPEC A5 Clean up on one Mac: its title names the Mac, its
// ephemeral workspaces say no Mac, and it deletes on that Mac alone.
func TestCleanUpOnAMacStaysOnThatMac(t *testing.T) {
	all := fleetOfTwo()
	a := cleanupSheet(ActionCleanup+":mini", all, cleanupsOf(all.Macs[1:2]), true)
	if a.Form.Title != "Clean up on mini" || a.Run.Target.Name != "mini" || len(a.Run.And) != 0 || fieldOf(a, "VMs").Options[0].Note != "stopped{sep}8 GB disk" {
		t.Errorf("Clean up on mini: %+v, runs %+v", a.Form, a.Run)
	}
}

// invariant: SPEC A5 Clean up: one running ephemeral workspace is said in
// the singular.
func TestCleanUpSaysOneRunningWorkspaceStays(t *testing.T) {
	if got := stays([]string{"linux-1"}); got != "linux-1 is still running, so it stays." {
		t.Errorf("the note says %q", got)
	}
}

// invariant: SPEC A5 Grant a password: one sheet per machine a password
// may go to, To swapping between them; This Mac's passwords as a pick
// list, the one that cannot go there disabled with the engine's line; the
// button names the password, and the call is grants.add.
func TestGrantAPasswordOffersEachMachineThePasswordsThatFit(t *testing.T) {
	to := runningWorkspaces(fleetOfTwo())
	if want := []workspace{{VM: "linux-1", Mac: "pro"}, {VM: "docs", Mac: "pro"}}; !slices.Equal(to, want) {
		t.Fatalf("the workspaces %v, want %v", to, want)
	}
	deploy := accounts.VaultEntry{ID: "pw-1", Label: "deploy token", Program: programFor(t, catalog.ProviderGitHub)}
	mail := accounts.VaultEntry{ID: "pw-2", Label: "mail token", Program: programFor(t, catalog.ProviderClaude)}
	target := func(vm string, fits accounts.TargetFit, line string) accounts.Target {
		return accounts.Target{To: vm, Fit: fits, Line: line}
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
	if first.Sheet != page.SheetGrant || first.Run.Call != CallGrant || first.Run.In["To"] != "linux-1" || first.Form.Fields[0].Default != "grant:linux-1" {
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

// invariant: one gesture, one surface: a Mac page's Clean up is Home's
// sheet scoped to that Mac, never a bare question.
func TestAMacPagesCleanUpIsTheSheetOfThatMac(t *testing.T) {
	all := fleetOfTwo()
	m := Machine{Name: "pro", All: &all, Now: sheetNow, Asks: graded}
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
