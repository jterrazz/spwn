package machines

import (
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

func TestAVMsHeadSaysItsStateThenHowItStandsWithItsFile(t *testing.T) {
	now := time.Date(2026, 11, 14, 18, 0, 0, 0, time.UTC)
	boot := now.Add(-3 * 24 * time.Hour)
	workspace := func(d *KeptDrift) VM {
		return VM{Name: "linux-1", Lifetime: Persistent, State: Running, BootAt: boot, Purpose: "Fix the flaky test.", Drift: d}
	}
	server := VM{Name: "infra", Lifetime: AlwaysOn, State: Running, BootAt: boot, Check: "tcp:22", CheckStatus: CheckStatus{State: CheckPassing}, SSH: true, IP: "192.168.64.5"}
	failing := server
	failing.CheckStatus = CheckStatus{State: CheckFailing, Since: time.Date(2026, 11, 14, 14, 2, 0, 0, time.UTC)}
	downWhy := failing
	downWhy.Cause = "Its check fails since 14:02, SSH still answers."
	restarting := server
	restarting.Restart = &RestartWalk{}
	const line = "Fix the flaky test. Up 3 days."
	for name, c := range map[string]struct {
		vm      VM
		want    verdict.Verdict
		pending bool
	}{
		"differs":         {workspace(&KeptDrift{At: now, Differences: 3}), verdict.Verdict{Tone: verdict.Warn, Sentence: "Running, with 3 differences from its file.", Line: line}, false},
		"one difference":  {workspace(&KeptDrift{At: now, Differences: 1}), verdict.Verdict{Tone: verdict.Warn, Sentence: "Running, with 1 difference from its file.", Line: line}, false},
		"matches":         {workspace(&KeptDrift{At: now}), verdict.Verdict{Tone: verdict.OK, Sentence: "Running, and it matches its file.", Line: line}, false},
		"no file":         {workspace(&KeptDrift{At: now, Unbound: true}), verdict.Verdict{Tone: verdict.OK, Sentence: "Running.", Line: line}, false},
		"file unreadable": {workspace(&KeptDrift{At: now, Failed: "linux-1 does not answer"}), verdict.Verdict{Tone: verdict.Warn, Sentence: "Running, but its file did not read.", Line: "linux-1 does not answer"}, false},
		"not read yet":    {workspace(nil), verdict.Verdict{Tone: verdict.OK, Sentence: "Running.", Line: line}, true},
		"server healthy":  {server, verdict.Verdict{Tone: verdict.OK, Sentence: "Healthy.", Line: "Up 3 days."}, false},
		"server down":     {downWhy, verdict.Verdict{Tone: verdict.Bad, Sentence: "infra is down.", Line: downWhy.Cause}, false},
		"check failing":   {failing, verdict.Verdict{Tone: verdict.Bad, Sentence: "infra is down.", Line: "Its check fails since 14:02."}, false},
		"stopped":         {VM{Name: "linux-1", Lifetime: Persistent, State: Stopped}, verdict.Verdict{Tone: verdict.Plain, Sentence: "linux-1 is stopped.", Line: LineStopped}, false},
		"restarting":      {restarting, verdict.Verdict{Tone: verdict.New, Sentence: SentenceRestarting}, false},
		"creating":        {VM{Name: "linux-2", State: Creating}, verdict.Verdict{Tone: verdict.New, Sentence: "Creating."}, false},
	} {
		got, pending := HeadOf(c.vm, now)
		if got != c.want || pending != c.pending {
			t.Errorf("%s: %+v pending %v; want %+v pending %v", name, got, pending, c.want, c.pending)
		}
	}
}
