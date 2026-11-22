package awake

import (
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

const DefaultFor = 4 * time.Hour

type Request struct {
	Lid     bool
	Battery bool
	For     time.Duration
	// value: the term given by its end (until 17:00): it wins over For, so
	// a term picked from a page read a while ago still ends at its hour.
	Until time.Time
	By    string
	// value: the term as the owner said it (4h, until 18:00), for the words
	// of its refusal.
	Words string
}

type Session struct {
	Lid       bool      `json:"lid"`
	Battery   bool      `json:"battery"`
	StartedAt time.Time `json:"started_at"`
	Until     time.Time `json:"until"`
	By        string    `json:"by"`
	// value: the term the owner asked for last (1 h, 4 h, or the time
	// left until the hour picked): what Keep awake for… shows chosen.
	For time.Duration `json:"for,omitempty"`
}

func (s Session) Asked() time.Duration {
	// value: the term a session was asked for: its own word, else its span
	// from its start, to the minute.
	if s.For > 0 {
		return s.For.Round(time.Minute)
	}
	return s.Until.Sub(s.StartedAt).Round(time.Minute)
}

type End struct {
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
	Detail string    `json:"detail"`
}

type GuardState struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type View struct {
	Session *Session     `json:"session"`
	LastEnd *End         `json:"last_end"`
	Reading Reading      `json:"reading"`
	Guards  []GuardState `json:"guards"`
	// value: the engine holds Never sleeps, and since when it has held it.
	NeverSleeps bool      `json:"never_sleeps"`
	NeverSince  time.Time `json:"never_since,omitzero"`
	// value: the policy's assertion is held now; false while the battery
	// guard lets it go.
	NeverHeld bool `json:"never_held"`
	// value: SPEC B5.15: macOS told this Mac it is going to sleep, and it
	// has not woken since.
	Sleeping bool `json:"sleeping,omitempty"`
}

type Slept struct {
	Ended bool          `json:"ended"`
	After time.Duration `json:"after"`
	Mac   string        `json:"mac"`
	// value: the Mac never sleeps, a setting off leaves as it is.
	NeverSleeps bool `json:"never_sleeps"`
}

// value: a Report's Daemon while its engine answers, and before it has.
const (
	EngineRunning = "running"
	EngineReading = "pending"
)

type Report struct {
	Now time.Time
	// value: the zone a page shows its clock times in (the owner's, read once
	// by the caller); nil is UTC.
	Zone    *time.Location
	Engine  string
	Reading Reading
	Session *Session
	LastEnd *End
	Guards  []GuardState
	// value: the Mac this report is of, This Mac's first; its reported
	// capabilities and the servers it hosts.
	Mac         string
	Caps        machine.Capabilities
	NeverSleeps bool
	NeverSince  time.Time
	NeverHeld   bool
	// value: the Mac's own sleep timer is off (pmset sleep 0), as System
	// Settings sets it, whatever the engine holds.
	MacNeverSleeps bool
	Servers        []string
	// value: SPEC B5.14: the workspaces running on it now, which pause
	// while it sleeps; a stopped one is not listed.
	Pauses []string
	// value: the terms Keep awake for… offers, each one awake.keep takes.
	Choices []Choice
	// value: how this Mac stands awake, VerdictOf's; nil while its engine
	// reads.
	Verdict *verdict.Verdict `json:",omitempty"`
}

func (r Report) EngineDown() bool {
	// value: the engine is stopped or not installed: it neither runs nor
	// reads.
	return r.Engine != "" && r.Engine != EngineRunning && r.Engine != EngineReading
}

func (r Report) Until() string {
	// value: the hour the session ends, in the words the Awake page says it.
	s := r.Session
	if s == nil {
		return ""
	}
	zone := r.Zone
	if zone == nil {
		zone = time.UTC
	}
	return "until " + s.Until.In(zone).Format("15:04")
}

func (r Report) Phrase() string { return strings.Join(r.Clauses(), ", ") }

func (r Report) Clauses() []string {
	// value: how this Mac sleeps, in the words its Awake line says, clause
	// by clause: "never sleeps" and "let go while the battery is low" while
	// the battery guard lets the policy go, "awake until 18:00" and "lid
	// closed" when the session holds it so, "awake stopped" with the guard's
	// words while that end is told, else "never sleeps" and "set in System
	// Settings" while the Mac's own sleep timer is off, else "sleeps
	// normally", as with its engine down.
	bySettings := []string{"never sleeps", "set in System Settings"}
	if r.EngineDown() {
		if r.MacNeverSleeps {
			return bySettings
		}
		return []string{"sleeps normally"}
	}
	switch r.Mode() {
	case NeverSleeps:
		if !r.NeverHeld {
			return []string{"never sleeps", "let go while the battery is low"}
		}
		return []string{"never sleeps"}
	case AwakeUntil:
		if r.Session.Lid {
			return []string{"awake " + r.Until(), "lid closed"}
		}
		return []string{"awake " + r.Until()}
	case StoppedByGuard:
		if r.LastEnd.Detail != "" {
			return []string{"awake stopped", r.LastEnd.Detail}
		}
		return []string{"awake stopped"}
	case SleepsNormally:
		if r.MacNeverSleeps {
			return bySettings
		}
	}
	return []string{"sleeps normally"}
}

func (r Report) Keeping() *Session {
	// value: the keep-awake that holds this Mac now; nil without one, and
	// under Never sleeps, which wins over it.
	if r.Mode() != AwakeUntil {
		return nil
	}
	return r.Session
}

func (r Report) Mode() Mode { return ModeOf(r.Now, r.NeverSleeps, r.Session, r.LastEnd) }

type Awoken struct {
	Session
	Refreshed bool   `json:"refreshed"`
	Mac       string `json:"mac"`
	// value: the Mac never sleeps: the keep-awake adds nothing.
	NeverSleeps bool `json:"never_sleeps"`
}
