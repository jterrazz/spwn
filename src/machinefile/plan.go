package machinefile

import (
	"slices"
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
// condition, its state, the method that holds it, the version there and
// the words of its read.

type Seen struct {
	Condition Condition
	State     catalog.State
	Method    string
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

// value: an install of the ledger the Machinefile lists no longer, as a
// line: its step is the item by the ledger's method, with what its removal
// asks of the owner and the extras it waits on, what it reads now, and
// what `apply --remove` does with it: remove it, or leave it.

type ExtraLine struct {
	Line
	Installed Installed
}

// value: what the plan asks of the owner, grouped by need, with the items
// that ask it.

type Asked struct {
	Need  catalog.Need
	Items []string
}

// value: a Machinefile against the machine: what it asks of the owner,
// each line in the order apply runs it, what station installed that it
// lists no longer, in the order `apply --remove` takes it back, what is
// skipped, and what has to happen once it is applied, each once.

type Plan struct {
	Name     string
	Needs    []Asked
	Lines    []Line
	Extra    []ExtraLine
	Skipped  []Skip
	Restarts []catalog.AfterApply
}

func Judge(r Resolved, cat catalog.Catalog, seen map[string]Seen, ledger []Installed) Plan {
	// invariant: an item's action follows from what it reads: absent is
	// installed or set, present kept, present by another way adopted and
	// never reinstalled, an app that no longer answers installed again, a
	// setting changed by hand or a read that did not answer left as it is
	// (the owner's to decide); an item of another phase is left to it.
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
	var extra []Installed
	for _, in := range ledger {
		if in.Machinefile == r.Name && !listed[in.Item] {
			extra = append(extra, in)
		}
	}
	p.Extra = extras(r.System, cat, seen, extra)
	p.Needs = p.Interactive(false)
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

// invariant: only what the ledger says station installed for this
// Machinefile is extra, and of it only what reads present or outdated by
// the method that installed it is removed: an extra already gone is
// removed from the ledger alone, and one present another way since (put
// back by hand, held by another method) is the owner's and left. The
// extras run in reverse dependency order: what another extra waits on
// (its manager, what it requires) is taken back after it.
func extras(sys catalog.System, cat catalog.Catalog, seen map[string]Seen, extra []Installed) []ExtraLine {
	out := make([]ExtraLine, 0, len(extra))
	for _, in := range extra {
		s := ExtraLine{Installed: in, Line: Line{Step: Step{Item: in.Item, Kind: catalog.KindApp, Method: in.Method, Phase: catalog.PhaseApply}, Seen: seen[in.Item]}}
		if s.Seen.Condition == "" {
			s.Seen.Condition = Unknown
		}
		s.Action = Leave
		switch s.Seen.Condition {
		case Present, Outdated:
			if s.Seen.Method == in.Method {
				s.Action = Remove
			}
		case Absent:
			s.Action = Remove
		case Changed, NotManaged, Extra, Unknown:
		}
		if it, ok := cat.Item(in.Item); ok {
			s.Step.Needs = removalNeeds(it.Item, in.Method, sys)
			s.Step.Waits = slices.Clone(it.Item.Requires)
			if owner, ok := managerOf(cat, in.Method); ok && owner != in.Item {
				s.Step.Waits = append(s.Step.Waits, owner)
			}
		}
		out = append(out, s)
	}
	return reversed(out)
}

// value: what removing an item by a method asks of the owner: the item's
// needs and the method's own (a cask's sudo); an extra already gone asks
// nothing, and Plan.Interactive judges that.
func removalNeeds(it catalog.Item, method string, sys catalog.System) []catalog.Need {
	needs := slices.Clone(it.Needs)
	for _, m := range it.Methods {
		if m.Key() == method && m.Matches(sys) {
			return append(needs, m.Needs...)
		}
	}
	return needs
}

func managerOf(cat catalog.Catalog, method string) (string, bool) {
	for _, it := range cat.Items {
		if slices.Contains(it.Item.Manages, catalog.ManagerKey(method)) {
			return it.Item.Name, true
		}
	}
	return "", false
}

// invariant: a stable order: the first extra (by name) that no remaining
// extra waits on runs next; a loop, which the catalog's own refusal of a
// cycle rules out, keeps the rest in name order.
func reversed(extra []ExtraLine) []ExtraLine {
	var out []ExtraLine
	left := slices.Clone(extra)
	for len(left) > 0 {
		next := slices.IndexFunc(left, func(s ExtraLine) bool {
			return !slices.ContainsFunc(left, func(o ExtraLine) bool { return slices.Contains(o.Step.Waits, s.Step.Item) })
		})
		if next < 0 {
			return append(out, left...)
		}
		out = append(out, left[next])
		left = slices.Delete(left, next, next+1)
	}
	return out
}

func asked(lines []Line) []Asked {
	var out []Asked
	for _, need := range catalog.NeedSudo.Enum() {
		var items []string
		for _, l := range lines {
			if slices.Contains(l.Step.Needs, catalog.Need(need)) && !slices.Contains(items, l.Step.Item) {
				items = append(items, l.Step.Item)
			}
		}
		if len(items) > 0 {
			out = append(out, Asked{Need: catalog.Need(need), Items: items})
		}
	}
	return out
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

func (s ExtraLine) Takes() bool {
	// value: whether `apply --remove` uninstalls the extra: it reads there
	// by the method that installed it; one already gone only leaves the
	// ledger.
	return s.Action == Remove && s.Seen.Condition != Absent
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
	return !slices.ContainsFunc(p.Lines, Line.Acts) && !slices.ContainsFunc(p.Extra, func(s ExtraLine) bool { return s.Action == Remove })
}

func (p Plan) Interactive(remove bool) []Asked {
	// value: the needs a run off a terminal cannot answer: every need asks
	// the owner something (a password, a license, a sign-in, a question):
	// those of each install and setting set, and with --remove those of
	// each extra it uninstalls.
	var lines []Line
	for _, l := range p.Lines {
		if l.Acts() {
			lines = append(lines, l)
		}
	}
	for _, s := range p.Extra {
		if remove && s.Takes() {
			lines = append(lines, s.Line)
		}
	}
	return asked(lines)
}

// value: why a line of a run ended as it did, when its words are the
// surface's: the run was interrupted at it, or stopped before it (the
// rest of an interrupted run, a step whose requirement failed).

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
// order it ran, each extra removed or left, the Machinefile bound when the
// run bound one, and whether it was interrupted (an item may have finished
// just before).

type Applied struct {
	Plan        Plan
	Done        []Done
	Removed     []Done
	Bound       string
	Interrupted bool
}
