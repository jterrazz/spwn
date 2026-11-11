package settings

import (
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

var Applies = section.Shown(section.Settings)

var ReadsAllMachines = machine.All(Applies, fromAMac)

func fromAMac(p machine.Profile) machine.Verdict {
	// invariant: the matrix is read from a Mac: the one kind that reaches
	// every other machine.
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " is a " + p.Kind.Word(p.System.OS) + ": the settings of every machine are read from a Mac")
	}
	return machine.Yes
}
