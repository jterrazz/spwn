package machinefile

import (
	"slices"
	"strconv"
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
// manager is not installed, whether station cannot uninstall it here,
// and for a manager, the owner's packages it holds (what no app of the
// catalog installs) and whether their list did not answer.

type Seen struct {
	Condition    Condition
	State        catalog.State
	Method       string
	Version      string
	Detail       string
	Lacks        string
	Irreversible bool
	Holds        []string
	HoldsUnread  bool
}

func (s Seen) holding() bool { return len(s.Holds) > 0 || s.HoldsUnread }

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
	Lock         Lock
	Source       *catalog.LockSource
	Needs        []Asked
	Lines        []Line
	Extra        []ExtraLine
	Skipped      []Skip
	Restarts     []catalog.AfterApply
	Passwordless bool
}

func Judge(r Resolved, cat catalog.Catalog, seen map[string]Seen, ledger []Installed, passwordless bool) Plan {
	// invariant: an item's action follows from what it reads: absent is
	// installed or set, present by its method kept, present by another of
	// its methods or another way adopted and never reinstalled, an app that no longer answers installed again, a
	// setting changed by hand or a read that did not answer left as it is
	// (the owner's to decide); an item of another phase is left to it. An
	// item skipped on this system is still listed, so never extra.
	p := Plan{Name: r.Name, Lock: r.Lock, Source: r.Source, Skipped: r.Skipped, Passwordless: passwordless}
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
// left and never asks anything, a manager that holds packages of the
// owner's, or whose packages do not read, is left with them, and one
// whose manager is missing here is left for a later run to judge. The extras run in reverse dependency
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
			if s.Seen.Method == in.Method && !s.Seen.Irreversible && !s.Seen.holding() {
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
// station cannot uninstall here, or a manager that holds the owner's
// packages, stays, so it keeps what it needs as well. What needs an extra
// that stays there by station's method is said, those two included, so
// its record is kept.
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
		if s := &extra[i]; held[s.Step.Item] || s.Fate() == KeptIrreversible || s.Fate() == KeptHolding {
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
		if m.Method() == method && m.Matches(sys) {
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
		if s.Kind != catalog.KindSetting && seen.Method != "" && seen.Method != s.Method {
			return Adopt
		}
		return Keep
	case Extra, Unknown:
	}
	return Leave
}

func (l Line) Method() string {
	// value: the method the line is carried by, as the plan and the run
	// show it: the one its step chose, or, for an app adopted from another
	// of its methods, the one that holds it.
	if l.Action == Adopt && (l.Seen.Condition == Present || l.Seen.Condition == Outdated) {
		return l.Seen.Method
	}
	return l.Step.Method
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
// there still needs it, station cannot uninstall it here, it is a manager
// that holds packages of the owner's (or whose packages do not read), its
// manager is missing here, it is there by another way than its method, or
// its read did not answer.

type Fate string

const (
	Taken              Fate = "taken"
	Gone               Fate = "gone"
	KeptRequired       Fate = "kept-required"
	KeptIrreversible   Fate = "kept-irreversible"
	KeptHolding        Fate = "kept-holding"
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
		switch {
		case s.Seen.Method != s.Installed.Method:
		case s.Seen.Irreversible:
			return KeptIrreversible
		case s.Seen.holding():
			return KeptHolding
		}
		return KeptAnotherWay
	case NotManaged:
		return KeptAnotherWay
	case Absent, Changed, Extra, Unknown:
	}
	return KeptUnread
}

// value: the words of an extra kept and let go of, the same in the plan
// and in the run: station cannot uninstall it here, or it is there by
// another way since; it is the owner's from then on.
const (
	CannotUninstall = "station cannot uninstall it: kept, and yours: remove it by hand if you want"
	AnotherWay      = "present another way since station installed it: kept, and yours"
)

func (s ExtraLine) Holding() string {
	// value: the words of a manager kept for what it holds, the same in the
	// plan and in the run.
	switch n := len(s.Seen.Holds); n {
	case 0:
		return "what it holds does not read"
	case 1:
		return "holds 1 package of yours"
	default:
		return "holds " + strconv.Itoa(n) + " packages of yours"
	}
}

func (s ExtraLine) Forgets() bool {
	// value: whether `apply --remove` changes anything for the extra: it
	// takes it back, or lets go of its record (already gone, or kept as the
	// owner's).
	switch s.Fate() {
	case Taken, Gone, KeptIrreversible, KeptAnotherWay:
		return true
	case KeptRequired, KeptHolding, KeptManagerMissing, KeptUnread:
	}
	return false
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
	// set, no extra `apply --remove` would take back or let go of.
	return !slices.ContainsFunc(p.Lines, Line.Acts) && !slices.ContainsFunc(p.Extra, ExtraLine.Forgets)
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
	// value: an upgrade keeps an app at the one version its Machinefile
	// asks, by a method that holds no range around it.
	Pinned Reason = "pinned"
	// value: an upgrade leaves an app that updates itself to its updater.
	SelfUpdating Reason = "updates-itself"
	// value: an unattended upgrade (the engine's, at night) leaves an app
	// whose update asks the owner something for the owner's own run.
	WaitsForYou Reason = "waits-for-you"
)

// value: one line of a run: what it was, how it ended, the version and
// the method that hold it now, its words (a failure's, the log that keeps
// its program's own), and whether this run installed or updated it.

type Done struct {
	Line      Line
	State     catalog.State
	Method    string
	Version   string
	Detail    string
	Log       string
	Reason    Reason
	Installed bool
	Updated   bool
	// value: the newest version the app's manager offers, read by an
	// upgrade within its range ("" where unread).
	Newest string
}

func (s Step) Kept() bool {
	// value: whether an upgrade leaves the app where it is: one that updates
	// itself, or one version asked of a method that holds no range around
	// it (apt, a release).
	return s.UpdatesSelf || (s.Version.Exact() && s.Spec.Holds() < catalog.HoldAny)
}

func (s Step) Within() string {
	// value: the range an upgrade hands the update, for a method that holds
	// any range; empty for one that updates as its manager does.
	if s.Spec.Holds() != catalog.HoldAny {
		return ""
	}
	if s.Version.Any() {
		return "latest"
	}
	return s.Version.Spec()
}

// value: a newer version past the range a Machinefile asks, which waits
// for the owner's gesture: the app, the version offered, the range as the
// file writes it, the same range moved to hold it, and the file.

type Waiting struct {
	App    string `json:"app"`
	Newest string `json:"newest"`
	Range  string `json:"range"`
	Moved  string `json:"moved"`
	File   string `json:"file,omitempty"`
}

func (v Waiting) Words() string {
	// value: a version waiting, in the words the owner acts on: `25.1.0
	// available: change '24' to '25' in <file>`.
	words := v.Newest + " available: change '" + v.Range + "' to '" + v.Moved + "'"
	if v.File != "" {
		words += " in " + v.File
	}
	return words
}

func WaitingOf(done []Done) []Waiting {
	// invariant: a version waits when the manager offers one past the
	// range asked: a major past `24`, a minor past `24.11`; latest has none.
	var out []Waiting
	for _, d := range done {
		r := d.Line.Step.Version
		if d.Newest == "" || !r.Past(d.Newest) {
			continue
		}
		out = append(out, Waiting{App: d.Line.Step.Item, Newest: d.Newest, Range: r.String(), Moved: r.Moved(d.Newest), File: d.Line.Step.Origin.File})
	}
	return out
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

func (p Plan) Upgrades() []Line {
	// invariant: an upgrade reaches every app the Machinefile lists that
	// reads there by one of its methods, the managers first and then the
	// rest, each in the plan's order, so a manager is fresh before what it
	// installs is updated; an app of another phase is its phase's, and a
	// copy no method holds is the owner's. It reaches too an app the lock
	// holds that is not there, whose method is there to install it: the
	// way out of a locked version that can no longer install.
	var managers, rest []Line
	for _, l := range p.Lines {
		if l.Step.Kind == catalog.KindSetting || l.Step.Phase != catalog.PhaseApply {
			continue
		}
		if !p.reaches(l) {
			continue
		}
		if l.Step.Kind == catalog.KindManager {
			managers = append(managers, l)
		} else {
			rest = append(rest, l)
		}
	}
	return append(managers, rest...)
}

func (p Plan) reaches(l Line) bool {
	if l.Seen.Method != "" && (l.Seen.Condition == Present || l.Seen.Condition == Outdated) {
		return true
	}
	if l.Seen.Condition != Absent || l.Action != Install || l.Step.Lock == "" {
		return false
	}
	return !slices.ContainsFunc(l.Step.Waits, func(w string) bool {
		return slices.ContainsFunc(p.Lines, func(o Line) bool { return o.Step.Item == w && o.Acts() })
	})
}

func (p Plan) UpgradeNeeds() []Asked {
	// value: what an upgrade asks of the owner: the needs of each app it
	// updates, a version the Machinefile pins being kept as it is. An app
	// updated asks only what its method asks; the item's needs are its
	// install's, asked when the upgrade installs an app that is not there.
	var lines []Line
	for _, l := range p.Upgrades() {
		if l.Step.Kept() && l.Seen.Condition != Absent {
			continue
		}
		if l.Seen.Condition != Absent {
			l.Step.Needs = l.Step.Spec.Needs
		}
		lines = append(lines, l)
	}
	return p.asked(lines)
}

// value: an upgrade run: its plan, each app as it ended in the order it
// ran, the lock before and after, whether it was interrupted, and why the
// shared source was not pulled, when it was not (the run went on from
// its clone as it stood).

type Upgraded struct {
	Plan        Plan
	Done        []Done
	Lock        LockDiff
	Waiting     []Waiting
	Interrupted bool
	Unpulled    string
}
