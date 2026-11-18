package settings_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/settings"
)

func TestUnifiedDiffShowsTheLineYouChangedAgainstStationsWithItsContext(t *testing.T) {
	want := "a\nb\nc\nd\ne\nf\ng\n"
	have := "a\nb\nc\nD\ne\nf\ng\n"
	got := settings.UnifiedDiff(want, have, "station", "~/x")
	expect := "--- station\n+++ ~/x\n@@ -2,5 +2,5 @@\n b\n c\n-d\n+D\n e\n f\n"
	if got != expect {
		t.Fatalf("diff =\n%s\nwant\n%s", got, expect)
	}
}

func TestUnifiedDiffIsEmptyWhenNothingChanged(t *testing.T) {
	if got := settings.UnifiedDiff("a\n", "a\n", "station", "~/x"); got != "" {
		t.Fatalf("diff = %q", got)
	}
}

func TestUnifiedDiffOfAMissingFileRemovesEveryLine(t *testing.T) {
	got := settings.UnifiedDiff("a\nb\n", "", "station", "~/x")
	if got != "--- station\n+++ ~/x\n@@ -1,2 +1,0 @@\n-a\n-b\n" {
		t.Fatalf("diff =\n%s", got)
	}
}

func TestAReversedDiffIsTheDiffDrawnTheOtherWay(t *testing.T) {
	for _, c := range []struct{ a, b string }{
		{"a\nb\nc\nd\ne\nf\ng\n", "a\nb\nc\nD\ne\nf\ng\n"},
		{"a\nb\n", ""},
		{"", "a\nb\n"},
		{"a\nb\nc\n", "a\nx\ny\nc\nz\n"},
		{"-- x\n", "++ y\n"},
		{"a\n-- x\nb\n", "a\n++ y\nb\n"},
		{"@@ -1 +1 @@\n", "--- one\n+++ two\n"},
	} {
		got := settings.ReversedDiff(settings.UnifiedDiff(c.a, c.b, "one", "two"))
		if want := settings.UnifiedDiff(c.b, c.a, "two", "one"); got != want {
			t.Errorf("reversed %q -> %q =\n%s\nwant\n%s", c.a, c.b, got, want)
		}
	}
	if got := settings.ReversedDiff(""); got != "" {
		t.Errorf("reversed nothing = %q", got)
	}
}
