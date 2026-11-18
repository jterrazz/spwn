package machines

import (
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type VMStanding string

const (
	StandingNotSeen    VMStanding = "not-seen"
	StandingRestarting VMStanding = "restarting"
	StandingCreating   VMStanding = "creating"
	StandingDown       VMStanding = "down"
	StandingStopped    VMStanding = "stopped"
	StandingSuspended  VMStanding = "suspended"
	StandingHealthy    VMStanding = "healthy"
	StandingRunning    VMStanding = "running"
)

func (VMStanding) Enum() []string {
	return []string{string(StandingNotSeen), string(StandingRestarting), string(StandingCreating), string(StandingDown),
		string(StandingStopped), string(StandingSuspended), string(StandingHealthy), string(StandingRunning)}
}

func (s VMStanding) Word() string {
	// value: a VM's standing as a client says it, a lowercase fragment.
	switch s {
	case StandingNotSeen:
		return "not seen"
	case StandingRestarting, StandingCreating, StandingDown, StandingStopped, StandingSuspended, StandingHealthy, StandingRunning:
	}
	return string(s)
}

func (s VMStanding) Tone() verdict.Tone {
	switch s {
	case StandingHealthy, StandingRunning:
		return verdict.OK
	case StandingRestarting, StandingCreating:
		return verdict.New
	case StandingDown:
		return verdict.Bad
	case StandingNotSeen, StandingStopped, StandingSuspended:
	}
	return verdict.Plain
}

func StandingOf(vm VM) VMStanding {
	// invariant: a VM has one standing, its first rule that matches: its
	// Mac silent, a restart walking, a create running, down, stopped,
	// suspended, a server healthy, running. Head, Health's down and Down
	// read it; no other rule says a VM is down.
	switch {
	case vm.State == UnknownState:
		return StandingNotSeen
	case vm.Restart != nil:
		return StandingRestarting
	case vm.State == Creating:
		return StandingCreating
	case isDown(vm):
		return StandingDown
	case vm.State == Stopped:
		return StandingStopped
	case vm.State == Suspended:
		return StandingSuspended
	case vm.Lifetime == AlwaysOn && vm.State == Running && passes(vm):
		return StandingHealthy
	}
	return StandingRunning
}

func isDown(vm VM) bool {
	// invariant: down is a VM in a loop or ended unexpectedly; a server in
	// a loop, powered off from inside, or stopped while its record wants it
	// running; a server running whose check fails (SPEC §10.1's red badge).
	switch vm.State {
	case Broken, Stale:
		return true
	case Stopped:
		return vm.Lifetime == AlwaysOn && (vm.Restarts.Looping || vm.Detail == PoweredOffInside || Wanted(vm) == "running")
	case Running:
		return vm.Lifetime == AlwaysOn && (vm.Restarts.Looping || vm.CheckStatus.State == CheckFailing)
	case Suspended, UnknownState, Creating:
	}
	return false
}

type MacStanding string

const (
	MacAsleep             MacStanding = "asleep"
	MacNotSeen            MacStanding = "not-seen"
	MacEngineStopped      MacStanding = "engine-stopped"
	MacEngineNotInstalled MacStanding = "engine-not-installed"
	MacOnline             MacStanding = "online"
)

func (MacStanding) Enum() []string {
	return []string{string(MacAsleep), string(MacNotSeen), string(MacEngineStopped), string(MacEngineNotInstalled), string(MacOnline)}
}

func (s MacStanding) Word() string {
	// value: a Mac's standing as a client says it, a lowercase fragment.
	return strings.ReplaceAll(string(s), "-", " ")
}

func (s MacStanding) Tone() verdict.Tone {
	// invariant: a Mac that does not answer is never bad: nothing proves it
	// down. Asleep is plain, silent otherwise warn.
	switch s {
	case MacOnline:
		return verdict.OK
	case MacNotSeen, MacEngineNotInstalled:
		return verdict.Warn
	case MacEngineStopped:
		return verdict.Bad
	case MacAsleep:
	}
	return verdict.Plain
}

func MacStandingOf(h MacView) MacStanding {
	// invariant: a Mac has one standing, its first rule that matches: asleep,
	// not seen, its engine stopped, its engine not installed, online. An
	// engine this Mac cannot know (another Mac, no hypervisor) reads online.
	// Its sentence, its Unseen verdict and its findings read it.
	switch Reach(h) {
	case ReachAsleep:
		return MacAsleep
	case ReachUnreachable:
		return MacNotSeen
	}
	switch h.Engine.State {
	case EngineStopped:
		return MacEngineStopped
	case EngineNotInstalled:
		return MacEngineNotInstalled
	case EngineRunning:
	}
	return MacOnline
}

type RollUpState string

const (
	RollUpDown               RollUpState = "down"
	RollUpEngineStopped      RollUpState = "engine-stopped"
	RollUpNotSeen            RollUpState = "not-seen"
	RollUpEngineNotInstalled RollUpState = "engine-not-installed"
	RollUpEmpty              RollUpState = "empty"
	RollUpFine               RollUpState = "fine"
)

func (RollUpState) Enum() []string {
	return []string{string(RollUpDown), string(RollUpEngineStopped), string(RollUpNotSeen), string(RollUpEngineNotInstalled), string(RollUpEmpty), string(RollUpFine)}
}

func (s RollUpState) Tone() verdict.Tone {
	// invariant: a state that counts a standing takes that standing's tone.
	switch s {
	case RollUpDown:
		return StandingDown.Tone()
	case RollUpEngineStopped:
		return MacEngineStopped.Tone()
	case RollUpNotSeen:
		return MacNotSeen.Tone()
	case RollUpEngineNotInstalled:
		return MacEngineNotInstalled.Tone()
	case RollUpFine:
		return verdict.OK
	case RollUpEmpty:
	}
	return verdict.Plain
}

func (s RollUpState) Word() string {
	// value: the state as a client says it bare; empty and fine are whole
	// sentences, the others read after their N through Noun.
	switch s {
	case RollUpEmpty:
		return "no machine yet"
	case RollUpFine:
		return "all running"
	case RollUpDown, RollUpEngineStopped, RollUpNotSeen, RollUpEngineNotInstalled:
	}
	return strings.ReplaceAll(string(s), "-", " ")
}

func (s RollUpState) Noun(n int) string {
	// value: what n of the state are called, read after n ("1 machine
	// down", "2 Macs not seen"); empty and fine count nothing and have none.
	one := n == 1
	switch s {
	case RollUpDown:
		return pick(one, "machine down", "machines down")
	case RollUpEngineStopped:
		return pick(one, "Mac with its engine stopped", "Macs with their engine stopped")
	case RollUpNotSeen:
		return pick(one, "Mac not seen", "Macs not seen")
	case RollUpEngineNotInstalled:
		return pick(one, "Mac without its engine", "Macs without their engine")
	case RollUpEmpty, RollUpFine:
	}
	return ""
}

func pick(one bool, singular, plural string) string {
	if one {
		return singular
	}
	return plural
}

type RollUp struct {
	State RollUpState
	// value: how many its state names: the VMs down, or the Macs whose
	// engine is stopped, not seen, or whose engine is not installed; absent
	// for empty and fine.
	N int `json:",omitempty"`
}

func (r RollUp) Words() string {
	// value: the roll-up as one phrase: its N and noun, or its word when it
	// counts nothing.
	if noun := r.State.Noun(r.N); noun != "" {
		return strconv.Itoa(r.N) + " " + noun
	}
	return r.State.Word()
}

func RollUpOf(macs []MacView) RollUp {
	// invariant: how a set of Macs and their VMs stands, read from their
	// standings alone, its first rule that matches, bad before warn: down (a
	// VM down), engine stopped, not seen, engine not installed (each a Mac
	// in that standing), empty (no VM), fine. A VM of a Mac that does not
	// answer counts through its Mac only; an asleep Mac is no fault.
	down, stopped, unseen, missing, vms := 0, 0, 0, 0, 0
	for _, h := range macs {
		vms += len(h.VMs)
		switch MacStandingOf(h) {
		case MacAsleep:
			continue
		case MacNotSeen:
			unseen++
			continue
		case MacEngineStopped:
			stopped++
		case MacEngineNotInstalled:
			missing++
		case MacOnline:
		}
		for _, vm := range h.VMs {
			if StandingOf(vm) == StandingDown {
				down++
			}
		}
	}
	switch {
	case down > 0:
		return RollUp{State: RollUpDown, N: down}
	case stopped > 0:
		return RollUp{State: RollUpEngineStopped, N: stopped}
	case unseen > 0:
		return RollUp{State: RollUpNotSeen, N: unseen}
	case missing > 0:
		return RollUp{State: RollUpEngineNotInstalled, N: missing}
	case vms == 0:
		return RollUp{State: RollUpEmpty}
	}
	return RollUp{State: RollUpFine}
}
