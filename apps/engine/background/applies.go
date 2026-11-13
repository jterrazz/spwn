package background

import (
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

var Applies = machine.All(section.Shown(section.Background), managed)

func managed(p machine.Profile) machine.Verdict {
	if !p.Caps.Jobs {
		return machine.No(p.Ref.Name + " has no job manager station reads")
	}
	return machine.Yes
}
