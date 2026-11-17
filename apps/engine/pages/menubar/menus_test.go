package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	coremachine "github.com/jterrazz/studio-station/apps/engine/core/machine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

// value: a menu as it reads: each entry's words, "-" between groups, a
// submenu's words after its own in brackets.
func menuWords(p page.Page, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == MenuSeparator {
			out = append(out, id)
			continue
		}
		a := actionOf(p, id)
		words := a.Label
		if a.Confirm != nil || a.Form != nil || a.Sheet != "" {
			words += "..."
		}
		if a.Destructive {
			words += " (red)"
		}
		if len(a.Submenu) > 0 {
			words += " [" + strings.Join(menuWords(p, a.Submenu), ", ") + "]"
		}
		out = append(out, words)
	}
	return out
}

func rowMenu(t *testing.T, p page.Page, row string) []string {
	t.Helper()
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if it.Row != nil && it.Row.ID == row {
				return menuWords(p, it.Row.Menu)
			}
		}
	}
	t.Fatalf("no row %s", row)
	return nil
}

func pageMenu(p page.Page) []string { return menuWords(p, actionOf(p, ActionMenu).Submenu) }

func yesNo(string, coremachine.Kind, ...string) string { return LevelYesNo }

// invariant: SPEC A5: a running workspace's row on its Mac's page opens
// it, opens its terminal, then offers its own page's verbs grouped: Share,
// Stop..., Delete... last, in red.
func TestAWorkspaceRowsMenuOffersItsOwnPagesVerbs(t *testing.T) {
	p := macPageOf(t, miniMac, nil)
	got := rowMenu(t, p, "linux-3")
	want := []string{"linux-3 on mac-mini", "Open", "Open in Terminal", "-", "Copy [SSH command]", "Share [A port...]", "-", "Stop...", "-", "Delete... (red)"}
	if !slices.Equal(got, want) {
		t.Fatalf("linux-3's menu %q", got)
	}
}

// invariant: a stopped workspace's menu offers Start in place of Stop.
func TestAStoppedWorkspacesMenuOffersStart(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) { m.All.Macs[1].VMs[0].State = model.Stopped })
	got := rowMenu(t, p, "linux-3")
	if slices.Contains(got, "Stop...") || !slices.Contains(got, "Start") {
		t.Fatalf("a stopped workspace's menu %q", got)
	}
}

// invariant: SPEC A5: another Mac's ... menu holds Restart..., Stop
// everything... and Forget...; This Mac's has no Restart (the Apple menu owns
// it) and no Forget, and Stop everything only while a workspace runs on it;
// an asleep Mac's only Wake and Forget....
func TestAMacsMenuOffersWhatThatMacTakes(t *testing.T) {
	if got := pageMenu(macPageOf(t, miniMac, nil)); !slices.Equal(got, []string{"mac-mini", "Open in Terminal", "-", "Restart...", "Stop everything... (red)", "-", "Forget... (red)"}) {
		t.Fatalf("mac-mini's menu %q", got)
	}
	working := func(m *Machine) {
		m.All.Macs[0].VMs = []model.VM{{Name: "linux-1", Lifetime: model.Ephemeral, State: model.Running}}
	}
	if got := pageMenu(macPageOf(t, proMac, working)); !slices.Equal(got, []string{"This Mac", "Open in Terminal", "-", "Stop everything... (red)"}) {
		t.Fatalf("This Mac's menu %q", got)
	}
	if got := pageMenu(macPageOf(t, proMac, nil)); !slices.Equal(got, []string{"This Mac", "Open in Terminal"}) {
		t.Fatalf("an idle This Mac's menu %q", got)
	}
	if got := pageMenu(macPageOf(t, airMac, nil)); !slices.Equal(got, []string{"macbook-air", "Wake macbook-air", "-", "Forget... (red)"}) {
		t.Fatalf("an asleep Mac's menu %q", got)
	}
}

// invariant: a server's ... menu says it is on its Mac, then Restart, Stop
// and Delete as its page asks them, Stop and Delete in red.
func TestAServersMenuNamesItsMacThenItsVerbs(t *testing.T) {
	got := pageMenu(macPageOf(t, "infra", nil))
	if !slices.Equal(got, []string{"infra on mac-mini", "Restart...", "Stop... (red)", "-", "Delete... (red)"}) {
		t.Fatalf("infra's menu %q", got)
	}
}

// invariant: SPEC A5: the station menu: New VM..., Add a Mac..., Open
// station in Terminal; then the app's own Open at login and CPU in the
// menu bar; Quit station last.
func TestTheStationMenuListsItsVerbsThenTheAppsOwn(t *testing.T) {
	got := pageMenu(homeOf(t, newHeroWorld()))
	want := []string{"New VM...", "Add a Mac...", "Open station in Terminal", "-", "Open at login", "CPU in the menu bar", "-", "Quit station"}
	if !slices.Equal(got, want) {
		t.Fatalf("the station menu %q", got)
	}
}

