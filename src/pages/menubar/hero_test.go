package menubar

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
	model "github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	"github.com/jterrazz/studio-station/src/ports"
)

const (
	heroPro  = "jterrazz-macbook-pro"
	heroMini = "jterrazz-mac-mini"
)

var heroNow = time.Date(2026, 11, 7, 14, 30, 0, 0, time.UTC)

type heroWorld struct {
	r Reading
}

func newHeroWorld() *heroWorld {
	infra := model.VM{Name: "infra", Lifetime: model.AlwaysOn, State: model.Running, BootAt: heroNow.Add(-12 * 24 * time.Hour)}
	linux1 := model.VM{Name: "linux-1", Lifetime: model.Throwaway, State: model.Running, Agent: &model.AgentView{State: "working"}}
	return &heroWorld{r: Reading{Now: heroNow, Zone: time.UTC, Ports: &ports.AllMachines{}, Accounts: &accounts.AllMachines{Now: heroNow},
		All: &model.AllMachines{Read: heroNow, Macs: []model.MacView{
			{Name: heroPro, ThisMac: true, Reachable: true, Answered: heroNow, Hardware: hardware(48), VMs: []model.VM{linux1}},
			{Name: heroMini, Reachable: true, Answered: heroNow, Hardware: hardware(64), VMs: []model.VM{infra}},
		}}}}
}

func (w *heroWorld) mini() *model.MacView { return &w.r.All.Macs[1] }

func (w *heroWorld) waiting(name, ask string, since time.Duration) *heroWorld {
	w.mini().VMs = append(w.mini().VMs, model.VM{Name: name, Lifetime: model.Permanent, State: model.Running,
		Agent: &model.AgentView{State: "waiting", Ask: ask, Since: heroNow.Add(-since)}})
	return w
}

func (w *heroWorld) lease(vm string, left time.Duration) *heroWorld {
	w.r.Accounts.Macs = append(w.r.Accounts.Macs, accounts.MacAccounts{Name: heroPro, ThisMac: true,
		Shares: []accounts.Shared{{Account: catalog.ProviderClaude, VM: vm, At: heroNow.Add(left - 12*time.Hour), Until: heroNow.Add(left)}}})
	return w
}

func (w *heroWorld) down() *heroWorld {
	infra := &w.mini().VMs[0]
	infra.State, infra.Cause, infra.BootAt = model.Stopped, "out of memory", time.Time{}
	infra.Restarts = model.Restarts{Looping: true}
	infra.CheckStatus = model.CheckStatus{State: model.CheckFailing, Since: heroNow.Add(-2 * time.Minute)}
	return w
}

func (w *heroWorld) exposed() *heroWorld {
	w.r.Ports.Macs = append(w.r.Ports.Macs, ports.MacPorts{Name: heroPro, ThisMac: true, Ports: []ports.Port{
		{Machine: machine.Ref{Name: heroPro}, Port: 5000, Process: "ControlCenter", Reach: ports.Outside, Answer: ports.Answers},
	}})
	return w
}

func hardware(gigabytes int64) machine.Hardware {
	return machine.Hardware{Memory: gigabytes << 30, CPUs: 12}
}

type heroSaid struct {
	title string
	facts string
	verb  string
	rows  []string
	link  string
}

func said(t *testing.T, w *heroWorld) (heroSaid, page.Page) {
	t.Helper()
	p := Page(w.r, page.Context{Machine: heroPro, Kind: "computer"})
	if len(p.Sections) == 0 || p.Sections[0].ID != SectionHero {
		t.Fatalf("the page leads with %q, not the hero", p.Sections[0].ID)
	}
	var out heroSaid
	for i, it := range p.Sections[0].Items {
		switch {
		case i == 0 && it.Row != nil:
			out.title, out.facts = it.Row.Label, heroWords(it.Row.Detail)
			if it.Row.Verb != "" {
				out.verb = verbLabel(p, it.Row.Verb)
			}
		case it.Row != nil:
			out.rows = append(out.rows, it.Row.Label+" | "+heroWords(it.Row.Detail)+" ["+verbLabel(p, it.Row.Verb)+"]")
		case it.Link != nil:
			out.link = strings.TrimSuffix(it.Link.Label+" | "+heroWords(it.Link.Detail), " | ")
		}
	}
	return out, p
}

