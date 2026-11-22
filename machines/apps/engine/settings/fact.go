package settings

import "github.com/jterrazz/studio-station/apps/engine/catalog"

type Fact struct {
	State  catalog.State
	Value  string
	Detail string
}

func Unread() Fact {
	// invariant: a fact the machine could not look at is unknown, never off: a
	// row must not claim a protection is down on a machine that could not look.
	return Fact{State: catalog.StateUnknown}
}

func NeverSleeps(system Fact) bool {
	// value: the Mac's own sleep timer, catalog.FactSystemSleep, reads off:
	// pmset sleep 0, what System Settings writes for never.
	return system.State == catalog.StateOK && system.Value == "0"
}