// invariant: a Mac's row on Home opens it, makes a VM there and opens
// its terminal; an asleep one only opens and wakes.
func TestAMacRowsMenuOnHome(t *testing.T) {
	w := newHeroWorld()
	w.r.All.Macs = append(w.r.All.Macs, model.MacView{Name: "macbook-air", Asleep: true, Answered: heroNow.Add(-2 * time.Hour), Laptop: true})
	p := homeOf(t, w)
	if got := rowMenu(t, p, heroMini); !slices.Equal(got, []string{"mac-mini", "Open", "New VM on mac-mini...", "Open in Terminal"}) {
		t.Fatalf("mac-mini's menu %q", got)
	}
	if got := rowMenu(t, p, "macbook-air"); !slices.Equal(got, []string{"macbook-air", "Open", "Wake macbook-air"}) {
		t.Fatalf("an asleep Mac's menu %q", got)
	}
}

// invariant: a server's row on Home asks Restart as its page does, down
// or up, since machines.restart asks a server's name typed; the item is
// red, as its typed question is.
func TestAServerRowsMenuOnHomeRestartsAsItsPageDoes(t *testing.T) {
	w := newHeroWorld()
	w.r.Asks = graded
	if got := rowMenu(t, homeOf(t, w), "infra"); !slices.Contains(got, "Restart... (red)") || !slices.Contains(got, "Delete... (red)") || got[0] != "infra on mac-mini" {
		t.Fatalf("infra's menu %q", got)
	}
	if got := rowMenu(t, homeOf(t, w.down()), "infra"); !slices.Contains(got, "Restart... (red)") {
		t.Fatalf("a down server's menu %q", got)
	}
}

// invariant: the Awake tile's menu keeps This Mac awake for a term or
// until the hour, the term in force picked; then Awake of all Macs.
func TestTheAwakeTilesMenuOffersItsTerms(t *testing.T) {
	w := newHeroWorld()
	w.r.Awake = &awake.Report{Now: heroNow, Engine: awake.EngineRunning, Mac: heroPro, Caps: coremachine.Capabilities{Engine: true},
		Session: &awake.Session{StartedAt: heroNow.Add(-time.Hour), For: 4 * time.Hour, Until: heroNow.Add(3 * time.Hour)}}
	p := homeOf(t, w)
	got := menuWords(p, actionOf(p, ActionMenu+":"+TileAwake).Submenu)
	if !slices.Equal(got[:3], []string{"Keep This Mac awake", "For 1 hr", "For 4 hr"}) || got[len(got)-1] != "Awake of all Macs" || !slices.Contains(got, "Let it sleep") {
		t.Fatalf("the Awake tile's menu %q", got)
	}
	if !actionOf(p, menuID(ActionKeep+"4h", TileAwake)).Picked {
		t.Fatalf("the 4 hr term in force is not picked")
	}
}

// invariant: SPEC A5: a grant to a VM: its program granted to the VM,
// then Remove... in red; only a grant to a VM of This Mac has one.
func TestAGrantsMenuRemovesIt(t *testing.T) {
	s := SignIns{Machines: []MachineSignIns{
		{Name: proMac, Mac: proMac, ThisMac: true},
		{Name: "linux-1", Mac: proMac, ThisMac: true, VM: true, Rows: []accounts.ProgramRow{grantedRow(t, catalog.ProviderGitHub, "pw-1")}},
		{Name: "linux-3", Mac: miniMac, VM: true, Rows: []accounts.ProgramRow{grantedRow(t, catalog.ProviderGitHub, "pw-1")}},
	}}
	p := Accounts(s, AccountsOf{Asks: yesNo}, page.Context{Machine: proMac})
	var rows []string
	for _, sec := range p.Sections {
		for _, it := range sec.Items {
			if it.Row != nil && strings.HasSuffix(it.Row.ID, "/"+string(programFor(t, catalog.ProviderGitHub))) {
				rows = append(rows, it.Row.ID)
			}
		}
	}
	if len(rows) != 2 {
		t.Fatalf("grant rows %q, want linux-1's and linux-3's", rows)
	}
	if got := rowMenu(t, p, rows[0]); !slices.Equal(got, []string{"GitHub granted to linux-1", "Remove... (red)"}) {
		t.Fatalf("the grant's menu %q", got)
	}
	if got := rowOf("linux-3", programFor(t, catalog.ProviderGitHub)); slices.ContainsFunc(p.Actions, func(a page.Action) bool { return a.ID == ActionRemoveGrant+got }) {
		t.Errorf("a grant to a VM of another Mac offers Remove")
	}
}
