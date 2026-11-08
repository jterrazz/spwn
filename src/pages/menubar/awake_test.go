package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/page"
	pageawake "github.com/jterrazz/studio-station/src/pages/awake"
)

// invariant: AW-21, E1: This Mac's Awake offers Lid closed and On battery
// only where its capabilities have them: a desktop Mac keeps its terms and
// no switch, a laptop has both.
func TestAwakeOffersTheSwitchesThisMacHas(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	offered := func(caps machine.Capabilities) []string {
		r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Reading: awake.Reading{Source: "ac"}, Caps: caps}
		p := Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: []awake.Report{r}}, page.Context{Machine: "this"})
		ids := make([]string, 0, len(p.Actions))
		for _, a := range p.Actions {
			ids = append(ids, a.ID)
		}
		return ids
	}
	if got, want := offered(machine.Capabilities{Engine: true}), []string{"keep:1h", "keep:4h", "keep-until", "keep-on"}; !slices.Equal(got, want) {
		t.Fatalf("a desktop Mac offers %q, want %q", got, want)
	}
	if got, want := offered(machine.Capabilities{Engine: true, Lid: true, Battery: true}), []string{"keep:1h", "keep:4h", "keep-until", "keep-on", "lid", "battery"}; !slices.Equal(got, want) {
		t.Fatalf("a laptop offers %q, want %q", got, want)
	}
}

// invariant: Keep awake for… shows the term in force: the one the session
// was taken for, else Until; nothing while This Mac sleeps normally.
func TestKeepAwakeForPicksTheTermInForce(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	picked := func(s *awake.Session) []string {
		r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Reading: awake.Reading{Source: "ac"}, Caps: machine.Capabilities{Engine: true}, Session: s}
		p := Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: []awake.Report{r}}, page.Context{Machine: "this"})
		var ids []string
		for _, a := range p.Actions {
			if a.Picked {
				ids = append(ids, a.ID)
			}
		}
		return ids
	}
	if got := picked(&awake.Session{StartedAt: now.Add(-3 * time.Minute), Until: now.Add(57 * time.Minute)}); !slices.Equal(got, []string{"keep:1h"}) {
		t.Fatalf("a 1 h session picks %q, want keep:1h", got)
	}
	if got := picked(&awake.Session{StartedAt: now.Add(-20 * time.Minute), Until: now.Add(4 * time.Hour), For: 4 * time.Hour}); !slices.Equal(got, []string{"keep:4h"}) {
		t.Fatalf("a 4 h term asked again picks %q, want keep:4h", got)
	}
	if got := picked(&awake.Session{StartedAt: now, Until: now.Add(150 * time.Minute)}); !slices.Equal(got, []string{"keep-until"}) {
		t.Fatalf("a session until 13:00 picks %q, want keep-until", got)
	}
	if got := picked(nil); len(got) != 0 {
		t.Fatalf("no session picks %q, want none", got)
	}
	if got := picked(&awake.Session{StartedAt: now, Until: now.Add(24 * time.Hour), Run: "linux-1"}); len(got) != 0 {
		t.Fatalf("a keep-awake that follows a run picks %q, want none", got)
	}
	for _, d := range []time.Duration{time.Hour, 4 * time.Hour} {
		if got := picked(&awake.Session{StartedAt: now, Until: now.Add(d), Run: "linux-1"}); len(got) != 0 {
			t.Fatalf("a run held for %s picks %q, want none", d, got)
		}
	}
}

// invariant: Until's hours are sent as the hour itself, and a switch
// flipped while a keep-awake holds sends the end it holds until, never a
// time left counted when the page was read: both end where they said.
func TestAwakeSendsTheEndItHoldsUntil(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	until := time.Date(2026, 11, 3, 14, 30, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Reading: awake.Reading{Source: "ac"},
		Caps:    machine.Capabilities{Engine: true, Lid: true, Battery: true},
		Session: &awake.Session{StartedAt: now.Add(-time.Hour), Until: until, For: 4 * time.Hour}}
	p := Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: []awake.Report{r}}, page.Context{Machine: "this"})
	for _, a := range p.Actions {
		switch a.ID {
		case ActionUntil:
			f := a.Form.Fields[0]
			if f.Key != "Until" || f.Options[0].Value != `"2026-11-03T11:00:00Z"` || f.Options[0].Label != "11:00" {
				t.Fatalf("Until offers %s = %s (%s)", f.Key, f.Options[0].Value, f.Options[0].Label)
			}
		case ActionLid, ActionBattery:
			if a.Run.In["Until"] != `"2026-11-03T14:30:00Z"` || a.Run.In["For"] != "" {
				t.Fatalf("%s sends %v", a.ID, a.Run.In)
			}
		}
	}
}

func awakeOf(t *testing.T, reports []awake.Report, silent ...pageawake.Silent) page.Page {
	t.Helper()
	page.Strict(true)
	defer page.Strict(false)
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	return Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: reports, Silent: silent}, page.Context{Machine: "this"})
}

func notes(s *page.Section) []string {
	var out []string
	for _, it := range s.Items {
		if it.Note != nil {
			out = append(out, it.Note.Text)
		}
	}
	return out
}

