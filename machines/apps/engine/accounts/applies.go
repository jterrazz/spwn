package accounts

import (
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

var Applies = section.Shown(section.Accounts)

var OnAMac machine.Applies = func(p machine.Profile) machine.Verdict {
	// invariant: a vault, and the engine that opens a link for a sign-in, are a Mac's.
	if p.Kind == machine.Computer {
		return machine.Yes
	}
	return machine.No("passwords and grants are a Mac's")
}
