package wire

import (
	"slices"
	"testing"
)

func TestOnlyTheEnginesTwoJobsAreRunAsBundledJobs(t *testing.T) {
	for args, want := range map[string]string{
		"engine run":     "com.jterrazz.spwn.engine",
		"machines agent": "com.jterrazz.spwn.agent",
		"engine start":   "",
		"engine":         "",
	} {
		if got := jobOf(splitWords(args)); got != want {
			t.Errorf("jobOf(%q) = %q, want %q", args, got, want)
		}
	}
}

func TestABundledJobTakesTheRecordedEnvironmentOverLaunchds(t *testing.T) {
	got := withStart([]string{"HOME=/Users/o", "PATH=/usr/bin:/bin", "TMPDIR=/tmp/x"}, map[string]string{"PATH": "/opt/homebrew/bin:/usr/bin", "XDG_STATE_HOME": "/s"})
	want := []string{"HOME=/Users/o", "TMPDIR=/tmp/x", "PATH=/opt/homebrew/bin:/usr/bin", "XDG_STATE_HOME=/s"}
	if !slices.Equal(got, want) {
		t.Errorf("env = %q, want %q", got, want)
	}
}

func splitWords(s string) []string {
	var words []string
	word := ""
	for _, r := range s + " " {
		if r == ' ' {
			if word != "" {
				words = append(words, word)
			}
			word = ""
			continue
		}
		word += string(r)
	}
	return words
}
