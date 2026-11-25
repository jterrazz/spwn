package devices

import "github.com/jterrazz/studio-station/apps/engine/core/machine"

var OnAMac machine.Applies = func(p machine.Profile) machine.Verdict {
	// invariant: a device book is a Mac's: its VMs trust what their Mac's
	// book says, and keep none of their own.
	if p.Kind == machine.Computer {
		return machine.Yes
	}
	return machine.No("the device book is a Mac's")
}
