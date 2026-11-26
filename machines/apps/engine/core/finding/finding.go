package finding

import (
	"cmp"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type Severity string

const (
	Broken     Severity = "broken"
	NeedsALook Severity = "needs-a-look"
)

func (Severity) Enum() []string { return []string{string(Broken), string(NeedsALook)} }

type Urgency string

const (
	// value: SPEC B5.7: what makes a finding a need of Needs you rather
	// than upkeep: red (a server down, a port exposed), what expires (a
	// lease ending). Upkeep has none.
	Red      Urgency = "red"
	Expiring Urgency = "expiring"
)

type Fix struct {
	// invariant: a fix that needs admin carries a Path or a Command, never a Call.
	Call string
	// invariant: each key is a field of Call's input, as its schema in
	// packages/api names it; a list field holds the one value given
	// (rule fix-args-are-inputs).
	Args map[string]string
	// value: the machine Call runs at when it is not the finding's own: a
	// VM's restart or delete goes to its Mac.
	On string
	// value: the name Call's question takes typed, when it asks one for
	// the machine it acts on; the printed command answers it.
	Confirm string
	Command string
	Path    string
	// value: the System Settings pane that holds the fix, as the URL that opens it; set only when the machine the fix is on is the Mac whose engine judged it.
	Pane string `json:",omitempty"`
	// invariant: Command is the owner's to run from a terminal (it asks for admin, or asks a question at a terminal), never Spwn's: a client fills it into Terminal and runs nothing until the owner presses Return.
	InTerminal bool `json:",omitempty"`
}

type Finding struct {
	Machine  machine.Ref
	Section  section.ID
	Severity Severity
	Subject  string
	Text     string
	// value: the sentence under the finding's title, empty when the model
	// tells none.
	Detail string `json:",omitempty"`
	Fix    Fix
	// value: it is of what failed to begin, a create: a fix by its start
	// runs it again, so a client words that button "Try again", not its
	// call's.
	Again   bool
	Urgency Urgency
	// value: when an expiring finding expires.
	Ends time.Time
}

func (f Finding) Target() string {
	// invariant: the machine a fix's call runs at: its own On, else the
	// machine the finding is about.
	return cmp.Or(f.Fix.On, f.Machine.Name)
}

func Rank(fs []Finding) []Finding {
	return RankBy(fs, func(f Finding) Finding { return f })
}

func RankBy[T any](xs []T, of func(T) Finding) []T {
	// invariant: SPEC §5.1.1 under B5.7. Red first, then what expires, the soonest first, then the rest; then broken
	// before needs a look; then the one section order; then the Macs in the
	// order they first appear (This Mac first, as every read lists it), each
	// Mac before its VMs. Ties keep their order.
	urgencyAt := func(u Urgency) int {
		if i := slices.Index([]Urgency{Red, Expiring}, u); i >= 0 {
			return i
		}
		return 2
	}
	return rank(xs, of, func(a, b Finding) int {
		if d := urgencyAt(a.Urgency) - urgencyAt(b.Urgency); d != 0 {
			return d
		}
		if d := a.Ends.Compare(b.Ends); a.Urgency == Expiring && d != 0 {
			return d
		}
		if d := severityAt(a.Severity) - severityAt(b.Severity); d != 0 {
			return d
		}
		return sectionAt(section.Order(), a.Section) - sectionAt(section.Order(), b.Section)
	})
}

func RankUpkeep[T any](xs []T, of func(T) Finding) []T {
	// invariant: SPEC B5.7's second list, worth doing when the owner has a
	// minute: a security setting first, then what fails, then sign-ins, then
	// updates, then the rest in the one section order; broken before needs a
	// look; then the Macs as Rank orders them.
	order := []section.ID{section.Settings, section.Background, section.Accounts, section.Apps}
	for _, id := range section.Order() {
		if !slices.Contains(order, id) {
			order = append(order, id)
		}
	}
	return rank(xs, of, func(a, b Finding) int {
		if d := sectionAt(order, a.Section) - sectionAt(order, b.Section); d != 0 {
			return d
		}
		return severityAt(a.Severity) - severityAt(b.Severity)
	})
}

func rank[T any](xs []T, of func(T) Finding, first func(a, b Finding) int) []T {
	macs := map[string]int{}
	for _, x := range xs {
		if _, seen := macs[of(x).Machine.Mac()]; !seen {
			macs[of(x).Machine.Mac()] = len(macs)
		}
	}
	vmAt := func(r machine.Ref) int {
		if r.On == "" {
			return 0
		}
		return 1
	}
	out := slices.Clone(xs)
	slices.SortStableFunc(out, func(x, y T) int {
		a, b := of(x), of(y)
		for _, d := range []int{first(a, b), macs[a.Machine.Mac()] - macs[b.Machine.Mac()], vmAt(a.Machine) - vmAt(b.Machine)} {
			if d != 0 {
				return d
			}
		}
		return 0
	})
	return out
}

func severityAt(s Severity) int {
	if s == Broken {
		return 0
	}
	return 1
}

func sectionAt(order []section.ID, id section.ID) int {
	if i := slices.Index(order, id); i >= 0 {
		return i
	}
	return len(order)
}
