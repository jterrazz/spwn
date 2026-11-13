package catalog

type State string

const (
	StateOK       State = "ok"
	StateDegraded State = "degraded"
	StateAbsent   State = "absent"
	StateBroken   State = "broken"
	StateUnknown  State = "unknown"
	// value: a part whose probe has not answered yet; it never outlives a reading.
	StatePending State = "pending"
)
