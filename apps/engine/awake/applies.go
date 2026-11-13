package awake

import (
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

var Applies = machine.All(followsItsMac, section.Shown(section.Awake), engined)

func followsItsMac(p machine.Profile) machine.Verdict {
	// invariant: a VM sleeps when its Mac does: it is never listed nor
	// targeted, its Mac is.
	if p.Kind == machine.Computer {
		return machine.Yes
	}
	fix := "station awake"
	if p.Ref.On != "" {
		fix += " --on " + p.Ref.On
	}
	return machine.Verdict{Why: p.Ref.Name + " follows its Mac", Fix: fix}
}

func engined(p machine.Profile) machine.Verdict {
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " runs no engine to hold it awake")
	}
	return machine.Yes
}
