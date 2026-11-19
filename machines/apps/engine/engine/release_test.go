package engine

import (
	"testing"
	"time"
)

func TestAReleaseWithoutThisMacsBinaryLeavesTheLatestAsItWas(t *testing.T) {
	now := time.Unix(100, 0)
	before := ReleaseKept{Read: time.Unix(0, 0), Latest: &Tagged{Tag: "v0.21.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.21.0"}}
	got := before.Answered(now, Tagged{Tag: "v0.22.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.22.0", Assets: []string{"station-linux-arm64", "station-darwin-amd64"}}, "arm64")
	if got.Latest == nil || got.Latest.Tag != "v0.21.0" || !got.Read.Equal(now) {
		t.Errorf("answered %+v; want v0.21.0 kept, read at %v", got, now)
	}
	got = before.Answered(now, Tagged{Tag: "v0.22.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.22.0", Assets: []string{"station-darwin-arm64"}}, "arm64")
	if got.Latest == nil || got.Latest.Tag != "v0.22.0" || got.Latest.URL != "https://github.com/jterrazz/spwn/releases/tag/v0.22.0" {
		t.Errorf("answered %+v; want v0.22.0 taken", got)
	}
}

func TestTheNextAskIsSixHoursAfterAnAnswerAndHalfAnHourAfterARefusal(t *testing.T) {
	read := time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
	k := ReleaseKept{Read: read}
	if got := k.Due(read.Add(time.Minute)); !got.Equal(read.Add(6 * time.Hour)) {
		t.Errorf("due %v after an answer; want 6 h after it", got)
	}
	refused := k.Failed(read.Add(time.Hour), "GitHub said 502")
	if got := refused.Due(read.Add(time.Hour)); !got.Equal(read.Add(time.Hour + 30*time.Minute)) {
		t.Errorf("due %v after a refusal; want 30 min after it", got)
	}
	if got := refused.Due(read.Add(-time.Hour)); !got.Equal(read.Add(-time.Hour)) {
		t.Errorf("due %v with a kept ask ahead of the clock; want now", got)
	}
	shown := ReleaseOf("v0.21.0", refused, true)
	if shown.Refused == nil || shown.Refused.Message != "GitHub said 502" || shown.Checked == nil || !shown.Checked.Equal(read) {
		t.Errorf("read %+v; want the refusal beside the last answer", shown)
	}
	if again := ReleaseOf("v0.21.0", refused.Answered(read.Add(2*time.Hour), Tagged{}, "arm64"), true); again.Refused != nil {
		t.Errorf("read %+v; an answer after a refusal shows none", again)
	}
}

func TestATagNewerCannotReadIsNotTaken(t *testing.T) {
	now := time.Unix(100, 0)
	before := ReleaseKept{Read: time.Unix(0, 0), Latest: &Tagged{Tag: "v0.21.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.21.0"}}
	for _, tag := range []string{"v0.22", "v0.22.0+b1", "latest", "0.22.0"} {
		got := before.Answered(now, Tagged{Tag: tag, URL: "https://github.com/jterrazz/spwn/releases/tag/" + tag, Assets: []string{"station-darwin-arm64"}}, "arm64")
		if got.Latest == nil || got.Latest.Tag != "v0.21.0" || !got.Read.Equal(now) {
			t.Errorf("tag %q answered %+v; want v0.21.0 kept, read at %v", tag, got, now)
		}
	}
}

func TestNotesAreKeptOnlyAsAPageOfGitHub(t *testing.T) {
	assets := []string{"station-darwin-arm64"}
	page := "https://github.com/jterrazz/spwn/releases/tag/v0.22.0"
	for _, c := range []struct{ given, want string }{
		{page, page},
		{"http://github.com/jterrazz/spwn", ""},
		{"https://github.com.evil.example/x", ""},
		{"javascript:alert(1)", ""},
		{"", ""},
	} {
		got := ReleaseKept{}.Answered(time.Unix(1, 0), Tagged{Tag: "v0.22.0", URL: c.given, Assets: assets}, "arm64")
		if got.Latest == nil || got.Latest.Tag != "v0.22.0" || got.Latest.URL != c.want {
			t.Errorf("notes %q kept as %+v; want %q", c.given, got.Latest, c.want)
		}
	}
}