// invariant: SPEC B5.14, A9 2-11: a keep-awake that follows a run picks no
// term and says the run it follows; the picker offers 1 hr, 4 hr and
// Until 18:00 in one unit.
func TestAwakeUntilARunSaysTheRunItFollows(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Caps: machine.Capabilities{Engine: true},
		Session: &awake.Session{StartedAt: now, Until: now.Add(24 * time.Hour), Run: "linux-1"}}
	p := awakeOf(t, []awake.Report{r})
	if got := notes(sectionOf(p, SectionThisMac)); !slices.Equal(got, []string{"Until linux-1 is done"}) {
		t.Fatalf("This Mac's card says %q", got)
	}
	var labels []string
	for _, a := range p.Actions {
		if strings.HasPrefix(a.ID, "keep") && a.ID != ActionKeepOn {
			labels = append(labels, a.Label)
		}
	}
	if want := []string{"1 hr", "4 hr", "Until 18:00"}; !slices.Equal(labels, want) {
		t.Fatalf("the picker offers %q, want %q", labels, want)
	}
}

func pickerOf(p page.Page) (labels, picked []string) {
	for _, a := range p.Actions {
		if strings.HasPrefix(a.ID, "keep") && a.ID != ActionKeepOn {
			labels = append(labels, a.Label)
			if a.Picked {
				picked = append(picked, a.Label)
			}
		}
	}
	return labels, picked
}

// invariant: SPEC A9 2-11: while a 4 hr term holds, the picker still
// reads 1 hr, 4 hr and Until 18:00, with 4 hr picked: Until offers the
// hour its form opens on, not the end of the term in force. Only an
// Until in force says the hour it holds until.
func TestAwakeUntilKeepsItsHourWhileATermHolds(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	held := func(s *awake.Session) page.Page {
		return awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Caps: machine.Capabilities{Engine: true}, Session: s}})
	}
	labels, picked := pickerOf(held(&awake.Session{StartedAt: now, Until: now.Add(4 * time.Hour), For: 4 * time.Hour}))
	if want := []string{"1 hr", "4 hr", "Until 18:00"}; !slices.Equal(labels, want) || !slices.Equal(picked, []string{"4 hr"}) {
		t.Fatalf("a held 4 hr term offers %q with %q picked, want %q with 4 hr", labels, picked, want)
	}
	labels, picked = pickerOf(held(&awake.Session{StartedAt: now, Until: now.Add(150 * time.Minute)}))
	if want := []string{"1 hr", "4 hr", "Until 13:00"}; !slices.Equal(labels, want) || !slices.Equal(picked, []string{"Until 13:00"}) {
		t.Fatalf("an Until 13:00 in force offers %q with %q picked, want %q", labels, picked, want)
	}
}

// invariant: SPEC A9 1-11: the pause line names only the running work,
// and the page has no footer.
func TestAwakePauseLineListsOnlyRunningWork(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Pauses: []string{"linux-1"}}
	p := awakeOf(t, []awake.Report{r})
	if got := notes(sectionOf(p, SectionPauses)); !slices.Equal(got, []string{"linux-1 pauses while This Mac sleeps."}) {
		t.Fatalf("the pause line says %q", got)
	}
	if p.Footer != nil {
		t.Fatalf("Awake has a footer: %+v", p.Footer)
	}
	two := awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true, Pauses: []string{"agent-7f", "linux-1"}}})
	if got := notes(sectionOf(two, SectionPauses)); !slices.Equal(got, []string{"agent-7f and linux-1 pause while This Mac sleeps."}) {
		t.Fatalf("two running workspaces say %q", got)
	}
	if s := sectionOf(awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true}}), SectionPauses); s != nil {
		t.Fatalf("nothing running still says %q", notes(s))
	}
}

// invariant: SPEC B5.14, B5.15: another Mac says its policy and what
// depends on it; one asleep reads asleep with when it was seen, dimmed and
// with no switch, never down.
func TestAwakeSaysEachMacsPolicyAndDependants(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	this := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", ThisMac: true}
	mini := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "mac-mini", NeverSleeps: true, NeverHeld: true, Servers: []string{"infra"}}
	p := awakeOf(t, []awake.Report{this, mini}, pageawake.Silent{Mac: "macbook-air", Why: "no answer", Asleep: true, Answered: now.Add(-2 * time.Hour)})
	row := sectionOf(p, "mac-mini").Items[0].Row
	if row.Label != "Never sleeps" || row.Detail.Text != "hosts infra, which pauses if it sleeps" || row.Verb != ActionChange+"mac-mini" {
		t.Fatalf("mac-mini reads %q, %q, verb %q", row.Label, row.Detail.Text, row.Verb)
	}
	if len(row.Cells) != 1 || row.Cells[0].Kind != page.KindOn {
		t.Fatalf("mac-mini's Never sleeps switch reads %+v, want on", row.Cells)
	}
	air := sectionOf(p, "macbook-air").Items[0].Row
	words := make([]string, 0, len(air.Detail.Items))
	for _, v := range air.Detail.Items {
		words = append(words, v.Text)
	}
	if air.Label != "May sleep" || air.Signal != page.SignalAsleep || air.Verb != "" || !slices.Equal(words, []string{"asleep", "seen 2 hr ago", "nothing depends on it"}) {
		t.Fatalf("the asleep Mac reads %q, %q (%s), verb %q", air.Label, words, air.Signal, air.Verb)
	}
}
