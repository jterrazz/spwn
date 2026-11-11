package machines

import "github.com/jterrazz/studio-station/src/core/machine"

var CanCreate = machine.All(runsAnEngine)

func VMNeeds() []machine.Need {
	// value: what making or booting a VM cannot run without.
	return []machine.Need{machine.Hypervisor}
}

var AddsMacs = machine.All(reachesMacs)

func reachesMacs(p machine.Profile) machine.Verdict {
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " is a " + p.Kind.Word() + ": Macs are added from a Mac")
	}
	return machine.Yes
}

func runsAnEngine(p machine.Profile) machine.Verdict {
	if !p.Caps.Engine {
		return machine.No(p.Ref.Name + " runs no engine: VMs are made on a Mac")
	}
	return machine.Yes
}
