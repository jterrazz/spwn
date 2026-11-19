package machines

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

func TestTheHeadOfAReleaseReading(t *testing.T) {
	for _, c := range []struct {
		installed, latest string
		want              verdict.Verdict
		behind            bool
	}{
		{"v0.21.0", "v0.22.0", verdict.Verdict{Tone: verdict.New, Sentence: "Spwn v0.22.0 is out.", Line: "This Mac runs v0.21.0."}, true},
		{"v0.22.0", "v0.22.0", verdict.Verdict{Tone: verdict.OK, Sentence: "This Mac runs the latest release."}, false},
		{"v0.21.0-alpha.1", "v0.20.0", verdict.Verdict{Tone: verdict.Plain, Sentence: "This Mac runs v0.21.0-alpha.1, past the latest release v0.20.0."}, false},
		{"v0.20.0", "v0.21.0-alpha.1", verdict.Verdict{Tone: verdict.New, Sentence: "Spwn v0.21.0-alpha.1 is out.", Line: "This Mac runs v0.20.0."}, true},
		{"v0.21.0-alpha.1", "v0.21.0", verdict.Verdict{Tone: verdict.New, Sentence: "Spwn v0.21.0 is out.", Line: "This Mac runs v0.21.0-alpha.1."}, true},
		{"v0.22.0", "", verdict.Verdict{Tone: verdict.Plain, Sentence: "The latest release is not known yet."}, false},
		{"dev", "v0.22.0", verdict.Verdict{Tone: verdict.Plain, Sentence: "This Mac runs a build of no release."}, false},
		{"dry-run-1018bfdf9", "v0.22.0", verdict.Verdict{Tone: verdict.Plain, Sentence: "This Mac runs a build of no release."}, false},
	} {
		got, behind := ReleaseHead(c.installed, c.latest), Newer(c.latest, c.installed)
		if got != c.want || behind != c.behind {
			t.Errorf("%s against %q: head %+v, behind %v; want %+v, %v", c.installed, c.latest, got, behind, c.want, c.behind)
		}
	}
}
