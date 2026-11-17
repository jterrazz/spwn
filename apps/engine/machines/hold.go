package machines

type HoldState string

// value: what keeps a VM, and so whether it is safe to delete: always on
// (a server), held (its create still runs, or the session that made it an
// ephemeral one still runs it), free (an ephemeral one stopped), yours (the
// owner keeps it), not seen (its Mac does not answer).
const (
	HoldAlwaysOn HoldState = "always-on"
	HoldHeld     HoldState = "held"
	HoldFree     HoldState = "free"
	HoldYours    HoldState = "yours"
	HoldNotSeen  HoldState = "not-seen"
)

func (HoldState) Enum() []string {
	return []string{string(HoldAlwaysOn), string(HoldHeld), string(HoldFree), string(HoldYours), string(HoldNotSeen)}
}

type Hold struct {
	State HoldState
	// value: who holds it, the session that made it, when one does; empty
	// otherwise.
	By string `json:",omitempty"`
	// value: the hold as a page says it, "Held by studio-os-fe. Not safe to
	// delete while it runs."
	Sentence string
}

func HoldOf(vm VM) Hold {
	// invariant: the first rule that matches says the hold, in this order:
	// its Mac unseen, a server, a create still running, an ephemeral one
	// its session runs, an ephemeral one stopped, the owner's.
	session := vm.Creator != "" && vm.Creator != ByOwner
	switch {
	case vm.State == UnknownState:
		return Hold{State: HoldNotSeen, Sentence: "Not seen: its Mac does not answer, so nothing says what holds it."}
	case vm.Lifetime == AlwaysOn:
		return Hold{State: HoldAlwaysOn, Sentence: "Always on. Stopping it needs its name typed."}
	case vm.State == Creating:
		return Hold{State: HoldHeld, By: holder(vm, session), Sentence: "Being created" + by(vm, session) + ". It cannot be deleted until its create ends."}
	case vm.State == Running && vm.Lifetime == Ephemeral && session:
		return Hold{State: HoldHeld, By: vm.Creator, Sentence: "It is held by its lot, by " + vm.Creator + ": work may be in progress."}
	case vm.Lifetime == Ephemeral && vm.State == Stopped:
		return Hold{State: HoldFree, Sentence: "Stopped. Safe to delete."}
	case vm.Lifetime == Ephemeral:
		return Hold{State: HoldYours, Sentence: "Yours. Deleted when it powers off."}
	}
	return Hold{State: HoldYours, Sentence: "Yours. Stays until you delete it."}
}

func holder(vm VM, session bool) string {
	if session {
		return vm.Creator
	}
	return ""
}

func by(vm VM, session bool) string {
	if session {
		return " by " + vm.Creator
	}
	return ""
}
