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
		got := awake.Choices(now, paris)
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
