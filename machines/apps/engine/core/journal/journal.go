package journal

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/finding"
)

type Outcome string

const (
	Done   Outcome = "done"
	Failed Outcome = "failed"
	Held   Outcome = "held"
)

func (Outcome) Enum() []string { return []string{string(Done), string(Failed), string(Held)} }

type Kind string

const (
	App     Kind = "app"
	Machine Kind = "machine"
	Setting Kind = "setting"
	Skill   Kind = "skill"
	Image   Kind = "image"
)

func (Kind) Enum() []string {
	return []string{string(App), string(Machine), string(Setting), string(Skill), string(Image)}
}

const (
	// value: how long a Mac keeps a line: the server strip's thirty days.
	Horizon = 30 * 24 * time.Hour
	// value: the days a read covers when it names none, and the most it may.
	Days     = 7
	MostDays = 30
	// value: the form of a day on the wire, a group's Day and a read's Before.
	DayForm = "2006-01-02"
	// value: who a line was done by when no session or job names itself.
	ByYou = "you"
	// value: who an engine job's line was done by: the nightly upgrade, and
	// the engine itself for what it saw of a server.
	ByNight   = "the night"
	ByStation = "station"
)

type Subject struct {
	Kind Kind
	Name string
	// value: the thing it names no longer exists at the read; its name stays.
	Gone bool
}

type Upgrade struct{ From, To string }

type Line struct {
	// value: the Mac's clock when it was done.
	At time.Time
	// value: the machine it concerns: the Mac that keeps it or one of its VMs.
	Machine string
	// value: the call that made it; empty for an event the engine tells.
	Call string
	// value: what was done, in the engine's words ("gate-2 deleted", "node").
	Text string
	// value: the sentence on it ("8 GB freed", "download timed out"), its
	// secrets withheld.
	Detail  string
	Subject *Subject
	// value: "you", a session's name, "the night" or "station".
	By string
	// value: a create's purpose, as it was asked.
	Purpose string
	Outcome Outcome
	Upgrade *Upgrade
	// value: the call that does it again, on a failed line that may be.
	Retry *finding.Fix
	// value: the run of the night it belongs to; zero for any other line.
	Night time.Time `json:",omitzero"`
}

type Retry struct {
	Call string
	Args map[string]string
	On   string
	// value: the press's word, as the engine says it.
	Verb string
}

type ReadLine struct {
	At      time.Time
	Text    string
	Subject *Subject
	By      string
	Purpose string
	// value: the machines it concerns: one, or each a night moved alike.
	On      []string
	Outcome Outcome
	Upgrade *Upgrade
	Detail  string
	Retry   *Retry
}

type Group struct {
	// value: the day it holds, on the reading Mac's clock.
	Day   string
	Title string
	// value: what a night did, counted ("3 moved, 1 held, 1 failed"); empty
	// for a day, and for any read narrowed to a Mac, a machine or a subject.
	Sum   string
	Night bool
	Lines []ReadLine
}

type Unheard struct {
	Mac string
	// value: why it gave no line, in the owner's words.
	Why string
}

type Reading struct {
	Groups  []Group
	Unheard []Unheard
	// value: the Before of the next page, the day after the next older day
	// that holds a line; empty when none is kept.
	Next string
	// value: the engine's sentence when no group and no silent Mac is said.
	Empty string
}
