package machines

type Gesture string

const (
	GestureStart   Gesture = "start"
	GestureStop    Gesture = "stop"
	GestureRestart Gesture = "restart"
)

func (Gesture) Enum() []string {
	return []string{string(GestureStart), string(GestureStop), string(GestureRestart)}
}

func GesturesOf(vm VM) []Gesture {
	// invariant: what a machine's header offers is what its state allows: a
	// running one stops, a running server also restarts, a stopped one
	// starts; a VM in any other state offers none.
	switch vm.State {
	case Running:
		if vm.Lifetime == AlwaysOn {
			return []Gesture{GestureStop, GestureRestart}
		}
		return []Gesture{GestureStop}
	case Stopped, Suspended:
		return []Gesture{GestureStart}
	case Stale, Broken, UnknownState, Creating:
	}
	return nil
}
