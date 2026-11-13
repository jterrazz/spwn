package awake

import (
	"slices"
	"time"
)

type Mode string

const (
	SleepsNormally Mode = "sleeps-normally"
	AwakeUntil     Mode = "awake-until"
	NeverSleeps    Mode = "never-sleeps"
	StoppedByGuard Mode = "stopped-by-guard"
)

const (
	// value: how long a guard's end is told, a finding, before the Mac
	// reads Sleeps normally again.
	StoppedFor = 10 * time.Minute
)

func (e *End) ByGuard() bool {
	return e != nil && slices.Contains([]string{Thermal, Battery, LowPower, Lid}, e.Reason)
}

func ModeOf(now time.Time, never bool, s *Session, e *End) Mode {
	// invariant: Never sleeps wins over a keep-awake, which wins over the
	// guard's end it may follow; a guard's end is told for StoppedFor.
	switch {
	case never:
		return NeverSleeps
	case s != nil:
		return AwakeUntil
	case e.ByGuard() && now.Sub(e.At) < StoppedFor:
		return StoppedByGuard
	}
	return SleepsNormally
}
