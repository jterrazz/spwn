package ports

import (
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

var Applies = section.Shown(section.Ports)

func ShareNeeds() []machine.Need {
	// value: a share is carried to the owner's other devices by the private
	// network; judged on a Mac, which shares for its VMs.
	return []machine.Need{machine.PrivateNetwork}
}

func DeclaredOnAMac(p machine.Profile) machine.Verdict {
	// invariant: a declaration is kept by a Mac, for itself and its VMs: a
	// VM's port is declared from its Mac.
	if p.Kind == machine.Computer {
		return machine.Yes
	}
	return machine.Verdict{Why: p.Ref.Name + "'s ports are declared from its Mac", Fix: "run it on its Mac, naming " + p.Ref.Name}
}

func LinksOnAMac(p machine.Profile) machine.Verdict {
	// invariant: HTTPS links are a Mac's: it serves every share of its VMs.
	if p.Kind == machine.Computer {
		return machine.Yes
	}
	return machine.Verdict{Why: p.Ref.Name + "'s HTTPS links are set on its Mac", Fix: "run it on its Mac"}
}