func heroWords(v page.Value) string {
	if v.Kind == page.KindList {
		parts := make([]string, 0, len(v.Items))
		for _, it := range v.Items {
			w := it.Text
			if it.Pri > 0 {
				w += "^" + strconv.Itoa(it.Pri)
			}
			parts = append(parts, w)
		}
		return strings.Join(parts, " | ")
	}
	return v.Text
}

func verbLabel(p page.Page, id string) string {
	for _, a := range p.Actions {
		if a.ID == id {
			return a.Label
		}
	}
	return "missing " + id
}

func equal(t *testing.T, got, want heroSaid) {
	t.Helper()
	if got.title != want.title || got.facts != want.facts || got.verb != want.verb || got.link != want.link || strings.Join(got.rows, "\n") != strings.Join(want.rows, "\n") {
		t.Fatalf("hero\n%+v\nwant\n%+v", got, want)
	}
}

// invariant: SPEC A4 Home 1: several needs: the count is the title and the
// badge's number, the three most urgent rows follow with their verbs, then
// one link to the Needs you page for the rest.
func TestSeveralNeedsAreCountedThenTheThreeMostUrgentAndALinkToTheRest(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", 6*time.Minute).lease("linux-1", 23*time.Minute)
	got, p := said(t, w)
	equal(t, got, heroSaid{title: "2 things need you", rows: []string{"review-bot | PR #42 is ready [Review]", "linux-1 | Claude ends in 23 min [Extend]"}})
	if p.Badge.Needs != 2 {
		t.Fatalf("the badge says %d, the hero 2", p.Badge.Needs)
	}
	w.waiting("linux-3", "asks which store", 2*time.Minute).waiting("api-bot", "PR #7 is ready", time.Minute).lease("linux-1", 9*time.Minute)
	got, _ = said(t, w)
	if got.title != "5 things need you" || len(got.rows) != 3 || got.link != "2 more need you" {
		t.Fatalf("five needs: %+v", got)
	}
}

// invariant: one need is the hero itself: its words as the title, its facts
// (the last dropped first), its one verb.
func TestOneNeedIsTheHeroWithItsFactsAndItsVerb(t *testing.T) {
	got, _ := said(t, newHeroWorld().waiting("review-bot", "PR #42 is ready", 6*time.Minute))
	equal(t, got, heroSaid{title: "review-bot is waiting", facts: "PR #42 is ready | 6 min^1", verb: "Review"})
	got, _ = said(t, newHeroWorld().lease("linux-1", 23*time.Minute))
	equal(t, got, heroSaid{title: "Claude on linux-1 ends in 23 min", facts: "shared from This Mac | ends 14:53^1", verb: "Extend"})
}

// invariant: a red need carries the hero with its fix; the others fold into
// one line naming each subject once, or keep their row when there is one;
// two reds are never merged, the first ranked carries the hero.
func TestARedNeedCarriesTheHeroAndTheOthersFoldUnderIt(t *testing.T) {
	got, _ := said(t, newHeroWorld().down().waiting("review-bot", "PR #42 is ready", 6*time.Minute).lease("linux-1", 23*time.Minute))
	equal(t, got, heroSaid{title: "infra is down", facts: "out of memory | since 14:28 | on mac-mini^1", verb: "Restart", link: "2 more need you | review-bot, linux-1"})
	got, _ = said(t, newHeroWorld().exposed().waiting("review-bot", "PR #42 is ready", 6*time.Minute))
	equal(t, got, heroSaid{title: "Port 5000 is exposed", facts: "This Mac | AirPlay Receiver^1", verb: "Review", rows: []string{"review-bot | PR #42 is ready [Review]"}})
	got, _ = said(t, newHeroWorld().down().exposed().waiting("review-bot", "PR #42 is ready", 6*time.Minute))
	equal(t, got, heroSaid{title: "infra is down", facts: "out of memory | since 14:28 | on mac-mini^1", verb: "Restart", link: "2 more need you | This Mac, review-bot"})
}

