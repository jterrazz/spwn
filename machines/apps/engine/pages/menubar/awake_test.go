package menubar

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/page"
	pageawake "github.com/jterrazz/studio-station/apps/engine/pages/awake"
)

// invariant: AW-21, E1: This Mac's Awake offers Lid closed and On battery
// only where its capabilities have them: a desktop Mac keeps its terms and
// no switch, a laptop has both.
func TestAwakeOffersTheSwitchesThisMacHas(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	offered := func(caps machine.Capabilities) []string {
		r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", Reading: awake.Reading{Source: "ac"}, Caps: caps}
		p := Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: []awake.Report{r}, This: "this"}, page.Context{Machine: "this"})
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

// invariant: This Mac's switches read as the board words them: "With the
// lid closed" and "On battery", under the Keep awake row.
func TestThisMacsSwitchesReadAsTheBoardWordsThem(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	p := awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", Caps: machine.Capabilities{Engine: true, Lid: true, Battery: true}}})
	items := sectionOf(p, SectionThisMac).Items[1:]
	got := make([]string, 0, len(items))
	for _, it := range items {
		got = append(got, it.Row.Label)
	}
	if want := []string{"With the lid closed", "On battery"}; !slices.Equal(got, want) {
		t.Fatalf("This Mac's switches read %q, want %q", got, want)
	}
}

// invariant: a Mac kept awake draws its sun monochrome, as its moon: the
// switch beside it says it is on, the glyph never turns orange.
func TestAKeptMacsSunIsMonochrome(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	p := awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", Caps: machine.Capabilities{Engine: true},
		Session: &awake.Session{StartedAt: now, Until: now.Add(time.Hour), For: time.Hour}}})
	if g := sectionOf(p, SectionThisMac).Items[0].Row.Glyph; g.Kind != "symbol:sun.max" || g.Signal != page.SignalQuiet {
		t.Fatalf("a kept Mac's sun is %+v, want sun.max quiet", g)
	}
}

// invariant: Keep awake for… shows the term in force: the one the session
// was taken for, else Until; nothing while This Mac sleeps normally.
func TestKeepAwakeForPicksTheTermInForce(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	picked := func(s *awake.Session) []string {
		r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", Reading: awake.Reading{Source: "ac"}, Caps: machine.Capabilities{Engine: true}, Session: s}
		p := Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: []awake.Report{r}, This: "this"}, page.Context{Machine: "this"})
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
}

// invariant: Until's hours are sent as the hour itself, and a switch
// flipped while a keep-awake holds sends the end it holds until, never a
// time left counted when the page was read: both end where they said.
func TestAwakeSendsTheEndItHoldsUntil(t *testing.T) {
	now := time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC)
	until := time.Date(2026, 11, 3, 14, 30, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", Reading: awake.Reading{Source: "ac"},
		Caps:    machine.Capabilities{Engine: true, Lid: true, Battery: true},
		Session: &awake.Session{StartedAt: now.Add(-time.Hour), Until: until, For: 4 * time.Hour}}
	p := Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: []awake.Report{r}, This: "this"}, page.Context{Machine: "this"})
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
	return Awake(pageawake.AllMachines{Now: now, Zone: time.UTC, Reports: reports, This: "this", Silent: silent}, page.Context{Machine: "this"})
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

// invariant: SPEC A9 2-11: while a 4 h term holds, the picker still
// reads 1 h, 4 h and Until 18:00, with 4 h picked: Until offers the
// hour its form opens on, not the end of the term in force. Only an
// Until in force says the hour it holds until.
func TestAwakeUntilKeepsItsHourWhileATermHolds(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	held := func(s *awake.Session) page.Page {
		return awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", Caps: machine.Capabilities{Engine: true}, Session: s}})
	}
	labels, picked := pickerOf(held(&awake.Session{StartedAt: now, Until: now.Add(4 * time.Hour), For: 4 * time.Hour}))
	if want := []string{"1 h", "4 h", "Until 18:00"}; !slices.Equal(labels, want) || !slices.Equal(picked, []string{"4 h"}) {
		t.Fatalf("a held 4 h term offers %q with %q picked, want %q with 4 h", labels, picked, want)
	}
	labels, picked = pickerOf(held(&awake.Session{StartedAt: now, Until: now.Add(150 * time.Minute)}))
	if want := []string{"1 h", "4 h", "Until 13:00"}; !slices.Equal(labels, want) || !slices.Equal(picked, []string{"Until 13:00"}) {
		t.Fatalf("an Until 13:00 in force offers %q with %q picked, want %q", labels, picked, want)
	}
}

// invariant: SPEC A9 1-11: the pause line names only the running work,
// and the page has no footer.
func TestAwakePauseLineListsOnlyRunningWork(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this", Pauses: []string{"linux-1"}}
	p := awakeOf(t, []awake.Report{r})
	if got := notes(sectionOf(p, SectionPauses)); !slices.Equal(got, []string{"linux-1 pauses while This Mac sleeps"}) {
		t.Fatalf("the pause line says %q", got)
	}
	if p.Footer != nil {
		t.Fatalf("Awake has a footer: %+v", p.Footer)
	}
	two := awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", Pauses: []string{"agent-7f", "linux-1"}}})
	if got := notes(sectionOf(two, SectionPauses)); !slices.Equal(got, []string{"agent-7f and linux-1 pause while This Mac sleeps"}) {
		t.Fatalf("two running workspaces say %q", got)
	}
	if s := sectionOf(awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this"}}), SectionPauses); s != nil {
		t.Fatalf("nothing running still says %q", notes(s))
	}
}

