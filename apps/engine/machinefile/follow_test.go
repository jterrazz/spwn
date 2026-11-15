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
		return machinefile.Following("m", "mac", false).Judged(machinefile.Drift{Differences: n})
	}
	unread := machinefile.Following("m", "mac", false)
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
		{"one unread", []machinefile.Follower{read(0), unread}, verdict.Plain, "Whether 1 machine matches does not read now."},
		{"a differing and an unread", []machinefile.Follower{read(2), unread}, verdict.Warn, "1 machine differs."},
	} {
		if got := machinefile.FollowersVerdict(c.in); got.Tone != c.tone || got.Sentence != c.said {
			t.Errorf("%s: %+v", c.how, got)
		}
	}
}

func TestAFollowerStandsAsItsDriftSaysAndWaitsUnreadUntilJudged(t *testing.T) {
	// invariant: a follower is unread until a drift judges it, matches
	// at no difference and differs with the count at any.
	for _, c := range []struct {
		how   string
		in    machinefile.Follower
		state machinefile.FollowerState
		diffs int
	}{
		{"not judged", machinefile.Following("m", "mac", false), machinefile.FollowerUnread, 0},
		{"no difference", machinefile.Following("m", "mac", false).Judged(machinefile.Drift{}), machinefile.FollowerMatches, 0},
		{"three differences", machinefile.Following("m", "mac", false).Judged(machinefile.Drift{Differences: 3}), machinefile.FollowerDiffers, 3},
	} {
		if c.in.State != c.state || c.in.Differences != c.diffs {
			t.Errorf("%s: %+v", c.how, c.in)
		}
	}
}

func TestTheFilesVerdictCountsTheFilesThatDifferFromTheirOwnVerdicts(t *testing.T) {
	// invariant: the page says how many files differ from what each file's
	// verdict says, never from its followers again.
	file := func(fs ...machinefile.Follower) machinefile.FileView {
		return machinefile.FileView{Followers: fs, Verdict: machinefile.FollowersVerdict(fs)}
	}
	judged := func(n int) machinefile.Follower {
		return machinefile.Following("m", "mac", false).Judged(machinefile.Drift{Differences: n})
	}
	unread := machinefile.Following("m", "mac", false)
	for _, c := range []struct {
		how  string
		in   []machinefile.FileView
		tone verdict.Tone
		said string
	}{
		{"no files", nil, verdict.Plain, "No machine follows a file."},
		{"no follower", []machinefile.FileView{file(), file()}, verdict.Plain, "No machine follows a file."},
		{"all match", []machinefile.FileView{file(judged(0)), file(), file(judged(0), judged(0))}, verdict.OK, "Every machine matches its file."},
		{"one file differs", []machinefile.FileView{file(judged(0)), file(judged(0), judged(2))}, verdict.Warn, "1 file differs from its machines."},
		{"two files differ", []machinefile.FileView{file(judged(1)), file(judged(2)), file(judged(0))}, verdict.Warn, "2 files differ from their machines."},
		{"one unread", []machinefile.FileView{file(judged(0)), file(unread)}, verdict.Plain, "Whether the machines match their files does not read now."},
		{"a difference and an unread", []machinefile.FileView{file(judged(1)), file(unread)}, verdict.Warn, "1 file differs from its machines."},
	} {
		if got := machinefile.FilesVerdict(c.in); got.Tone != c.tone || got.Sentence != c.said {
			t.Errorf("%s: %+v", c.how, got)
		}
	}
}
