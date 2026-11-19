package machines

type Gesture string

const (
	GestureStart   Gesture = "start"
	GestureStop    Gesture = "stop"
	GestureRestart Gesture = "restart"
	// value: the start of a VM whose create failed, worded apart from the
	// start of a VM its owner stopped: it finishes a create, it does not
	// resume a VM.
	GestureTryAgain Gesture = "try-again"
)

func (Gesture) Enum() []string {
	return []string{string(GestureStart), string(GestureStop), string(GestureRestart), string(GestureTryAgain)}
}

func GesturesOf(vm VM) []Gesture {
	// invariant: what a machine's header offers is what its state allows: a
	// running one stops, a running server also restarts, a stopped one
	// starts, a broken server stops or restarts; a VM in any other state
	// offers none. A VM whose create failed offers try-again whatever its
	// state: that start finishes its create.
	if vm.Birth != nil && vm.Birth.Failed {
		return []Gesture{GestureTryAgain}
	}
	switch vm.State {
	case Running:
		if vm.Lifetime == AlwaysOn {
			return []Gesture{GestureStop, GestureRestart}
		}
		return []Gesture{GestureStop}
	case Stopped, Suspended:
		return []Gesture{GestureStart}
	case Broken:
		if vm.Lifetime == AlwaysOn {
			return []Gesture{GestureStop, GestureRestart}
		}
	case Stale, UnknownState, Creating:
	}
	return nil
}
