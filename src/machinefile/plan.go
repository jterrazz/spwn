package machinefile

import (
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: what one item of a Machinefile is on the machine now, as drift
// says it: there as the Machinefile asks, not there, there behind its
// newest, no longer as station left it (a setting edited by hand, an app
// that no longer answers), there by another way than its method, installed
// by station for a Machinefile that lists it no longer, or not readable now.

type Condition string

const (
	Present    Condition = "present"
	Absent     Condition = "absent"
	Outdated   Condition = "outdated"
	Changed    Condition = "changed"
	NotManaged Condition = "not-managed"
	Extra      Condition = "extra"
	Unknown    Condition = "unknown"
)

// value: what apply does with an item: install it, set it, adopt it as
// it is (never reinstalled), keep it (already there), leave it as it is
// (the owner's change, a read that did not answer), or remove it (an extra,
// with --remove alone).

type Action string

const (
	Install Action = "install"
	Set     Action = "set"
	Adopt   Action = "adopt"
	Keep    Action = "keep"
	Leave   Action = "leave"
	Remove  Action = "remove"
)

// value: what an item reads on the machine, as its family read it: its
// condition, its state, the version there and the words of its read.

type Seen struct {
	Condition Condition
	State     catalog.State
	Version   string
	Detail    string
}

// value: one line of a plan: the step resolved, what it reads now, and
// what apply does with it.

type Line struct {
	Step   Step
	Seen   Seen
	Action Action
}

// value: one install station made for a Machinefile, as its ledger keeps
// it: the item, the method that installed it, the Machinefile that listed
// it, and when.

type Installed struct {
	Item        string    `json:"item"`
	Method      string    `json:"method"`
	Machinefile string    `json:"machinefile"`
	At          time.Time `json:"at"`
}

// value: what the plan asks of the owner, grouped by need, with the items
// that ask it.

type Asked struct {
	Need  catalog.Need
	Items []string
}

// value: a Machinefile against the machine: what it asks of the owner,
// each line in the order apply runs it, what station installed that it
// lists no longer, what is skipped, and what has to happen once it is
// applied, each once.

type Plan struct {
	Name     string
	Needs    []Asked
	Lines    []Line
	Extra    []Installed
	Skipped  []Skip
	Restarts []catalog.AfterApply
}

func Judge(r Resolved, seen map[string]Seen, ledger []Installed) Plan {
	// invariant: an item's action follows from what it reads: absent is
	// installed or set, present kept, present by another way adopted and
	// never reinstalled, an app that no longer answers installed again, a
	// setting changed by hand or a read that did not answer left as it is
	// (the owner's to decide); an item of another phase is left to it. What
	// the ledger holds that the Machinefile lists no longer is extra, and
	// only that is ever removed.
	p := Plan{Name: r.Name, Skipped: r.Skipped}
	listed := map[string]bool{}
	for _, s := range r.Steps {
		listed[s.Item] = true
		l := Line{Step: s, Seen: seen[s.Item]}
		if l.Seen.Condition == "" {
			l.Seen.Condition = Unknown
		}
		l.Action = actionOf(s, l.Seen)
		if s.Phase != catalog.PhaseApply {
			l.Action = Leave
		}
		p.Lines = append(p.Lines, l)
	}
	for _, in := range ledger {
		if !listed[in.Item] {
			p.Extra = append(p.Extra, in)
		}
	}
	for _, need := range catalog.NeedSudo.Enum() {
		var items []string
		for _, l := range p.Lines {
			if l.Acts() && slices.Contains(l.Step.Needs, catalog.Need(need)) {
				items = append(items, l.Step.Item)
			}
		}
		if len(items) > 0 {
			p.Needs = append(p.Needs, Asked{Need: catalog.Need(need), Items: items})
		}
	}
	for _, l := range p.Lines {
		if !l.Acts() {
			continue
		}
		for _, r := range l.Step.Restarts {
			if !slices.Contains(p.Restarts, r) {
				p.Restarts = append(p.Restarts, r)
			}
		}
	}
	return p
}

func actionOf(s Step, seen Seen) Action {
	switch seen.Condition {
	case Absent:
		if seen.State == catalog.StateAbsent || seen.State == "" {
			if s.Kind == catalog.KindSetting {
				return Set
			}
			return Install
		}
		return Leave
	case Changed:
		if s.Kind != catalog.KindSetting && seen.State == catalog.StateBroken {
			return Install
		}
		return Leave
	case NotManaged:
		return Adopt
	case Present, Outdated:
		return Keep
	case Extra, Unknown:
	}
	return Leave
}

func (l Line) Acts() bool {
	// value: whether apply changes the machine for the line: an install or a
	// setting set.
	return l.Action == Install || l.Action == Set
}

func (p Plan) Of(a Action) []Line {
	// value: the lines of an action, in the order apply runs them.
	var out []Line
	for _, l := range p.Lines {
		if l.Action == a {
			out = append(out, l)
		}
	}
	return out
}

func (p Plan) Calm() bool {
	// value: the plan holds nothing for apply to do: no install, no setting
	// set, no extra to remove.
	return !slices.ContainsFunc(p.Lines, Line.Acts) && len(p.Extra) == 0
}

func (p Plan) Interactive() []Asked {
	// value: the needs a run off a terminal cannot answer: every need asks
	// the owner something (a password, a license, a sign-in, a question).
	return p.Needs
}

func Shown(v catalog.OptionValue) string {
	// value: an option value as a plan shows it: a `vault:` reference stays
	// the reference, never what it names.
	if ref, ok := strings.CutPrefix(string(v), catalog.VaultPrefix); ok {
		return catalog.VaultPrefix + ref
	}
	return string(v)
}

// value: why a line of a run ended as it did, when its words are the
// surface's: the run was interrupted at it, or stopped before it.

type Reason string

const (
	Interrupted Reason = "interrupted"
	NotRun      Reason = "not-run"
)

// value: one line of a run: what it was, how it ended, the version and
// the method that hold it now, its words (a failure's, the log that keeps
// its program's own), and whether this run installed it.

type Done struct {
	Line      Line
	State     catalog.State
	Version   string
	Detail    string
	Log       string
	Reason    Reason
	Installed bool
}

// value: a Machinefile applied: its plan, each line as it ended in the
// order it ran, each extra removed, and the Machinefile bound when the run
// bound one.

type Applied struct {
	Plan    Plan
	Done    []Done
	Removed []Done
	Bound   string
}
