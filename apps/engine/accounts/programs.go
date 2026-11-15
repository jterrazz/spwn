package accounts

import (
	"fmt"
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

func ProgramOf(account string) Program {
	// value: the program a provider's id names, by the catalog's
	// label; an id no program carries is its own name.
	for _, p := range catalog.Providers {
		if p.ID != account {
			continue
		}
		for _, known := range catalog.LoginPrograms {
			if known.Label == p.Label {
				return Program(known.Name)
			}
		}
	}
	return Program(account)
}

func (t Program) Label() string {
	if known, ok := t.login(); ok {
		return known.Label
	}
	return string(t)
}

func (t Program) Known() bool {
	_, ok := t.login()
	return ok
}

func (t Program) login() (catalog.LoginProgram, bool) {
	for _, known := range catalog.LoginPrograms {
		if known.Name == string(t) {
			return known, true
		}
	}
	return catalog.LoginProgram{}, false
}

func (t Program) Flow() StepKind {
	known, _ := t.login()
	switch known.Flow {
	case catalog.FlowDeviceCode:
		return StepDeviceCode
	case catalog.FlowLink:
		return StepLink
	case catalog.FlowPaste:
		return StepPaste
	}
	return StepLink
}

func (t Program) Site() string {
	known, _ := t.login()
	return known.Site
}

func (t Program) Captures() bool {
	// invariant: a program's sign-in can be taken into the vault exactly
	// when its delivery names the command that prints its token.
	d, ok := t.Delivery()
	return ok && len(d.Capture) > 0
}

func (t Program) CaptureKind() KeyKind {
	// invariant: the kind of the token a sign-in leaves is the delivery's
	// to say, empty for a program that leaves none to take.
	d, _ := t.Delivery()
	return KeyKind(d.CaptureKind)
}

func (t Program) Kinds() []KeyKind {
	// value: the kinds a token typed for the program may be, the first the default.
	d, _ := t.Delivery()
	kinds := make([]KeyKind, 0, len(d.Kinds))
	for _, k := range d.Kinds {
		kinds = append(kinds, KeyKind(k))
	}
	return kinds
}

func LoginTitle(p Program, on string) string {
	// value: the heading of a login step, the program's wire name and the
	// machine it signs in on.
	return "Sign in to " + string(p) + " on " + on
}

func (t Program) StepLine(kind StepKind, site string) string {
	// value: the status line under the code of a login step, worded for
	// the program and the site the owner is sent to.
	switch kind {
	case StepDeviceCode:
		return "Waiting for you to enter the " + t.Label() + " code at " + site
	case StepPaste:
		return "Waiting for the " + t.Label() + " code from " + site
	case StepLink:
	}
	return "Waiting for you to approve " + t.Label() + " at " + site
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
	// value: the vault entry a granted sign-in came from, and its label.
	Entry      string `json:",omitempty"`
	EntryLabel string `json:",omitempty"`
	// invariant: a row never offers an action the engine would refuse: where
	// it would, the action is none and Line says what to do instead; a
	// broken sign-in that can be signed in again says why in Line.
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

func (k StepKind) Heading() string {
	// value: what a step of the kind is headed: what the owner does next.
	switch k {
	case StepDeviceCode:
		return "Enter the code"
	case StepPaste:
		return "Paste the code"
	case StepLink:
	}
	return "Open the link"
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
	// value: the status line under the code, worded for the program.
	Line string
	// value: the heading of the step: which program signs in on which machine.
	Title string
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
	Program    Program
	State      ProgramState
	Who        string
	Scopes     []string
	Plan       string
	How        ProgramHow
	Entry      string
	EntryLabel string
	// value: why a broken sign-in no longer works, in the engine's short words.
	Reason string
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
	row := ProgramRow{Program: f.Program, State: f.State, Who: f.Who, Scopes: f.Scopes, Plan: f.Plan, How: f.How, Entry: f.Entry, EntryLabel: f.EntryLabel}
	if f.State == ProgramBroken {
		row.Line = f.Reason
		if row.Line == "" {
			row.Line = "The sign-in no longer works"
		}
	}
	switch {
	case w.Kind == machine.Server:
		row.Action, row.Line = ActionNone, lineServer
	case w.Stopped:
		row.Action, row.Line = ActionNone, lineStopped
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
		return verdict.Verdict{Tone: verdict.OK, Sentence: SentenceSignInsAll.Text()}
	case in == 0:
		return verdict.Verdict{Tone: verdict.Plain, Sentence: SentenceSignInsNone.Text()}
	}
	return verdict.Verdict{Tone: verdict.Plain, Sentence: fmt.Sprintf(SentenceSignInsSome.Text(), in, len(rows))}
}
