package awake_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type awakeCase struct {
	engine                                                string
	never, held, session, lid, ended, near, lowPower, mac bool
}

func (c awakeCase) report() awake.Report {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	r := awake.Report{Now: now, Engine: c.engine, Caps: machine.Capabilities{Engine: true, Battery: true, Lid: true}, NeverSleeps: c.never, NeverHeld: c.held, MacNeverSleeps: c.mac,
		Reading: awake.Reading{Lid: "open", HasBattery: true, Source: "ac", Percent: 80, LowPower: c.lowPower}}
	if c.session {
		r.Session = &awake.Session{StartedAt: now.Add(-time.Hour), Until: time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC), Lid: c.lid}
	}
	if c.ended {
		r.LastEnd = &awake.End{At: now.Add(-2 * time.Minute), Reason: awake.Battery, Detail: "battery 18 %"}
	}
	if c.near {
		r.Reading.Source, r.Reading.Percent = "battery", 25
	}
	return r
}

func everyAwakeCase() []awakeCase {
	engines := []string{awake.EngineReading, awake.EngineStopped, awake.EngineAbsent, awake.EngineRunning}
	out := make([]awakeCase, 0, len(engines)<<8)
	for _, engine := range engines {
		for bits := range 1 << 8 {
			on := func(i int) bool { return bits&(1<<i) != 0 }
			out = append(out, awakeCase{engine: engine, never: on(0), held: on(1), session: on(2), lid: on(3), ended: on(4), near: on(5), lowPower: on(6), mac: on(7)})
		}
	}
	return out
}

func (c awakeCase) want() *verdict.Verdict {
	until := "Awake until 18:00."
	if c.lid {
		until = "Awake until 18:00, lid closed."
	}
	switch {
	case c.engine == awake.EngineReading:
		return nil
	case c.engine != awake.EngineRunning && c.mac:
		return &verdict.Verdict{Tone: verdict.Plain, Sentence: "Never sleeps, set in System Settings.", Line: "the engine is " + c.engine}
	case c.engine != awake.EngineRunning:
		return &verdict.Verdict{Tone: verdict.Plain, Sentence: "Sleeps normally.", Line: "the engine is " + c.engine}
	case c.never && !c.held:
		return &verdict.Verdict{Tone: verdict.Warn, Sentence: "Never sleeps, let go while the battery is low.", Line: "the always-on Mac policy"}
	case c.never:
		return &verdict.Verdict{Tone: verdict.OK, Sentence: "Never sleeps.", Line: "the always-on Mac policy"}
	case c.session && c.near:
		return &verdict.Verdict{Tone: verdict.Warn, Sentence: until, Line: "battery 25 %, it stops at 20 %"}
	case c.session:
		return &verdict.Verdict{Tone: verdict.OK, Sentence: until}
	case c.ended:
		return &verdict.Verdict{Tone: verdict.Warn, Sentence: "Awake stopped, battery 18 %."}
	case c.mac:
		return &verdict.Verdict{Tone: verdict.OK, Sentence: "Never sleeps, set in System Settings."}
	case c.lowPower:
		return &verdict.Verdict{Tone: verdict.Warn, Sentence: "Sleeps normally.", Line: "cannot be kept awake: Low Power Mode is on"}
	}
	return &verdict.Verdict{Tone: verdict.Plain, Sentence: "Sleeps normally."}
}

// invariant: docs/reference/awake.md's verdict table, held whole over every
// mode, engine state, policy hold, battery, guard and the Mac's own sleep
// timer: the first case that matches says the verdict.
func TestAMacsAwakeVerdictIsTheFirstCaseOfItsTableThatMatches(t *testing.T) {
	for _, c := range everyAwakeCase() {
		got, want := awake.VerdictOf(c.report()), c.want()
		if fmt.Sprint(got) != fmt.Sprint(want) || (got == nil) != (want == nil) {
			t.Errorf("%+v: %+v; want %+v", c, got, want)
		}
	}
}

// invariant: a verdict needs a look exactly where Awake's findings name
// one, but for the two cases no finding names: Never sleeps let go, and a
// Mac a keep-awake would be refused on.
func TestAnAwakeVerdictWarnsWhereItsFindingsDo(t *testing.T) {
	for _, c := range everyAwakeCase() {
		if c.engine != awake.EngineRunning {
			continue
		}
		v := awake.VerdictOf(c.report())
		found := len(awake.Findings(c.report(), machine.Ref{Name: "this"})) > 0
		unnamed := (c.never && !c.held) || (!c.never && !c.session && !c.ended && !c.mac && c.lowPower)
		if found != (v.Tone == verdict.Warn && !unnamed) {
			t.Errorf("%+v: verdict %s, a finding: %v", c, v.Tone, found)
		}
		if unnamed && v.Tone != verdict.Warn {
			t.Errorf("%+v: %s; want warn", c, v.Tone)
		}
	}
}

// invariant: the battery is near its end from BatteryWarns down to just
// above BatteryEnds, on battery, on a Mac that reports one.
func TestTheBatteryIsNearItsEndFromThirtyDownToTwentyOne(t *testing.T) {
	for percent, want := range map[int]string{31: "", 30: "battery 30 %, it stops at 20 %", 21: "battery 21 %, it stops at 20 %", 20: ""} {
		r := awake.Report{Caps: machine.Capabilities{Battery: true}, Reading: awake.Reading{HasBattery: true, Source: "battery", Percent: percent}}
		if got := awake.BatteryNear(r); got != want {
			t.Errorf("at %d %%: %q; want %q", percent, got, want)
		}
	}
	if got := awake.BatteryNear(awake.Report{Reading: awake.Reading{HasBattery: true, Source: "battery", Percent: 25}}); got != "" {
		t.Errorf("a Mac that reports no battery: %q", got)
	}
}

// invariant: the keep-awake a verdict says would be refused is the one a
// page asks, with the lid open: a lid closed on battery refuses none.
func TestAVerdictJudgesTheKeepAwakeAPageAsks(t *testing.T) {
	r := awake.Report{Engine: awake.EngineRunning, Caps: machine.Capabilities{Battery: true, Lid: true}, Reading: awake.Reading{Lid: "closed", HasBattery: true, Source: "battery", Percent: 80}}
	if v := awake.VerdictOf(r); v.Tone != verdict.Plain || v.Line != "" {
		t.Fatalf("lid closed on battery: %+v", v)
	}
}
