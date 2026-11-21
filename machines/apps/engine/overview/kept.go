package overview

import "time"

type Kept struct {
	// value: an Overview as the engine's store holds it (ADR-070): its facts
	// only, so two readings of an unchanged machine are equal. The head says
	// no uptime and no line made of it, Read is zero, and Boot is the instant
	// the machine started, zero where unknown.
	Summary Summary
	Boot    time.Time `json:",omitzero"`
}

func KeptOf(s Summary) Kept {
	// invariant: the boot instant is the reading's instant less its uptime,
	// to the second, as the meter reads boot; the reading's own instant is
	// the slice's, never the value's.
	var boot time.Time
	if s.Head.Up > 0 && !s.Read.IsZero() {
		boot = s.Read.Add(-s.Head.Up).Round(time.Second)
	}
	s.Head.Up, s.Read = 0, time.Time{}
	s.Head.Verdict = HeadOf(s)
	return Kept{Summary: s, Boot: boot}
}
