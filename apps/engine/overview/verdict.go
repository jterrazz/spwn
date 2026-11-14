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
	// invariant: SPEC B5.7: a machine broken (down, silent, with no engine)
	// or a port exposed is red; a finding keeps an urgency it was given.
	if f.Urgency == "" && f.Severity == finding.Broken && (f.Section == section.Overview || f.Section == section.Ports) {
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

func Idle(mac string, vm machines.VM, now time.Time) (finding.Finding, bool) {
	// invariant: a workspace that runs on after its agent finished, for
	// IdleAfter or longer, holds its memory for nothing: a need that can
	// wait, its gesture the delete.
	if vm.Lifetime == machines.AlwaysOn || vm.State != machines.Running || !vm.Finished() || vm.Agent.Ended.IsZero() || now.Sub(vm.Agent.Ended) < IdleAfter {
		return finding.Finding{}, false
	}
	hours := int(now.Sub(vm.Agent.Ended) / time.Hour)
	text := vm.Name + " has been idle for " + strconv.Itoa(hours) + " h and holds " + strconv.FormatInt(vm.Resources.MemoryBytes>>30, 10) + " GB"
	return finding.Finding{Machine: machine.Ref{Name: vm.Name, On: mac}, Section: section.Overview, Severity: finding.NeedsALook, Subject: vm.Name, Text: text,
		Fix: finding.Fix{Call: "machines.delete", Args: map[string]string{"VM": vm.Name}}}, true
}

const (
	SentenceEngineStopped = "The engine is stopped."
	// value: how long a workspace's agent has been finished before the
	// workspace is idle, a need that can wait.
	IdleAfter         = time.Hour
	LineEngineStopped = "Your VMs do not run and nothing is shared until it starts."
	SentenceFine      = "Everything is fine."
	SentenceReady     = "This Mac is ready."
)

func VerdictOf(s Summary) verdict.Verdict {
	// invariant: the head of an Overview and of Home: the engine stopped
	// first, then what needs you counted with the first of it as its line,
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
		return verdict.Verdict{Tone: tone, Sentence: things(len(urgent), "needs", "need") + " you.", Line: urgent[0].Text}
	}
	line := ""
	if waits > 0 {
		line = things(waits, "can", "can") + " wait."
	}
	switch {
	case s.Pending:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "Reading " + s.Head.Name + ".", Line: line}
	case s.Head.Mac() && s.Head.ThisMac && s.VMs != nil && len(s.VMs) == 0:
		return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceReady, Line: line}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceFine, Line: line}
}

func things(n int, one, other string) string {
	if n == 1 {
		return "One thing " + one
	}
	return strconv.Itoa(n) + " things " + other
}
