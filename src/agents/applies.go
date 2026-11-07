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
