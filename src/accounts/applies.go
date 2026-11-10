package accounts

import (
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
)

var Applies = section.Shown(section.Accounts)

func ShareNeeds() []machine.Need {
	// value: a share copies an account out of this Mac's credential store.
	return []machine.Need{machine.Credentials}
}

var SharedFrom machine.Applies = func(p machine.Profile) machine.Verdict {
	// invariant: a share is a Mac's: its accounts are copied into its own VMs.
	if p.Kind == machine.Mac {
		return machine.Yes
	}
	return machine.No("accounts are shared from a Mac")
}

var SharedInto machine.Applies = func(p machine.Profile) machine.Verdict {
	// invariant: a share lands only in a machine that keeps accounts: a
	// server keeps its own secrets, and station shares none with it.
	if Applies(p).OK {
		return machine.Yes
	}
	return machine.No(p.Ref.Name + " keeps its own secrets")
}
