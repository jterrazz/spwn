package approvals

import "github.com/jterrazz/studio-station/apps/engine/core/machine"

var OnAMac machine.Applies = func(p machine.Profile) machine.Verdict {
	// invariant: an approval is macOS's consent to the Mac's own Spwn; a VM
	// asks none of its own.
	if p.Kind == machine.Computer {
		return machine.Yes
	}
	return machine.No("approvals are a Mac's")
}
