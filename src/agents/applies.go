package agents

import "github.com/jterrazz/studio-station/src/core/machine"

func Applies(p machine.Profile) machine.Verdict {
	// invariant: an agent's hook reports from inside the workspace it works
	// in, where its journal is kept.
	if p.Kind != machine.Workspace {
		return machine.No(p.Ref.Name + " is a " + string(p.Kind) + ": agents are recorded inside the workspace they work in")
	}
	return machine.Yes
}

func Reaches(p machine.Profile) machine.Verdict {
	// invariant: SPEC B5.22: the fleet's agents are read and started from a
	// Mac, which reaches every workspace, or inside the workspace itself; a
	// server hosts none.
	if p.Kind == machine.Mac {
		return machine.Yes
	}
	return Applies(p)
}
