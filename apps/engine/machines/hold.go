package machines

import (
	"strconv"
	"time"
)

const IdleAfter = time.Hour // value: how long a workspace's agent is finished before it is idle.

type HoldState string

// value: what keeps a VM, and so whether it is safe to delete: always on
// (a server), held (its create still runs, or an agent or the session that
// made it still works in it), idle (its agent finished IdleAfter ago or
// more), free (its agent finished, or an ephemeral one stopped), yours (the
// owner keeps it), not seen (its Mac does not answer).
const (
	HoldAlwaysOn HoldState = "always-on"
	HoldHeld     HoldState = "held"
	HoldIdle     HoldState = "idle"
	HoldFree     HoldState = "free"
	HoldYours    HoldState = "yours"
	HoldNotSeen  HoldState = "not-seen"
)

func (HoldState) Enum() []string {
	return []string{string(HoldAlwaysOn), string(HoldHeld), string(HoldIdle), string(HoldFree), string(HoldYours), string(HoldNotSeen)}
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

func IdleFor(vm VM, now time.Time) (time.Duration, bool) {
	// value: how long a running workspace has sat since its agent finished,
	// and whether that makes it idle.
	if vm.Lifetime == AlwaysOn || vm.State != Running || !vm.Finished() || vm.Agent.Ended.IsZero() {
		return 0, false
	}
	d := now.Sub(vm.Agent.Ended)
	return d, d >= IdleAfter
}

func HoldOf(vm VM, now time.Time) Hold {
	// invariant: the first rule that matches says the hold, in this order:
	// its Mac unseen, a server, a create still running, an agent or its
	// session still at work, idle, finished or stopped, the owner's.
	session := vm.Creator != "" && vm.Creator != ByOwner
	working := vm.Agent != nil && !vm.Finished() || session && vm.Lifetime == Ephemeral && vm.Agent == nil
	idle, isIdle := IdleFor(vm, now)
	switch {
	case vm.State == UnknownState:
		return Hold{State: HoldNotSeen, Sentence: "Not seen: its Mac does not answer, so nothing says what holds it."}
	case vm.Lifetime == AlwaysOn:
		return Hold{State: HoldAlwaysOn, Sentence: "Always on. Stopping it needs its name typed."}
	case vm.State == Creating:
		return Hold{State: HoldHeld, By: holder(vm, session), Sentence: "Being created" + by(vm, session) + ". It cannot be deleted until its create ends."}
	case vm.State == Running && working:
		return Hold{State: HoldHeld, By: holder(vm, session), Sentence: heldBy(vm, session)}
	case isIdle:
		return Hold{State: HoldIdle, Sentence: "Idle for " + strconv.Itoa(int(idle/time.Hour)) + " h. Safe to delete."}
	case vm.State == Running && vm.Finished():
		return Hold{State: HoldFree, Sentence: "Its agent finished. Safe to delete."}
	case vm.Lifetime == Ephemeral && vm.State == Stopped:
		return Hold{State: HoldFree, Sentence: "Stopped. Safe to delete."}
	case vm.Lifetime == Ephemeral:
		return Hold{State: HoldYours, Sentence: "Yours. Deleted when it powers off."}
	}
	return Hold{State: HoldYours, Sentence: "Yours. Stays until you delete it."}
}

func heldBy(vm VM, session bool) string {
	// value: the sentence a delete's question carries for a VM someone works
	// in; the lot is the session that made it.
	if session {
		return "It is held by its lot, by " + vm.Creator + ": work may be in progress."
	}
	return "It is held by its agent: work may be in progress."
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
