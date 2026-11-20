package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

const (
	needsPro  = "macbook-pro"
	needsMini = "mac-mini"
)

var needsNow = time.Date(2026, 11, 7, 14, 10, 0, 0, time.UTC)

// value: the prototype's fleet on its worst day: infra down on mac-mini,
// This Mac's AirPlay Receiver reachable from outside, and This Mac's
// upkeep.
func needsFleet() NeedsReading {
	infra := model.VM{Name: "infra", Mac: needsMini, Lifetime: model.AlwaysOn, State: model.Stopped, Want: "running", Cause: "out of memory",
		Restarts: model.Restarts{Count: 5, Streak: 5, Looping: true}, CheckStatus: model.CheckStatus{State: model.CheckFailing, Since: needsNow.Add(-8 * time.Minute)}}
	all := model.AllMachines{Macs: []model.MacView{
		{Name: needsPro, ThisMac: true, Reachable: true, VMs: []model.VM{
			{Name: "linux-1", Mac: needsPro, Lifetime: model.Ephemeral, State: model.Running},
		}},
		{Name: needsMini, Reachable: true, VMs: []model.VM{infra,
			{Name: "linux-3", Mac: needsMini, Lifetime: model.Ephemeral, State: model.Running},
		}},
	}}
	standUnsent(&all)
	exposed := ports.AllMachines{Macs: []ports.MacPorts{{Name: needsPro, ThisMac: true, Reachable: true, Ports: []ports.Port{
		{Machine: machine.Ref{Name: needsPro}, Port: 5000, Process: "ControlCenter", Reach: ports.Outside},
		{Machine: machine.Ref{Name: "linux-1", On: needsPro}, Port: 5173, Process: "node", Reach: ports.Tailnet},
	}}}}
	home := overview.Summary{Head: overview.Head{Name: needsPro, Kind: machine.Computer, ThisMac: true}, Needs: []overview.Need{
		{Section: section.Apps, Subject: "tool-a", Text: "2 tool-a updates waiting", Call: CallUpdate},
		{Section: section.Accounts, Subject: "wrangler", Broken: true, Text: "signed out"},
		{Section: section.Background, Subject: "com.example.socket", Broken: true, Text: "com.example.socket is failing, last exit 78", Command: "station background"},
		{Section: section.Apps, Subject: "tool-b", Text: "2 tool-b updates waiting", Call: CallUpdate},
		{Section: section.Settings, Subject: "firewall", Text: "Firewall off", Path: "System Settings > Network > Firewall"},
	}}
	mini := overview.Summary{Head: overview.Head{Name: needsMini, Kind: machine.Computer}}
	return NeedsReading{Now: needsNow, Zone: time.UTC, All: &all, Ports: &exposed, Overviews: []overview.Summary{home, mini}}
}

