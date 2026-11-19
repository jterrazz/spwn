package awake_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
)

func TestEveryChoiceIsATermKeepTakesAndEndsWhereKeepEndsIt(t *testing.T) {
	paris := time.FixedZone("CET", 3600)
	for name, now := range map[string]time.Time{
		"before 18:00": time.Date(2026, 11, 4, 9, 0, 0, 0, paris),
		"after 18:00":  time.Date(2026, 11, 4, 19, 30, 0, 0, paris),
		"at 18:00":     time.Date(2026, 11, 4, 18, 0, 0, 0, paris),
	} {
		got := awake.Choices(now, paris, nil)
		if len(got) != 3 || got[0].Label != "1 hr" || got[1].Label != "4 hr" || got[2].Label != "Until 18:00" {
			t.Fatalf("%s offers %+v", name, got)
		}
		for i, words := range []string{"1h", "4h"} {
			if want, err := awake.For(words, now, paris); err != nil || got[i].For != want || got[i].Until != "" {
				t.Errorf("%s: %s is %v, keep takes %v (%v)", name, got[i].Label, got[i].For, want, err)
			}
		}
		left, err := awake.For("until 18:00", now, paris)
		if err != nil {
			t.Fatal(err)
		}
		if want := now.Add(left).In(paris).Format(time.RFC3339); got[2].Until != want || got[2].For != 0 {
			t.Errorf("%s: Until 18:00 is %q, keep ends it at %q", name, got[2].Until, want)
		}
	}
}

func TestTheTermOfTheKeepAwakeHeldIsTheOneChoicePicked(t *testing.T) {
	paris := time.FixedZone("CET", 3600)
	now := time.Date(2026, 11, 4, 9, 0, 0, 0, paris)
	for name, c := range map[string]struct {
		held      *awake.Session
		label, at string
	}{
		"nothing held":           {nil, "", ""},
		"asked for 1 h":          {&awake.Session{StartedAt: now, Until: now.Add(time.Hour), For: time.Hour}, "1 hr", ""},
		"asked for 4 h":          {&awake.Session{StartedAt: now.Add(-time.Hour), Until: now.Add(3 * time.Hour), For: 4 * time.Hour}, "4 hr", ""},
		"4 h, to the minute":     {&awake.Session{StartedAt: now, Until: now.Add(4*time.Hour - 20*time.Second), For: 4*time.Hour - 20*time.Second}, "4 hr", ""},
		"its span, no word":      {&awake.Session{StartedAt: now.Add(-3 * time.Minute), Until: now.Add(57 * time.Minute)}, "1 hr", ""},
		"until an hour none has": {&awake.Session{StartedAt: now, Until: now.Add(8 * time.Hour), For: 8 * time.Hour}, "Until 17:00", "2026-11-04T17:00:00+01:00"},
	} {
		var picked []awake.Choice
		for _, ch := range awake.Choices(now, paris, c.held) {
			if ch.Picked {
				picked = append(picked, ch)
			}
		}
		switch {
		case c.label == "" && len(picked) != 0:
			t.Errorf("%s picks %+v, want none", name, picked)
		case c.label != "" && (len(picked) != 1 || picked[0].Label != c.label || picked[0].Until != c.at):
			t.Errorf("%s picks %+v, want %s (%s) alone", name, picked, c.label, c.at)
		}
	}
}

func TestNeverSleepsPicksNoTermOfTheKeepAwakeItWinsOver(t *testing.T) {
	now := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	r := awake.Report{Now: now, NeverSleeps: true, NeverHeld: true, Session: &awake.Session{StartedAt: now, Until: now.Add(4 * time.Hour), For: 4 * time.Hour}}
	for _, c := range awake.Choices(now, time.UTC, r.Keeping()) {
		if c.Picked {
			t.Fatalf("Never sleeps picks %+v", c)
		}
	}
}
