package overview

import (
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
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
	VerbTryAgain    Verb = "try-again"
	VerbUpdate      Verb = "update"
	VerbRemove      Verb = "remove"
	VerbMove        Verb = "move"
	VerbStartEngine Verb = "start-engine"
	VerbFix         Verb = "fix"
)

func (Verb) Enum() []string {
	return []string{string(VerbReview), string(VerbRestart), string(VerbStart), string(VerbTryAgain), string(VerbUpdate), string(VerbRemove),
		string(VerbMove), string(VerbStartEngine), string(VerbFix)}
}

type Mark string

const (
	MarkExposed    Mark = "exposed"
	MarkDown       Mark = "down"
	MarkNotCreated Mark = "not-created"
)

func (Mark) Enum() []string {
	return []string{string(MarkExposed), string(MarkDown), string(MarkNotCreated), string(NeedsYou), string(CanWait)}
}

func MarkOf(f finding.Finding, vms []VM) Mark {
	// invariant: a port reachable outside is Exposed and a server that is
	// down is Down and a VM whose create failed is Not created, all red; any
	// other need wears its Upkeep's word.
	switch {
	case f.Severity == finding.Broken && f.Section == section.Ports:
		return MarkExposed
	case f.Severity == finding.Broken && f.Section == section.Overview && f.Again && slices.ContainsFunc(vms, func(vm VM) bool { return vm.Name == f.Subject }):
		return MarkNotCreated
	case f.Severity == finding.Broken && f.Section == section.Overview && slices.ContainsFunc(vms, func(vm VM) bool { return vm.Name == f.Subject }):
		return MarkDown
	}
	return Mark(UpkeepOf(f))
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
	case "machines.restart":
		return VerbRestart
	case "machines.start":
		if f.Again {
			return VerbTryAgain
		}
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

const (
	SentenceEngineStopped = "The engine is stopped."
	LineEngineStopped     = "Your VMs do not run and nothing is shared until it starts."
	SentenceFine          = "Everything is fine."
	SentenceReady         = "This Mac is ready."
)

func VerdictOf(s Summary, elsewhere *machines.RollUp) verdict.Verdict {
	// invariant: the head of Home: the engine stopped first, then what needs
	// you: a server down by its name (or a VM not created, said so), else the count; the other Macs' roll-up
	// next, said as "Elsewhere" when it is not fine; a reading still
	// running, then a Mac with no machine yet ready, else all is well. What
	// is left past the first is said as the rest, the machines counted, and
	// what can wait stays the client's count of the can-wait needs.
	// elsewhere is in three states: the zero roll-up when the other Macs are
	// not asked (today's wording, scoped to this Mac), nil while they are
	// asked and still read (the verdict claims nothing of them), else their
	// roll-up.
	if s.Engine == EngineStopped {
		return verdict.Verdict{Tone: verdict.Bad, Sentence: SentenceEngineStopped, Line: LineEngineStopped}
	}
	var urgent []Need
	broken := false
	for _, n := range s.Needs {
		if n.Upkeep == NeedsYou {
			urgent = append(urgent, n)
			broken = broken || n.Broken
		}
	}
	tail := machinesLine(s)
	claim, unknown := "", elsewhere == nil
	var claimTone verdict.Tone
	if elsewhere != nil {
		switch elsewhere.State {
		case "", machines.RollUpFine, machines.RollUpEmpty:
		default:
			claim, claimTone = "Elsewhere, "+elsewhere.Words()+".", elsewhere.State.Tone()
		}
	}
	rest := claim
	switch {
	case unknown:
	case claim == "":
		rest = "The rest is fine."
	}
	if len(urgent) > 0 {
		tone := verdict.Warn
		if broken || claimTone == verdict.Bad {
			tone = verdict.Bad
		}
		if down, ok := serverDown(urgent); ok {
			line := rest
			if len(urgent) > 1 {
				line = more(len(urgent)-1) + " " + line
			}
			sentence := down.Subject + " is down."
			if down.Verb == VerbTryAgain {
				sentence = down.Subject + " was not created."
			}
			return verdict.Verdict{Tone: verdict.Bad, Sentence: sentence, Line: strings.TrimSpace(line + " " + tail)}
		}
		return verdict.Verdict{Tone: tone, Sentence: things(len(urgent), "needs", "need") + " you.", Line: strings.TrimSpace(rest + " " + tail)}
	}
	switch {
	case claim != "":
		return verdict.Verdict{Tone: claimTone, Sentence: claim, Line: tail}
	case s.Pending || unknown:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "Reading " + s.Head.Name + ".", Line: tail}
	case s.Head.Mac() && s.Head.ThisMac && s.VMs != nil && len(s.VMs) == 0:
		return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceReady}
	}
	return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceFine, Line: tail}
}

func serverDown(urgent []Need) (Need, bool) {
	for _, n := range urgent {
		if n.Mark == MarkDown || n.Mark == MarkNotCreated {
			return n, true
		}
	}
	return Need{}, false
}

func more(n int) string {
	if n == 1 {
		return "1 more thing needs you."
	}
	return strconv.Itoa(n) + " more things need you."
}

func machinesLine(s Summary) string {
	// invariant: the count is of the machines this Overview's Mac holds, the
	// only ones whose needs it reads, so it names that Mac, never a count
	// of Macs it has not read.
	switch len(s.VMs) {
	case 0:
		return ""
	case 1:
		return "1 machine on " + s.Head.Name
	}
	return strconv.Itoa(len(s.VMs)) + " machines on " + s.Head.Name
}

func things(n int, one, other string) string {
	if n == 1 {
		return "1 thing " + one
	}
	return strconv.Itoa(n) + " things " + other
}