// invariant: SPEC B5.7: every need is red, a server down first (two reds
// never merged); each with its verb and its three wordings: the hero's,
// the row's and Find's.
func TestTheFleetsNeedsComeInThePrototypesOrderWithTheirVerbsAndThreeWordings(t *testing.T) {
	needs, _ := needsOf(needsFleet(), needsPro)

	type said struct{ hero, subject, what, find, verb string }
	got := make([]said, 0, len(needs))
	for _, n := range needs {
		got = append(got, said{n.Hero, n.Subject, n.What, strings.Join(n.Find, " | "), n.Verb.Label})
	}
	want := []said{
		{"infra is down", "infra", "down since 14:02", "Restart infra", "Restart"},
		{"Port 5000 is exposed", "Port 5000", catalog.ReachableOutside, "Port 5000 is exposed", "Review"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("the needs say\n%v\nwant\n%v", got, want)
	}
	if facts := texts(needs[0].Facts); !slices.Equal(facts, []string{"out of memory", "since 14:02", "on mac-mini"}) {
		t.Errorf("infra's facts %q", facts)
	}
	if facts := texts(needs[1].Facts); !slices.Equal(facts, []string{"This Mac", "AirPlay Receiver"}) {
		t.Errorf("the exposed port's facts %q", facts)
	}
	if run := needs[0].Verb.Run; run == nil || run.Call != CallRestartVM || run.Target == nil || run.Target.Name != needsMini {
		t.Errorf("Restart infra runs %+v, want machines.restart on mac-mini", run)
	}
}

// invariant: SPEC B5.7, B5.18: upkeep is a list apart, never a need: a
// security setting first, then what fails, then sign-ins, then the updates
// as one item; each with its verb and its detail.
func TestUpkeepListsTheFirewallThenTheFailingItemThenTheSignInThenTheUpdatesAsOne(t *testing.T) {
	_, upkeep := needsOf(needsFleet(), needsPro)

	type said struct{ hero, subject, what, verb, detail string }
	got := make([]said, 0, len(upkeep))
	for _, n := range upkeep {
		got = append(got, said{n.Hero, n.Subject, n.What, n.Verb.Label, strings.Join(n.Detail, " | ")})
	}
	want := []said{
		{"Firewall off on This Mac", "This Mac", "firewall off", "How to", "Machines never changes it for you"},
		{"A login item failed on This Mac", "This Mac", "1 login item failed", "Show", "com.example.socket is failing, last exit 78"},
		{"Cloudflare needs sign-in on This Mac", "This Mac", "Cloudflare needs sign-in", "Sign in", "signed out"},
		{"4 updates to install", "This Mac", "4 updates", "Update", "2 tool-a updates waiting | 2 tool-b updates waiting"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("upkeep says\n%v\nwant\n%v", got, want)
	}
	if upkeep[0].Verb.Sheet != page.SheetHint+"firewall" {
		t.Errorf("How to opens %q, want the firewall's hint", upkeep[0].Verb.Sheet)
	}
	if upkeep[1].Verb.Run != nil || upkeep[1].Nav != machinePage(needsPro) {
		t.Errorf("Show runs %+v and opens %q, want no call and This Mac's page", upkeep[1].Verb.Run, upkeep[1].Nav)
	}
	if hud := upkeep[3].Verb.Hud; hud == nil || !slices.Equal(texts(hud.Items), []string{"Everything is up to date"}) {
		t.Errorf("Update says %+v once it ran", hud)
	}
}

// invariant: SPEC A4: menubar.needs holds every need then the upkeep;
// menubar.needs:<Mac> those of the Mac and its VMs, its own in its own
// words; with nothing, one empty card. Each page passes page.Check.
func TestTheNeedsPageListsEveryNeedThenUpkeepAndAMacsPageOnlyItsOwn(t *testing.T) {
	here := page.Context{Machine: needsPro, Kind: "computer"}
	rows := func(p page.Page, id string) []string {
		var out []string
		for _, s := range p.Sections {
			if s.ID != id {
				continue
			}
			for _, it := range s.Items {
				switch {
				case it.Row != nil:
					detail := it.Row.Detail.Text
					if it.Row.Detail.Kind == page.KindList {
						detail = strings.Join(texts(it.Row.Detail.Items), " | ")
					}
					out = append(out, it.Row.Label+" | "+detail+" ["+it.Row.Verb+"]")
				case it.Empty != nil:
					out = append(out, it.Empty.Text+" | "+it.Empty.Sub)
				}
			}
		}
		return out
	}
	all := NeedsPage(needsFleet(), "", here)
	mini := NeedsPage(needsFleet(), needsMini, here)
	pro := NeedsPage(needsFleet(), needsPro, here)
	calm := needsFleet()
	calm.All, calm.Ports, calm.Overviews = &model.AllMachines{}, nil, nil
	none := NeedsPage(calm, "", here)
	for _, p := range []page.Page{all, mini, pro, none} {
		if err := page.Check(p); err != nil {
			t.Errorf("%s: %v", p.ID, err)
		}
	}
	if got := len(rows(all, SectionNeeds)); got != 2 {
		t.Errorf("menubar.needs lists %d needs, want 2", got)
	}
	if got := len(rows(all, SectionUpkeep)); got != 4 {
		t.Errorf("menubar.needs lists %d upkeep items, want 4", got)
	}
	if got, want := rows(mini, SectionNeeds), []string{"infra | down since 14:02 [down:infra]"}; !slices.Equal(got, want) {
		t.Errorf("mac-mini's needs %q, want %q", got, want)
	}
	if got := rows(pro, SectionUpkeep); len(got) == 0 || got[0] != "Firewall off | Machines never changes it for you [upkeep:"+needsPro+"/settings/4]" {
		t.Errorf("This Mac's upkeep %q: its own rows say what, then the detail", got)
	}
	if got, want := rows(none, SectionNeeds), []string{"Nothing needs you | "}; !slices.Equal(got, want) {
		t.Errorf("a calm fleet's needs %q, want %q", got, want)
	}
	if NeedsPage(NeedsReading{}, "", here).State != page.Loading {
		t.Error("a needs page read from nothing is not its frame")
	}
}

// invariant: SPEC B5.7, B5.18: a failing login item is shown from the
// fleet's list, which opens its Mac; only the Mac's own page copies its fix.
func TestAFailingLoginItemIsShownFromTheFleetAndItsFixCopiedOnItsMac(t *testing.T) {
	here := page.Context{Machine: needsPro, Kind: "computer"}
	verb := func(p page.Page) page.Action {
		for _, a := range p.Actions {
			if strings.Contains(a.ID, "/background/") {
				return a
			}
		}
		t.Fatalf("%s carries no verb for the login item", p.ID)
		return page.Action{}
	}
	all := verb(NeedsPage(needsFleet(), "", here))
	if all.Label != VerbShow || all.Run != nil {
		t.Errorf("menubar.needs says %q and runs %+v, want Show and no call", all.Label, all.Run)
	}
	own := verb(NeedsPage(needsFleet(), needsPro, here))
	if own.Label != VerbCopyFix || own.Run == nil || own.Run.Copy != "station background" {
		t.Errorf("This Mac's page says %q and runs %+v, want Copy fix of station background", own.Label, own.Run)
	}
	if own.Hud == nil || !slices.Equal(texts(own.Hud.Items), []string{"Copied station background"}) {
		t.Errorf("Copy fix says %+v once it ran", own.Hud)
	}
}

func TestUpkeepIsNeverCountedAmongWhatNeedsYou(t *testing.T) {
	p := NeedsPage(needsFleet(), "", page.Context{Machine: needsPro, Kind: "computer"})
	for _, s := range p.Sections {
		switch s.ID {
		case SectionNeeds:
			if len(s.Counts) == 0 {
				t.Error("Needs you carries no count of its needs")
			}
		case SectionUpkeep:
			if !s.Uncounted || len(s.Counts) > 0 {
				t.Errorf("Upkeep is counted: uncounted %v, counts %v", s.Uncounted, s.Counts)
			}
		}
	}
}

func TestARefusedFileOfTheOwnerIsUpkeepNamingItsFileAndLine(t *testing.T) {
	r := needsFleet()
	r.Overviews[0].Needs = []overview.Need{{Machine: needsPro, Section: section.Overview, Broken: true, Severity: finding.Broken, Subject: overview.RefusedSubject,
		Text: "~/.config/station/catalog/apps/jq.yaml:3: unknown key", Path: "~/.config/station/catalog/apps/jq.yaml"}}
	_, upkeep := needsOf(r, needsPro)
	if len(upkeep) != 1 {
		t.Fatalf("upkeep %d items, want the refused file alone", len(upkeep))
	}
	n := upkeep[0]
	if n.Hero != "A file of yours is refused on This Mac" || n.Verb.Label != VerbShow ||
		!slices.Equal(n.Detail, []string{"~/.config/station/catalog/apps/jq.yaml:3: unknown key", "fix ~/.config/station/catalog/apps/jq.yaml: Machines reads the built-in packages meanwhile"}) {
		t.Errorf("the refused file says %q, %q, %q", n.Hero, n.Verb.Label, n.Detail)
	}
}
