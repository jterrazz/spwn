package overview

import "github.com/jterrazz/studio-station/apps/engine/core/machine"

func Applies(machine.Profile) machine.Verdict {
	// invariant: every machine has an Overview.
	return machine.Yes
}
