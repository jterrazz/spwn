package accounts

import (
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/verdict"
)

type Program string

func (Program) Enum() []string {
	names := make([]string, 0, len(catalog.LoginPrograms))
	for _, t := range catalog.LoginPrograms {
		names = append(names, t.Name)
	}
	return names
}

func Programs() []Program {
	programs := make([]Program, 0, len(catalog.LoginPrograms))
	for _, t := range catalog.LoginPrograms {
		programs = append(programs, Program(t.Name))
	}
	return programs
}

func (t Program) Label() string {
	for _, known := range catalog.LoginPrograms {
		if known.Name == string(t) {
			return known.Label
		}
	}
	return string(t)
}

func (t Program) Known() bool {
	for _, known := range catalog.LoginPrograms {
		if known.Name == string(t) {
			return true
		}
	}
	return false
}

type ProgramState string

const (
	ProgramSignedIn  ProgramState = "signed-in"
	ProgramSignedOut ProgramState = "signed-out"
	ProgramBroken    ProgramState = "broken"
	ProgramReading   ProgramState = "reading"
	ProgramWaiting   ProgramState = "waiting"
)

func (ProgramState) Enum() []string {
	return []string{string(ProgramSignedIn), string(ProgramSignedOut), string(ProgramBroken), string(ProgramReading), string(ProgramWaiting)}
}

type ProgramHow string

const (
	HowOwn     ProgramHow = "own"
	HowGranted ProgramHow = "granted"
)

func (ProgramHow) Enum() []string { return []string{string(HowOwn), string(HowGranted)} }

type ProgramAction string

const (
	ActionSignIn      ProgramAction = "sign-in"
	ActionSignInAgain ProgramAction = "sign-in-again"
	ActionSignOut     ProgramAction = "sign-out"
	ActionRemove      ProgramAction = "remove"
	ActionNone        ProgramAction = "none"
)

func (ProgramAction) Enum() []string {
	return []string{string(ActionSignIn), string(ActionSignInAgain), string(ActionSignOut), string(ActionRemove), string(ActionNone)}
}

type ProgramRow struct {
	Program Program
	State   ProgramState
	Who     string     `json:",omitempty"`
	Scopes  []string   `json:",omitempty"`
	Plan    string     `json:",omitempty"`
	How     ProgramHow `json:",omitempty"`
	// value: the vault entry a granted sign-in came from.
	Entry string    `json:",omitempty"`
	Until time.Time `json:",omitzero"`
	// invariant: a row never offers an action the engine would refuse: where
	// it would, the action is none and Line says what to do instead.
	Action ProgramAction
	Line   string `json:",omitempty"`
	// value: the question the engine asks before sign-out or remove.
	Ask string `json:",omitempty"`
}

type ProgramsRead struct {
	Verdict  verdict.Verdict
	Programs []ProgramRow
}

type StepKind string

const (
	StepDeviceCode StepKind = "device-code"
	StepLink       StepKind = "link"
	StepPaste      StepKind = "paste"
)

func (StepKind) Enum() []string {
	return []string{string(StepDeviceCode), string(StepLink), string(StepPaste)}
}

type Step struct {
	Kind StepKind
	// value: the one-time code the owner types at Site, for a device code.
	Code string `json:",omitempty"`
	// value: the site the owner is sent to, never a URL.
	Site string `json:",omitempty"`
	// value: an opaque id of the login exchange, never a URL.
	Exchange string
	Expires  time.Time
	// value: the engine's clock when the step was sent, so a client counts
	// down to Expires without its own.
	Now time.Time
	// value: the engine's sentence for this kind of step.
	Line string
	// value: the engine's sentence once Expires has passed.
	Expired string
}

type LoginRead struct {
	Step     *Step        `json:",omitempty"`
	Programs []ProgramRow `json:",omitempty"`
}

type Opened struct {
	Site string
}

type Answered struct{}

type Facts struct {
	// value: what a machine's own read says of one program, and what the engine
	// itself knows of how it got there.
	Program Program
	State   ProgramState
	Who     string
	Scopes  []string
	Plan    string
	How     ProgramHow
	Entry   string
	Until   time.Time
}

type Where struct {
	// value: where a row is read, as its engine knows it.
	Kind    machine.Kind
	Stopped bool
	// value: the Mac whose VM it is, when that is not this Mac.
	FarMac string
}

func Describe(f Facts, w Where) ProgramRow {
	// invariant: Action and Line follow from the facts and the place alone,
	// the one rule every surface reads.
	row := ProgramRow{Program: f.Program, State: f.State, Who: f.Who, Scopes: f.Scopes, Plan: f.Plan, How: f.How, Entry: f.Entry, Until: f.Until}
	switch {
	case w.Kind == machine.Server:
		row.Action, row.Line = ActionNone, "A server keeps its own sign-ins"
	case w.Stopped:
		row.Action, row.Line = ActionNone, "Start the machine first"
	case w.FarMac != "":
		row.Action, row.Line = ActionNone, "Sign in from "+w.FarMac
	case f.State == ProgramReading || f.State == ProgramWaiting:
		row.Action, row.Line = ActionNone, "Reading"
	case f.State == ProgramSignedOut:
		row.Action = ActionSignIn
	case f.State == ProgramBroken:
		row.Action = ActionSignInAgain
	case f.How == HowGranted:
		row.Action, row.Ask = ActionRemove, "Remove "+f.Program.Label()+"? The grant ends here; the token stays yours."
	default:
		row.Action, row.Ask = ActionSignOut, "Sign out of "+f.Program.Label()+"? The token stays valid at "+f.Program.Label()+"."
	}
	return row
}

func Summarize(rows []ProgramRow, name string, w Where) verdict.Verdict {
	// invariant: the head of a machine's sign-ins, one sentence and one tone, from its rows and its place.
	switch {
	case w.Kind == machine.Server:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: name + " is a server: it takes no grant and keeps its own sign-ins"}
	case w.Stopped:
		return verdict.Verdict{Tone: verdict.Warn, Sentence: name + " is stopped: its sign-ins are read when it runs"}
	}
	in := 0
	for _, r := range rows {
		if r.State == ProgramSignedIn {
			in++
		}
	}
	switch {
	case in == len(rows):
		return verdict.Verdict{Tone: verdict.OK, Sentence: "Signed in to every program"}
	case in == 0:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: "Signed in to no program"}
	}
	return verdict.Verdict{Tone: verdict.Plain, Sentence: "Signed in to " + strconv.Itoa(in) + " of " + strconv.Itoa(len(rows)) + " programs"}
}
