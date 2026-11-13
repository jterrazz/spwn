package machines

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Lifetime string

const (
	Ephemeral  Lifetime = "ephemeral"
	Persistent Lifetime = "persistent"
	// value: a server's: it starts with its Mac and is never thrown away.
	AlwaysOn Lifetime = "always-on"
)

// value: the words a station older than ADR-032 says for Ephemeral and
// Persistent: read for one release, written only to a peer that reads no
// other (remote/api's OlderWords); words, not lifetimes.
const (
	OlderEphemeral  = "throwaway"
	OlderPersistent = "permanent"
)

func (l Lifetime) Kind() machine.Kind {
	// invariant: the lifetime is the one record of a VM's kind: always on is
	// a server, every other a workspace.
	if l == AlwaysOn {
		return machine.Server
	}
	return machine.Workspace
}

func (l Lifetime) Word() string {
	if l == AlwaysOn {
		return "always on"
	}
	return string(l)
}

func (l *Lifetime) UnmarshalText(text []byte) error {
	// value: alias of the lifetime words of v2 (temporary, lasting) and of
	// v3 (throwaway, permanent), read for one release and written in
	// ephemeral and persistent.
	// invariant: a record or a request of an older station reads in today's
	// words; the next write keeps the new word.
	switch v := Lifetime(text); v {
	case "temporary", OlderEphemeral:
		*l = Ephemeral
	case "lasting", OlderPersistent:
		*l = Persistent
	default:
		*l = v
	}
	return nil
}

type Size struct {
	CPUs         int
	Memory, Disk int64
}

type Network struct {
	// value: on its Mac's network, through the Mac's default interface;
	// false keeps it behind its Mac.
	Bridged bool
}

type EndReason string

const (
	GuestPoweroff EndReason = "poweroff"
	GuestReboot   EndReason = "reboot"
	Crashed       EndReason = "crashed"
	Unknown       EndReason = "unknown"
)

type RunEnd struct {
	Reason   EndReason
	ExitCode int
	Detail   string
	// value: SPEC B5.5: the last lines its run wrote, as LastLines keeps them.
	LastLines []string
}

const (
	rerunWindow = 60 * time.Second
	rerunBudget = 2
)

type Action int

const (
	ActRerun Action = iota
	ActDelete
	ActStopped
	ActBroken
	ActStale
	// value: a server's run that ended unexpectedly: it boots again after
	// its wait (Restarts.After).
	ActRestart
	// value: a server something inside powered off: it stays stopped, its
	// wanted state stopped, and the owner is told.
	ActPoweredOff
)

func BrokenDetail() string { return "rebooting in a loop" }

func AfterRun(l Lifetime, end RunEnd, reruns []time.Time, now time.Time) (Action, []time.Time) {
	// invariant: a workspace never restarts by itself (SP-4). A poweroff its
	// guest declared deletes an ephemeral one and stops a persistent one; a
	// reboot from inside runs it again, and the third boot within 60 s that
	// ends in a reboot is a loop, stopped (SPEC §7.3); a crash or
	// an end nobody can read keeps it, stale, for the owner to start or delete.
	// A server always comes back, unless something inside it powered it off:
	// a crash or an unread end restarts it, a poweroff keeps it stopped.
	switch end.Reason {
	case GuestPoweroff:
		switch l {
		case Ephemeral:
			return ActDelete, reruns
		case AlwaysOn:
			return ActPoweredOff, reruns
		case Persistent:
		}
		return ActStopped, reruns
	case GuestReboot:
	case Crashed, Unknown:
		if l == AlwaysOn {
			return ActRestart, reruns
		}
		return ActStale, reruns
	default:
		return ActStale, reruns
	}
	recent := reruns[:0:0]
	for _, at := range reruns {
		if now.Sub(at) < rerunWindow {
			recent = append(recent, at)
		}
	}
	if len(recent) >= rerunBudget {
		return ActBroken, recent
	}
	return ActRerun, append(recent, now)
}

type Convergence int

const (
	// value: a run outlived the engine: adopted, never touched.
	Adopt Convergence = iota
	// value: a server wanted running: it starts with its Mac's engine.
	Boot
	// value: a workspace wanted running whose run is gone: stopped, kept.
	EndedWithItsMac
	Stay
)

func Converge(l Lifetime, wantRunning, live bool) Convergence {
	// invariant: at an engine's start a live run is adopted; with none, a
	// server wanted running boots (always on), a workspace never starts by
	// itself: one its record wanted running ended with its Mac.
	switch {
	case live:
		return Adopt
	case wantRunning && l == AlwaysOn:
		return Boot
	case wantRunning:
		return EndedWithItsMac
	}
	return Stay
}
