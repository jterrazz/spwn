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
// condition, its state, the method that holds it, the version there, the
// words of its read, the manager its method lacks here, when that
// manager is not installed, and whether station cannot uninstall it here.

type Seen struct {
	Condition    Condition
	State        catalog.State
	Method       string
	Version      string
	Detail       string
	Lacks        string
	Irreversible bool
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
	// value: the apps there that still need it (what requires it, what it
	// provides, what it manages), when they keep it: none of them is taken
	// back.
	RequiredBy []string
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
// skipped, what has to happen once it is applied, each once, and whether
// a line run as root asks no password here, as the machine answered it.

type Plan struct {
	Name         string
	Needs        []Asked
	Lines        []Line
	Extra        []ExtraLine
	Skipped      []Skip
	Restarts     []catalog.AfterApply
	Passwordless bool
}

func Judge(r Resolved, cat catalog.Catalog, seen map[string]Seen, ledger []Installed, passwordless bool) Plan {
	// invariant: an item's action follows from what it reads: absent is
	// installed or set, present kept, present by another way adopted and
	// never reinstalled, an app that no longer answers installed again, a
	// setting changed by hand or a read that did not answer left as it is
	// (the owner's to decide); an item of another phase is left to it. An
	// item skipped on this system is still listed, so never extra.
	p := Plan{Name: r.Name, Skipped: r.Skipped, Passwordless: passwordless}
	listed := map[string]bool{}
	for _, s := range r.Skipped {
		listed[s.Item] = true
	}
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
// the method that installed it is removed, whoever put it back there: an
// extra already gone is removed from the ledger alone, one held by another
// method is the owner's and left, one station cannot uninstall here is
// left and never asks anything, and one whose manager is missing here is
// left for a later run to judge. The extras run in reverse dependency
// order: what another extra leans on (what it requires, its provider, its
// manager) is taken back after it.
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
			if s.Seen.Method == in.Method && !s.Seen.Irreversible {
				s.Action = Remove
			}
		case Absent:
			if s.Seen.Lacks == "" {
				s.Action = Remove
			}
		case Changed, NotManaged, Extra, Unknown:
		}
		if it, ok := cat.Item(in.Item); ok {
			s.Step.Needs = removalNeeds(it.Item, in.Method, sys)
			for _, l := range cat.Links(it.Item, in.Method, sys) {
				s.Step.Waits = append(s.Step.Waits, l.Items...)
			}
		}
		out = append(out, s)
	}
	return reversed(required(sys, cat, seen, out))
}

// invariant: an extra another app there still needs (it leans on it by
// any link the resolution follows: it requires it, its method is provided
// by it, or its method's manager is it) is kept while that app is not
// taken back with it, whoever installed that app; an app whose read did
// not answer may be there, so it keeps what it needs too, and an extra
// station cannot uninstall here stays, so it keeps what it needs as well.
// What needs an extra that stays there by station's method is said, the
// one station cannot uninstall included, so its record is kept.
func required(sys catalog.System, cat catalog.Catalog, seen map[string]Seen, extra []ExtraLine) []ExtraLine {
	taken := map[string]bool{}
	for _, s := range extra {
		taken[s.Step.Item] = s.Takes()
	}
	held := map[string]bool{}
	for kept := true; kept; {
		kept = false
		for _, s := range extra {
			if taken[s.Step.Item] && len(needing(sys, cat, seen, taken, s.Step.Item)) > 0 {
				taken[s.Step.Item], held[s.Step.Item], kept = false, true, true
			}
		}
	}
	for i := range extra {
		if s := &extra[i]; held[s.Step.Item] || s.Fate() == KeptIrreversible {
			s.Action, s.RequiredBy = Leave, needing(sys, cat, seen, taken, s.Step.Item)
		}
	}
	return extra
}

// invariant: an app leans by its method as the inventory reads it, the
// one apps chose here, even for an app present another way; an app with
// no row of the inventory does not live on this machine, so holds
// nothing, and a setting's requirement holds no app.
func needing(sys catalog.System, cat catalog.Catalog, seen map[string]Seen, taken map[string]bool, item string) []string {
	var out []string
	for _, it := range cat.Items {
		name := it.Item.Name
		if name == item || taken[name] || it.Item.Kind == catalog.KindSetting || !there(seen[name]) {
			continue
		}
		if slices.ContainsFunc(cat.Links(it.Item, seen[name].Method, sys), func(l catalog.Link) bool { return l.Names(item) }) {
			out = append(out, name)
		}
	}
	return out
}

func there(s Seen) bool {
	switch s.Condition {
	case Present, Outdated, NotManaged, Changed, Unknown:
		return true
	case Absent, Extra:
	}
	return false
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

func (p Plan) asked(lines []Line) []Asked {
	var out []Asked
	for _, need := range catalog.NeedSudo.Enum() {
		if !p.Asks(catalog.Need(need)) {
			continue
		}
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

// value: what `apply --remove` does with an extra: takes it back, lets
// go of its record as already gone, or keeps it there, because an app
// there still needs it, station cannot uninstall it here, its manager is
// missing here, it is there by another way than its method, or its read
// did not answer.

type Fate string

const (
	Taken              Fate = "taken"
	Gone               Fate = "gone"
	KeptRequired       Fate = "kept-required"
	KeptIrreversible   Fate = "kept-irreversible"
	KeptManagerMissing Fate = "kept-manager-missing"
	KeptAnotherWay     Fate = "kept-another-way"
	KeptUnread         Fate = "kept-unread"
)

func (s ExtraLine) Fate() Fate {
	// invariant: what needs an extra is said before why else it stays.
	switch {
	case len(s.RequiredBy) > 0:
		return KeptRequired
	case s.Takes():
		return Taken
	case s.Action == Remove:
		return Gone
	case s.Seen.Lacks != "":
		return KeptManagerMissing
	}
	switch s.Seen.Condition {
	case Present, Outdated:
		if s.Seen.Method == s.Installed.Method && s.Seen.Irreversible {
			return KeptIrreversible
		}
		return KeptAnotherWay
	case NotManaged:
		return KeptAnotherWay
	case Absent, Changed, Extra, Unknown:
	}
	return KeptUnread
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
	// value: the needs a run off a terminal cannot answer, each that asks
	// the owner something here (a password, a license, a sign-in, a
	// question): those of each install and setting set, and with --remove
	// those of each extra it uninstalls.
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
	return p.asked(lines)
}

func (p Plan) Asks(n catalog.Need) bool {
	// value: whether a need asks the owner something here: every need
	// does, but sudo where the machine answered that it asks no password.
	return n != catalog.NeedSudo || !p.Passwordless
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
