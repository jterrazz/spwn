package machines

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const LineStopped = "It holds no memory. Start it to use it again."

func HeadOf(vm VM, now time.Time) (head verdict.Verdict, pending bool) {
	// invariant: a VM's head is its standing first, in that standing's tone;
	// running, how it stands with its file as its Mac's engine keeps it: its
	// differences, that it matches, or that the file did not read. A server,
	// which station sets up nothing inside, a workspace bound to no file, and
	// one its engine has not read yet (pending) are only running.
	standing := StandingOf(vm)
	word := "Running"
	switch standing {
	case StandingNotSeen, StandingSuspended:
		return verdict.Verdict{Tone: standing.Tone(), Sentence: vm.State.Word() + "."}, false
	case StandingRestarting:
		return verdict.Verdict{Tone: standing.Tone(), Sentence: SentenceRestarting}, false
	case StandingCreating:
		return verdict.Verdict{Tone: standing.Tone(), Sentence: vm.State.Word() + "."}, false
	case StandingNotCreated:
		return verdict.Verdict{Tone: standing.Tone(), Sentence: vm.Name + " was not created: it failed at " + vm.Birth.Step.Noun() + ".", Line: vm.Birth.Cause}, false
	case StandingDown:
		return verdict.Verdict{Tone: standing.Tone(), Sentence: vm.Name + " is down.", Line: downLine(vm)}, false
	case StandingStopped:
		return verdict.Verdict{Tone: standing.Tone(), Sentence: vm.Name + " is stopped.", Line: LineStopped}, false
	case StandingHealthy:
		word = "Healthy"
	case StandingRunning:
	}
	v := verdict.Verdict{Tone: standing.Tone(), Sentence: word + ".", Line: runLine(vm, now)}
	kept := vm.Drift
	switch {
	case vm.Lifetime == AlwaysOn:
	case kept == nil:
		return v, true
	case kept.Unbound:
	case kept.Failed != "":
		v.Tone, v.Sentence, v.Line = verdict.Warn, word+", but its file did not read.", kept.Failed
	case kept.Differences > 0:
		v.Tone, v.Sentence = verdict.Warn, word+", with "+difference(kept.Differences)+" from its file."
	default:
		v.Sentence = word + ", and it matches its file."
	}
	return v, false
}

func downLine(vm VM) string {
	switch {
	case vm.Cause != "":
		return vm.Cause
	case vm.CheckStatus.State == CheckFailing && !vm.CheckStatus.Since.IsZero():
		return "Its check fails since " + vm.CheckStatus.Since.Format("15:04") + "."
	}
	return ""
}

func runLine(vm VM, now time.Time) string {
	parts := []string{}
	if purpose := strings.TrimRight(vm.Purpose, "."); purpose != "" {
		parts = append(parts, purpose)
	}
	if !vm.BootAt.IsZero() && now.After(vm.BootAt) {
		parts = append(parts, "Up "+Silence(now.Sub(vm.BootAt)))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ". ") + "."
}

func difference(n int) string {
	if n == 1 {
		return "1 difference"
	}
	return strconv.Itoa(n) + " differences"
}
