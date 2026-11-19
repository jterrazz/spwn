package awake

import (
	"errors"
	"time"
)

const (
	EngineStopped = "stopped"
	EngineAbsent  = "not installed"
	EnginePending = EngineReading
)

var ErrEngineDown = errors.New("the engine does not answer")

const (
	ReasonThermal  = Thermal
	ReasonBattery  = Battery
	ReasonLowPower = LowPower
	ReasonLid      = Lid
	ReasonUntil    = "until"
	ReasonSleep    = "sleep"
	ReasonStop     = "stop"
	ReasonQuit     = "quit"
)

const (
	GuardOK   = "ok"
	GuardNear = "near"
)

type RangeError struct{ Long bool }

func (e *RangeError) Error() string {
	if e.Long {
		return "an Awake session lasts at most 24h"
	}
	return "an Awake session lasts at least 1m"
}

type GuardError struct {
	Guard string
	Words string
}

func (e *GuardError) Error() string { return "Awake stays off: " + e.Words }

type UnsupportedError struct{}

func (*UnsupportedError) Error() string { return "Awake runs on a Mac" }

func CheckFor(d time.Duration) error {
	switch {
	case d > Longest:
		return &RangeError{Long: true}
	case d < Shortest:
		return &RangeError{}
	}
	return nil
}

func GuardStates(r Reading, s *Session) []GuardState {
	state := func(near bool) string {
		if near {
			return GuardNear
		}
		return GuardOK
	}
	return []GuardState{
		{Name: ReasonBattery, State: state(r.OnBattery() && r.Percent <= BatteryWarns)},
		{Name: ReasonLowPower, State: state(false)},
		{Name: ReasonThermal, State: state(r.Thermal == ThermalHeavy)},
		{Name: ReasonLid, State: state(s != nil && s.Lid && s.Battery && r.LidClosedOnBattery())},
	}
}

type OffMacError struct{}

func (*OffMacError) Error() string { return "Awake runs on a Mac; here is a VM" }

type EngineDownError struct {
	Mac       string
	Installed bool
}

func (e *EngineDownError) Error() string {
	if e.Installed {
		return "the engine is stopped on " + e.Mac
	}
	return "the engine is not installed on " + e.Mac
}

type Kept struct {
	// value: what an engine keeps of Awake between two commands of a
	// stepped engine, as a real one keeps it in its memory.
	Session    *Session  `json:"session,omitempty"`
	LastEnd    *End      `json:"last_end,omitempty"`
	NeverSince time.Time `json:"never_since,omitzero"`
	NeverTrip  string    `json:"never_trip,omitempty"`
	HeavySince time.Time `json:"heavy_since,omitzero"`
	// value: the lid as the last tick read it, so a close is seen once.
	Lid string `json:"lid,omitempty"`
	// value: the policy's assertion was held when the engine paused.
	NeverHeld bool `json:"never_held,omitempty"`
	// value: the name the session's assertion is held under, which a
	// refresh keeps: the next engine takes the same one again.
	Held string `json:"held,omitempty"`
}
