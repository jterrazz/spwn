package overview

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type Upkeep string

const (
	NeedsYou Upkeep = "needs-you"
	CanWait  Upkeep = "can-wait"
)

func (Upkeep) Enum() []string { return []string{string(NeedsYou), string(CanWait)} }

type Verb string

const (
	VerbReview      Verb = "review"
	VerbRestart     Verb = "restart"
	VerbStart       Verb = "start"
	VerbDelete      Verb = "delete"
	VerbUpdate      Verb = "update"
	VerbRemove      Verb = "remove"
	VerbMove        Verb = "move"
	VerbStartEngine Verb = "start-engine"
	VerbFix         Verb = "fix"
)

func (Verb) Enum() []string {
	return []string{string(VerbReview), string(VerbRestart), string(VerbStart), string(VerbDelete), string(VerbUpdate), string(VerbRemove),
		string(VerbMove), string(VerbStartEngine), string(VerbFix)}
}

func urgencyOf(f finding.Finding) finding.Urgency {
	// invariant: SPEC B5.7: whatever is broken is red, in any section (a
	// machine down, a port exposed, a sign-in expired, a background task
	// failing); a finding keeps an urgency it was given.
	if f.Urgency == "" && f.Severity == finding.Broken {
		return finding.Red
	}
	return f.Urgency
}

func UpkeepOf(f finding.Finding) Upkeep {
	// invariant: SPEC B5.7: what is urgent needs you; everything else is
	// upkeep, which can wait.
	if urgencyOf(f) != "" {
		return NeedsYou
	}
	return CanWait
}

func VerbOf(f finding.Finding) Verb {
	// invariant: a need's verb is its call's, and a port's line with no call
	// is reviewed on its machine's Ports.
	switch f.Fix.Call {
	case "engine.start":
		return VerbStartEngine
	case "apps.update":
		return VerbUpdate
	case "background.remove":
		return VerbRemove
	case "disk.move":
		return VerbMove
	case "machines.delete":
		return VerbDelete
	case "machines.restart":
		return VerbRestart
	case "machines.start":
		return VerbStart
	}
	if f.Section == section.Ports {
		return VerbReview
	}
	return VerbFix
}

const NounSignIns = "Sign-ins"

func NounOf(id section.ID) string {
	// invariant: every Accounts finding is a sign-in that fails or expired,
	// so its noun is the tab it opens, not the section's word.
	if id == section.Accounts {
		return NounSignIns
	}
	return id.Title()
}

func Idle(mac string, vm machines.VM, now time.Time) (finding.Finding, bool) {
	// invariant: a workspace idle by machines.IdleFor holds its memory for
	// nothing: a need that can wait, its gesture the delete.
	idle, ok := machines.IdleFor(vm, now)
	if !ok {
		return finding.Finding{}, false
	}
	hours := int(idle / time.Hour)
	text := vm.Name + " has been idle for " + strconv.Itoa(hours) + " h and holds " + strconv.FormatInt(vm.Resources.MemoryBytes>>30, 10) + " GB"
	return finding.Finding{Machine: machine.Ref{Name: vm.Name, On: mac}, Section: section.Overview, Severity: finding.NeedsALook, Subject: vm.Name, Text: text,
		Fix: finding.Fix{Call: "machines.delete", Args: map[string]string{"VM": vm.Name}}}, true
}

const (
	SentenceEngineStopped = "The engine is stopped."
	LineEngineStopped     = "Your VMs do not run and nothing is shared until it starts."
	SentenceFine          = "Everything is fine."
	SentenceReady         = "This Mac is ready."
)

func VerdictOf(s Summary) verdict.Verdict {
	// invariant: the head of an Overview and of Home: the engine stopped
	// first, then what needs you: the one need's own text, or the count with
	// the first of them as its line,
	// then a reading still running, then a Mac with no machine yet ready,
	// else all is well, what can wait counted as its line.
	if s.Engine == EngineStopped {
		return verdict.Verdict{Tone: verdict.Bad, Sentence: SentenceEngineStopped, Line: LineEngineStopped}
	}
	var urgent []Need
	waits, broken := 0, false
	for _, n := range s.Needs {
		if n.Upkeep == CanWait {
			waits++
			continue
		}
		urgent = append(urgent, n)
		broken = broken || n.Broken
	}
	if len(urgent) > 0 {
		tone := verdict.Warn
		if broken {
			tone = verdict.Bad
		}
		if len(urgent) == 1 {
			return verdict.Verdict{Tone: tone, Sentence: urgent[0].Text, Line: waitLine(waits)}
		}
		return verdict.Verdict{Tone: tone, Sentence: things(len(urgent), "needs", "need") + " you.", Line: urgent[0].Text}
	}
	line := waitLine(waits)
	switch {
	case s.Pending:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "Reading " + s.Head.Name + ".", Line: line}
	case s.Head.Mac() && s.Head.ThisMac && s.VMs != nil && len(s.VMs) == 0:
		return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceReady, Line: line}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceFine, Line: line}
}

func waitLine(waits int) string {
	if waits == 0 {
		return ""
	}
	return things(waits, "can", "can") + " wait."
}

func things(n int, one, other string) string {
	if n == 1 {
		return "One thing " + one
	}
	return strconv.Itoa(n) + " things " + other
}