// invariant: a server restarting carries the hero while it boots, its step
// counted as it runs, and the needs fold under it as under a red.
func TestAServerRestartingCarriesTheHeroWithItsStep(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", 6*time.Minute).lease("linux-1", 23*time.Minute)
	w.mini().VMs[0].Restart = &model.RestartWalk{Step: 2, At: heroNow}
	got, _ := said(t, w)
	equal(t, got, heroSaid{title: "Restarting infra", facts: "on mac-mini^2 | has an address | 2 of 4", link: "2 more need you | review-bot, linux-1"})
}

// invariant: a restart begins while the server is still down: from its
// first step it is the restart, never a need folded under it nor counted.
func TestAServerRestartingFromDownIsNeitherANeedNorCounted(t *testing.T) {
	w := newHeroWorld().down().waiting("review-bot", "PR #42 is ready", 6*time.Minute).lease("linux-1", 23*time.Minute)
	w.mini().VMs[0].Restart = &model.RestartWalk{Step: 1, At: heroNow}
	got, p := said(t, w)
	if got.title != "Restarting infra" || got.link != "2 more need you | review-bot, linux-1" {
		t.Fatalf("hero %+v", got)
	}
	if b := p.Badge; b == nil || b.Needs != 2 || !b.Restarting || b.Down {
		t.Fatalf("badge %+v", p.Badge)
	}
}

// invariant: nothing needs you: the fleet's facts, the server's uptime
// dropped first, then the agents at work, then whether every Mac is awake.
func TestAFineFleetSaysItsFactsInOrderOfWhatGoesFirst(t *testing.T) {
	got, _ := said(t, newHeroWorld())
	equal(t, got, heroSaid{title: "All machines are fine", facts: "infra healthy | up 12 days^3 | all 2 Macs answer^1 | 1 agent working^2"})
}

// invariant: an empty fleet has no hero but the empty state, with the free
// memory of This Mac and New VM, and no + in the header.
func TestAnEmptyFleetIsTheEmptyStateWithNewVM(t *testing.T) {
	w := newHeroWorld()
	w.r.All.Macs = w.r.All.Macs[:1]
	w.r.All.Macs[0].VMs = nil
	p := Page(w.r, page.Context{Machine: heroPro, Kind: "computer"})
	e := p.Sections[0].Items[0].Empty
	if e == nil || e.Text != "No VMs yet" || e.Sub != "44 GB free on This Mac" || len(e.Actions) != 1 || e.Actions[0] != ActionNewVM {
		t.Fatalf("empty state %+v", e)
	}
	if len(p.Header.Buttons) != 0 {
		t.Fatalf("the empty fleet's header shows %v", p.Header.Buttons)
	}
}

// invariant: SPEC B5.15: a Mac silent for more than 30 s is checked: the
// header says so, its needs are drawn stale, an asleep Mac is not checked.
func TestAMacStillAnsweringIsCheckedAndItsNeedsDrawnStale(t *testing.T) {
	w := newHeroWorld().waiting("review-bot", "PR #42 is ready", 6*time.Minute).lease("linux-1", 23*time.Minute)
	w.mini().Reachable, w.mini().Answered = false, heroNow.Add(-40*time.Second)
	_, p := said(t, w)
	if p.Header.Stale == nil || p.Header.Stale.Text != "checking mac-mini" {
		t.Fatalf("header %+v", p.Header)
	}
	signals := map[string]page.Signal{}
	for _, it := range p.Sections[0].Items[1:] {
		signals[it.Row.Label] = it.Row.Signal
	}
	if signals["review-bot"] != page.SignalStale || signals["linux-1"] != "" {
		t.Fatalf("signals %v", signals)
	}
	w.mini().Asleep = true
	if _, p = said(t, w); p.Header.Stale != nil {
		t.Fatalf("an asleep Mac is checked: %v", p.Header.Stale)
	}
}
