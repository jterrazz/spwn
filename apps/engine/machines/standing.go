package machines

import (
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
	RollUpDown    RollUpState = "down"
	RollUpNotSeen RollUpState = "not-seen"
	RollUpEmpty   RollUpState = "empty"
	RollUpFine    RollUpState = "fine"
)

func (RollUpState) Enum() []string {
	return []string{string(RollUpDown), string(RollUpNotSeen), string(RollUpEmpty), string(RollUpFine)}
}

func (s RollUpState) Tone() verdict.Tone {
	switch s {
	case RollUpDown:
		return verdict.Bad
	case RollUpNotSeen:
		return verdict.Warn
	case RollUpFine:
		return verdict.OK
	case RollUpEmpty:
	}
	return verdict.Plain
}

type RollUp struct {
	State RollUpState
	// value: how many machines its state names: the VMs down and the Macs
	// whose engine is stopped or not installed, or the Macs not seen; absent
	// for empty and fine.
	N int `json:",omitempty"`
}

func RollUpOf(macs []MacView) RollUp {
	// invariant: how a set of Macs and their VMs stands, read from their
	// standings alone, its first rule that matches: down (a VM down, or an
	// engine stopped or not installed), not seen (a Mac not seen), empty (no
	// VM), fine. A VM of a Mac that does not answer counts through its Mac
	// only; an asleep Mac is no fault.
	down, unseen, vms := 0, 0, 0
	for _, h := range macs {
		vms += len(h.VMs)
		switch MacStandingOf(h) {
		case MacAsleep:
			continue
		case MacNotSeen:
			unseen++
			continue
		case MacEngineStopped, MacEngineNotInstalled:
			down++
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
	case unseen > 0:
		return RollUp{State: RollUpNotSeen, N: unseen}
	case vms == 0:
		return RollUp{State: RollUpEmpty}
	}
	return RollUp{State: RollUpFine}
}
