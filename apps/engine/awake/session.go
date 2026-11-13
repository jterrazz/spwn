package awake

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
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
	// value: the VM whose agent the keep-awake follows: it ends when that
	// agent leaves working, at the latest after Longest.
	Run string
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
	// value: the VM whose agent it follows (until linux-1 is done).
	Run string `json:"run,omitempty"`
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
	ThisMac     bool
	Caps        machine.Capabilities
	NeverSleeps bool
	NeverSince  time.Time
	NeverHeld   bool
	Servers     []string
	// value: SPEC B5.14: the workspaces running on it now, which pause
	// while it sleeps; a stopped one is not listed.
	Pauses []string
}

func (r Report) Mode() Mode { return ModeOf(r.Now, r.NeverSleeps, r.Session, r.LastEnd) }

type Awoken struct {
	Session
	Refreshed bool   `json:"refreshed"`
	Mac       string `json:"mac"`
	// value: the Mac never sleeps: the keep-awake adds nothing.
	NeverSleeps bool `json:"never_sleeps"`
}