// invariant: SPEC B5.14, B5.15: another Mac says its policy and what
// depends on it; one asleep reads asleep with when it was seen, dimmed and
// with no switch, never down.
func TestAwakeSaysEachMacsPolicyAndDependants(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	this := awake.Report{Now: now, Engine: awake.EngineRunning, Mac: "this"}
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
	if air.Label != "Sleeps as usual" || air.Signal != page.SignalAsleep || air.Verb != "" || !slices.Equal(words, []string{"asleep", "seen 2 hr ago", "nothing depends on it"}) {
		t.Fatalf("the asleep Mac reads %q, %q (%s), verb %q", air.Label, words, air.Signal, air.Verb)
	}
}

// invariant: another Mac's card row is worded and marked by its Awake
// verdict, over every mode, engine state, policy hold, battery and guard:
// its label the verdict's sentence without the period, its mark awaiting
// while there is none, ok OK, warn Attention, plain Absent.
func TestAMacsCardRowIsWordedAndMarkedByItsVerdict(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	marks := map[string]page.Mark{"ok": page.OK, "warn": page.Attention, "plain": page.Absent}
	for _, engine := range []string{awake.EngineReading, awake.EngineStopped, awake.EngineAbsent, awake.EngineRunning} {
		for bits := range 1 << 6 {
			on := func(i int) bool { return bits&(1<<i) != 0 }
			r := awake.Report{Now: now, Engine: engine, Mac: "mini", Caps: machine.Capabilities{Engine: true, Battery: true}, NeverSleeps: on(0), NeverHeld: on(1),
				Reading: awake.Reading{HasBattery: true, Source: "ac", Percent: 80, LowPower: on(5)}}
			if on(2) {
				r.Session = &awake.Session{StartedAt: now, Until: now.Add(4 * time.Hour)}
			}
			if on(3) {
				r.LastEnd = &awake.End{At: now.Add(-time.Minute), Reason: awake.Thermal, Detail: "thermal serious"}
			}
			if on(4) {
				r.Reading.Source, r.Reading.Percent = "battery", 25
			}
			want, label := page.Awaiting, "Checking"
			if v := awake.VerdictOf(r); v != nil {
				want, label = marks[string(v.Tone)], strings.TrimSuffix(v.Sentence, ".")
				if r.EngineDown() || r.Mode() == awake.SleepsNormally {
					label = "Sleeps as usual"
				}
			}
			got := otherMac(r, time.UTC)
			if got.Mark != want {
				t.Errorf("%s %06b: %s; its verdict says %s", engine, bits, got.Mark, want)
			}
			if got.Label != label {
				t.Errorf("%s %06b: labeled %q; its verdict says %q", engine, bits, got.Label, label)
			}
		}
	}
}

// invariant: the page is headed Keep awake, and This Mac's row says the
// term in force ("Awake until 18:00", lid or not), else only Keep awake.
func TestAwakeHeadAndThisMacsRowSayTheTerm(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	held := awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this", Caps: machine.Capabilities{Engine: true, Lid: true},
		Session: &awake.Session{StartedAt: now, Until: now.Add(7*time.Hour + 30*time.Minute), Lid: true}}})
	if held.Header.Title != "Keep awake" || held.Sections[0].Caption[0].Text != "Which Mac may sleep, and until when" {
		t.Fatalf("the page is headed %q, %+v", held.Header.Title, held.Sections[0].Caption)
	}
	if got := sectionOf(held, SectionThisMac).Items[0].Row.Label; got != "Awake until 18:00" {
		t.Fatalf("This Mac holding until 18:00 reads %q", got)
	}
	free := awakeOf(t, []awake.Report{{Now: now, Engine: awake.EngineRunning, Mac: "this"}})
	if got := sectionOf(free, SectionThisMac).Items[0].Row.Label; got != "Keep awake" {
		t.Fatalf("This Mac with no term reads %q", got)
	}
}

// invariant: Until opens on the end awake.Choices offers, the next 18:00,
// which closes its hours when it comes after them.
func TestUntilOpensOnTheEndAwakeOffers(t *testing.T) {
	for name, c := range map[string]struct {
		now        time.Time
		def, label string
		n          int
	}{
		"morning": {time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC), `"2026-11-03T18:00:00Z"`, "18:00", untilHours},
		"evening": {time.Date(2026, 11, 3, 19, 30, 0, 0, time.UTC), `"2026-11-04T18:00:00Z"`, "18:00", untilHours + 1},
	} {
		f := untilField(c.now, time.UTC)
		last := f.Options[len(f.Options)-1]
		if f.Default != c.def || len(f.Options) != c.n || !slices.ContainsFunc(f.Options, func(o page.Option) bool { return o.Value == c.def && o.Label == c.label }) {
			t.Errorf("%s: opens on %s among %d hours, the last %s", name, f.Default, len(f.Options), last.Value)
		}
	}
}
