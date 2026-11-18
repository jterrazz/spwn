package awake_test

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/awake"
)

func TestAMacSaysHowItSleepsInTheWordsOfItsAwakeLine(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 30, 0, 0, time.UTC)
	until := time.Date(2026, 11, 7, 18, 0, 0, 0, time.UTC)
	for name, c := range map[string]struct {
		report awake.Report
		want   string
	}{
		"nothing keeps it":   {awake.Report{Now: now}, "sleeps normally"},
		"never sleeps":       {awake.Report{Now: now, NeverSleeps: true, NeverHeld: true}, "never sleeps"},
		"let go":             {awake.Report{Now: now, NeverSleeps: true}, "never sleeps, let go while the battery is low"},
		"engine stopped":     {awake.Report{Now: now, Engine: awake.EngineStopped, NeverSleeps: true, NeverHeld: true}, "sleeps normally"},
		"until an hour":      {awake.Report{Now: now, Session: &awake.Session{Until: until}}, "awake until 18:00"},
		"lid closed":         {awake.Report{Now: now, Session: &awake.Session{Until: until, Lid: true}}, "awake until 18:00, lid closed"},
		"stopped by guard":   {awake.Report{Now: now, LastEnd: &awake.End{At: now.Add(-time.Minute), Reason: awake.Lid, Detail: "lid closed on battery"}}, "awake stopped, lid closed on battery"},
		"guard's end is old": {awake.Report{Now: now, LastEnd: &awake.End{At: now.Add(-time.Hour), Reason: awake.Lid, Detail: "lid closed on battery"}}, "sleeps normally"},
	} {
		if got := c.report.Phrase(); got != c.want {
			t.Errorf("%s: %q; want %q", name, got, c.want)
		}
	}
}
