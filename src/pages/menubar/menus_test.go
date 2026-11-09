package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/catalog"
	coremachine "github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
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
			words += "…"
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

func yesNo(string, coremachine.Kind) string { return LevelYesNo }

// invariant: SPEC A5: a running workspace's row on its Mac's page opens
// it, opens its terminal, then offers its own page's verbs grouped: Share,
// Stop…, Delete… last, in red.
func TestAWorkspaceRowsMenuOffersItsOwnPagesVerbs(t *testing.T) {
	p := macPageOf(t, miniMac, nil)
	got := rowMenu(t, p, "linux-3")
	want := []string{"linux-3 on mac-mini", "Open", "Open in Terminal", "-", "Copy [ssh command]", "Share [A port…, An account…]", "-", "Stop…", "-", "Delete… (red)"}
	if !slices.Equal(got, want) {
		t.Fatalf("linux-3's menu %q", got)
	}
}

// invariant: a stopped workspace's menu offers Start in place of Stop.
func TestAStoppedWorkspacesMenuOffersStart(t *testing.T) {
	p := macPageOf(t, miniMac, func(m *Machine) { m.All.Macs[1].VMs[0].State = model.Stopped })
	got := rowMenu(t, p, "linux-3")
	if slices.Contains(got, "Stop…") || !slices.Contains(got, "Start") {
		t.Fatalf("a stopped workspace's menu %q", got)
	}
}

// invariant: SPEC A5: another Mac's … menu holds Restart…, Stop
// everything… and Forget…; This Mac's has no Restart (the Apple menu owns
// it) and no Forget; an asleep Mac's only Wake and Forget….
func TestAMacsMenuOffersWhatThatMacTakes(t *testing.T) {
	if got := pageMenu(macPageOf(t, miniMac, nil)); !slices.Equal(got, []string{"mac-mini", "Open in Terminal", "-", "Restart…", "Stop everything… (red)", "-", "Forget… (red)"}) {
		t.Fatalf("mac-mini's menu %q", got)
	}
	if got := pageMenu(macPageOf(t, proMac, nil)); !slices.Equal(got, []string{"This Mac", "Open in Terminal", "-", "Stop everything… (red)"}) {
		t.Fatalf("This Mac's menu %q", got)
	}
	if got := pageMenu(macPageOf(t, airMac, nil)); !slices.Equal(got, []string{"macbook-air", "Wake macbook-air", "-", "Forget… (red)"}) {
		t.Fatalf("an asleep Mac's menu %q", got)
	}
}

// invariant: a server's … menu says it is on its Mac, then Restart, Stop
// and Delete as its page asks them.
func TestAServersMenuNamesItsMacThenItsVerbs(t *testing.T) {
	got := pageMenu(macPageOf(t, "infra", nil))
	if !slices.Equal(got, []string{"infra on mac-mini", "Restart…", "Stop…", "-", "Delete… (red)"}) {
		t.Fatalf("infra's menu %q", got)
	}
}

// invariant: SPEC A5: the station menu: New VM…, Add a Mac…, Open
// station in Terminal; then the app's own Open at login and CPU in the
// menu bar; Quit station last.
func TestTheStationMenuListsItsVerbsThenTheAppsOwn(t *testing.T) {
	got := pageMenu(homeOf(t, newHeroWorld()))
	want := []string{"New VM…", "Add a Mac…", "Open station in Terminal", "-", "Open at login", "CPU in the menu bar", "-", "Quit station"}
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
	if got := rowMenu(t, p, heroMini); !slices.Equal(got, []string{"mac-mini", "Open", "New VM on mac-mini…", "Open in Terminal"}) {
		t.Fatalf("mac-mini's menu %q", got)
	}
	if got := rowMenu(t, p, "macbook-air"); !slices.Equal(got, []string{"macbook-air", "Open", "Wake macbook-air"}) {
		t.Fatalf("an asleep Mac's menu %q", got)
	}
}

// invariant: a server's row on Home asks Restart as its page does; down,
// its Restart is direct.
func TestAServerRowsMenuOnHomeRestartsAsItsPageDoes(t *testing.T) {
	w := newHeroWorld()
	w.r.Asks = yesNo
	if got := rowMenu(t, homeOf(t, w), "infra"); !slices.Contains(got, "Restart…") || !slices.Contains(got, "Delete… (red)") || got[0] != "infra on mac-mini" {
		t.Fatalf("infra's menu %q", got)
	}
	if got := rowMenu(t, homeOf(t, w.down()), "infra"); !slices.Contains(got, "Restart") {
		t.Fatalf("a down server's menu %q", got)
	}
}

// invariant: the Awake tile's menu keeps This Mac awake for a term or
// until the hour, the term in force picked; then Awake · All Macs.
func TestTheAwakeTilesMenuOffersItsTerms(t *testing.T) {
	w := newHeroWorld()
	w.r.Awake = &awake.Report{Now: heroNow, Engine: awake.EngineRunning, Mac: heroPro, ThisMac: true, Caps: coremachine.Capabilities{Engine: true},
		Session: &awake.Session{StartedAt: heroNow.Add(-time.Hour), For: 4 * time.Hour, Until: heroNow.Add(3 * time.Hour)}}
	p := homeOf(t, w)
	got := menuWords(p, actionOf(p, ActionMenu+":"+TileAwake).Submenu)
	if !slices.Equal(got[:3], []string{"Keep This Mac awake", "For 1 hr", "For 4 hr"}) || got[len(got)-1] != "Awake · All Macs" || !slices.Contains(got, "Let it sleep") {
		t.Fatalf("the Awake tile's menu %q", got)
	}
	if !actionOf(p, menuID(ActionKeep+"4h", TileAwake)).Picked {
		t.Fatalf("the 4 hr term in force is not picked")
	}
}

// invariant: SPEC A5: a copy lent to a VM: its account to the VM, Extend
// 4 hr, then Take back… in red.
func TestALoansMenuExtendsOrTakesItBack(t *testing.T) {
	all := accounts.AllMachines{Now: macNow, Macs: []accounts.MacAccounts{{Name: proMac, ThisMac: true,
		Shares: []accounts.Shared{{Account: catalog.ProviderGitHub, VM: "linux-1", At: macNow.Add(-time.Hour), Until: macNow.Add(3 * time.Hour)}}}}}
	p := Accounts(all, AccountsOf{Asks: yesNo}, page.Context{Machine: proMac})
	var lease string
	for _, s := range p.Sections {
		for _, it := range s.Items {
			if it.Row != nil && strings.HasSuffix(it.Row.ID, "/linux-1/github") {
				lease = it.Row.ID
			}
		}
	}
	if got := rowMenu(t, p, lease); !slices.Equal(got, []string{"GitHub → linux-1", "Extend 4 hr", "-", "Take back… (red)"}) {
		t.Fatalf("the loan's menu %q", got)
	}
}
