package machinefile_test

import (
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

func TestTheFollowersVerdictSaysHowManyMachinesDiffer(t *testing.T) {
	// invariant: a follower that did not read is never counted matching,
	// and a differing one is counted whatever the unread ones are.
	read := func(n int) machinefile.Follower {
		return machinefile.Follower{}.Judged(machinefile.Drift{Differences: n})
	}
	for _, c := range []struct {
		how  string
		in   []machinefile.Follower
		tone verdict.Tone
		said string
	}{
		{"none", nil, verdict.Plain, "No machine follows it."},
		{"all match", []machinefile.Follower{read(0), read(0)}, verdict.OK, "Every machine matches."},
		{"one differs", []machinefile.Follower{read(0), read(3)}, verdict.Warn, "1 machine differs."},
		{"two differ", []machinefile.Follower{read(1), read(3), read(0)}, verdict.Warn, "2 machines differ."},
		{"one unread", []machinefile.Follower{read(0), {}}, verdict.Plain, "Whether 1 machine matches does not read now."},
		{"a differing and an unread", []machinefile.Follower{read(2), {}}, verdict.Warn, "1 machine differs."},
	} {
		if got := machinefile.FollowersVerdict(c.in); got.Tone != c.tone || got.Sentence != c.said {
			t.Errorf("%s: %+v", c.how, got)
		}
	}
}
