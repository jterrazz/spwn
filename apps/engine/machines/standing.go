package machines

import "github.com/jterrazz/studio-station/apps/engine/core/verdict"

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
