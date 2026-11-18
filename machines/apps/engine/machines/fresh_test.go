package machines

import (
	"testing"
	"time"
)

func TestAMacIsStaleOnceUnheardForThirtySeconds(t *testing.T) {
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name  string
		mac   MacView
		stale bool
	}{
		{"This Mac", MacView{ThisMac: true}, false},
		{"a Mac that answers", MacView{Reachable: true, Answered: now.Add(-time.Hour)}, false},
		{"silent, last heard 30 s ago", MacView{Answered: now.Add(-StaleAfter)}, false},
		{"silent, last heard 31 s ago", MacView{Answered: now.Add(-31 * time.Second)}, true},
		{"never heard from", MacView{}, true},
	} {
		if got := Unheard(c.mac, now); got != c.stale {
			t.Errorf("%s: stale %v, want %v", c.name, got, c.stale)
		}
	}
}

func TestAnAsleepMacIsToldApartFromAnUnreachableOne(t *testing.T) {
	for _, c := range []struct {
		name string
		mac  MacView
		want string
	}{
		{"This Mac", MacView{ThisMac: true}, ReachAnswers},
		{"a laptop answering with its lid closed", MacView{Reachable: true, Asleep: true}, ReachAnswers},
		{"silent after it answered on battery with its lid closed", MacView{Asleep: true}, ReachAsleep},
		{"silent otherwise", MacView{Laptop: true}, ReachUnreachable},
	} {
		if got := Reach(c.mac); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
