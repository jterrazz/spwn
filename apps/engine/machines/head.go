package machines

import (
	"strconv"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const LineStopped = "It holds no memory. Start it to use it again."

type KeptDrift struct {
	// value: how a running workspace stands with its Machinefile, as its
	// Mac's engine last read it: when, how many differences every chapter
	// counts, that no file is bound to it, or why its file did not read.
	At          time.Time
	Differences int
	Unbound     bool   `json:",omitempty"`
	Failed      string `json:",omitempty"`
}

func HeadOf(vm VM, now time.Time) (head verdict.Verdict, pending bool) {
	// invariant: a VM's head is its state first (restarting or creating,
	// down, stopped), then, running, how it stands with its file as its
	// Mac's engine keeps it: its differences, that it matches, or that the
	// file did not read. A server, which station sets up nothing inside, a
	// workspace bound to no file, and one its engine has not read yet
	// (pending) are only running.
	switch {
	case vm.Restart != nil:
		return verdict.Verdict{Tone: verdict.New, Sentence: SentenceRestarting}, false
	case vm.State == Creating:
		return verdict.Verdict{Tone: verdict.New, Sentence: vm.State.Word() + "."}, false
	case down(vm):
		return verdict.Verdict{Tone: verdict.Bad, Sentence: vm.Name + " is down.", Line: downLine(vm)}, false
	case vm.State == Stopped:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: vm.Name + " is stopped.", Line: LineStopped}, false
	case vm.State != Running:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: vm.State.Word() + "."}, false
	}
	word := "Running"
	if vm.Lifetime == AlwaysOn && HealthVerdict(vm).Sentence == SentenceHealthy {
		word = "Healthy"
	}
	v := verdict.Verdict{Tone: verdict.OK, Sentence: word + ".", Line: runLine(vm, now)}
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

func down(vm VM) bool {
	switch {
	case Down(vm), vm.State == Broken, vm.State == Stale:
		return true
	}
	return vm.Lifetime == AlwaysOn && vm.State == Running && vm.CheckStatus.State == CheckFailing
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
