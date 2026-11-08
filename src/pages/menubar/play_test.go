package menubar

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
)

func apiPage(t *testing.T, state model.VMState) page.Page {
	t.Helper()
	fleet := &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true, Reachable: true, VMs: []model.VM{
		{Name: "api", Lifetime: model.Permanent, State: state},
	}}}}
	return MachinePage(Machine{Name: "api", All: fleet, Asks: func(string, machine.Kind) string { return LevelYesNo }})
}

func actionIDs(p page.Page) []string {
	out := make([]string, 0, len(p.Actions))
	for _, a := range p.Actions {
		out = append(out, a.ID)
	}
	return out
}

// invariant: a stopped workspace's main action is Start, and it offers no
// share: there is nothing to share while it is stopped.
func TestAStoppedWorkspaceStartsAndSharesNothing(t *testing.T) {
	stopped := actionIDs(apiPage(t, model.Stopped))
	if !slices.Contains(stopped, ActionStartVM) || slices.Contains(stopped, ActionStop) || slices.Contains(stopped, ActionShare) {
		t.Fatalf("stopped workspace offers %q", stopped)
	}
	running := actionIDs(apiPage(t, model.Running))
	if slices.Contains(running, ActionStartVM) || !slices.Contains(running, ActionStop) || !slices.Contains(running, ActionShare) {
		t.Fatalf("running workspace offers %q", running)
	}
	for _, a := range apiPage(t, model.Stopped).Actions {
		if a.ID == ActionStartVM && (a.Run.Call != CallStartVM || a.Run.In["VM"] != "api") {
			t.Fatalf("Start runs %+v", a.Run)
		}
	}
}

// invariant: Inside says what is there and what needs a look, in the
// vision's few words; never what station does not manage.
func TestInsideSaysWhatIsThereInFewWords(t *testing.T) {
	apps := page.List(page.Value{Kind: page.KindCount, N: 38, Unit: "ok", Label: "ok"}, page.Value{Kind: page.KindCount, N: 1, Unit: "not managed", Label: "unknown"},
		page.Value{Kind: page.KindCount, N: 62, Unit: "not installed", Label: "absent"}, page.Value{Kind: page.KindCount, N: 2, Unit: "outdated", Label: "attention"})
	if got := words(inside("apps", apps)); got != "38 installed | 2 need a look" {
		t.Fatalf("apps %q", got)
	}
	settings := page.List(page.Text("set by station"), page.Value{Kind: page.KindText, Text: "0 of 6", Label: "absent"})
	if got := words(inside("settings", settings)); got != "0 of 6 set" {
		t.Fatalf("settings %q", got)
	}
}

// invariant: an account lent to a workspace says what it may do and its
// time left, never its internal scope (`session:github`).
func TestASharedAccountSaysNoInternalScope(t *testing.T) {
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	fleet := &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true, Reachable: true, VMs: []model.VM{
		{Name: "linux-1", Lifetime: model.Throwaway, State: model.Running},
	}}}}
	report := &accounts.Report{Machines: []accounts.Machine{{Name: "linux-1", Accounts: []accounts.Account{
		{Provider: "github", Scopes: []string{"session:github", "read only"}, Shared: &accounts.Loan{Until: now.Add(4 * time.Hour)}},
	}}}}
	p := MachinePage(Machine{Name: "linux-1", All: fleet, Accounts: report, Now: now, Asks: func(string, machine.Kind) string { return LevelYesNo }})
	for _, s := range p.Sections {
		if s.ID != SectionShared {
			continue
		}
		if r := s.Items[0].Row; words(r.Detail) != "read only" || r.Left == nil || r.Left.N != int64((4*time.Hour).Seconds()) {
			t.Fatalf("shared account says %q, %+v left", words(r.Detail), r.Left)
		}
		return
	}
	t.Fatal("no shared section")
}

// invariant: Home lists a Mac's running workspaces before its stopped ones.
func TestHomeListsRunningWorkspacesFirst(t *testing.T) {
	fleet := &model.AllMachines{Macs: []model.MacView{{Name: "jterrazz-macbook-pro", ThisMac: true, Reachable: true, VMs: []model.VM{
		{Name: "api", Lifetime: model.Permanent, State: model.Stopped},
		{Name: "linux-1", Lifetime: model.Throwaway, State: model.Running},
	}}}}
	s := Tree(fleet, nil, func(model.MacView) page.Value { return page.None() })
	vms := s.Items[0].Group.Items[0].Group.Items
	names := make([]string, 0, len(vms))
	for _, vm := range vms {
		names = append(names, vm.Row.Label)
	}
	if !slices.Equal(names, []string{"linux-1", "api"}) {
		t.Fatalf("workspaces %q", names)
	}
}

// invariant: the Awake page names another Mac as Home does (`mac-mini`),
// and says a cool Mac's temperature is normal.
func TestAwakeNamesMacsAsHomeDoes(t *testing.T) {
	f := pageawake.AllMachines{Reports: []awake.Report{
		{Mac: "jterrazz-macbook-pro", ThisMac: true, Caps: machine.Capabilities{Thermal: true}},
		{Mac: "jterrazz-mac-mini", NeverSleeps: true},
	}}
	p := Awake(f, page.Context{})
	for _, s := range p.Sections {
		switch s.ID {
		case SectionOthers:
			if got := s.Items[0].Row.Short; got != "mac-mini" {
				t.Fatalf("other Mac named %q", got)
			}
		case SectionGuards:
			if got := s.Items[0].Fact.Value.Text; got != "normal" {
				t.Fatalf("temperature %q", got)
			}
		}
	}
}

// invariant: the Accounts tile says which account needs a sign-in.
func TestTheAccountsTileSaysWhoNeedsSignIn(t *testing.T) {
	got := words(signIns(page.List(page.Value{Kind: page.KindText, Text: "6", Label: "ok"}, page.Value{Kind: page.KindText, Text: "registry", Label: "broken"})))
	if got != "6 | registry needs sign-in" {
		t.Fatalf("tile %q", got)
	}
}

func words(v page.Value) string {
	parts := v.Items
	if v.Kind != page.KindList {
		parts = []page.Value{v}
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		switch p.Kind {
		case page.KindCount:
			out = append(out, strconv.FormatInt(p.N, 10)+" "+p.Unit)
		default:
			out = append(out, p.Text)
		}
	}
	return strings.Join(out, " | ")
}

// invariant: a VM of another Mac is stopped and restarted by that Mac: its
// actions target the Mac that holds it, never the VM itself.
func TestAServerOfAnotherMacIsActedOnByItsMac(t *testing.T) {
	fleet := &model.AllMachines{Macs: []model.MacView{
		{Name: "jterrazz-macbook-pro", ThisMac: true, Reachable: true},
		{Name: "jterrazz-mac-mini", Reachable: true, VMs: []model.VM{{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running}}},
	}}
	p := MachinePage(Machine{Name: "infra", All: fleet, Asks: func(string, machine.Kind) string { return LevelYesNo }})
	for _, a := range p.Actions {
		if a.Run == nil || a.Run.Call == "" {
			continue
		}
		if a.Run.Target == nil || *a.Run.Target != (page.Target{Name: "jterrazz-mac-mini"}) {
			t.Fatalf("%s targets %+v, want mac-mini", a.ID, a.Run.Target)
		}
	}
}
