package machinefile

import (
	"cmp"
	"slices"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: the hour of the night the engine upgrades a bound Machinefile at,
// on the machine's own clock.

const NightHour = 3

// value: what the night's upgrade did: when it ended, the Machinefile it
// ran, each app it moved, each version waiting past a range, the apps it
// left for the owner's run (a step that asks a password), the apps whose
// update failed, whether it was interrupted, why the shared source was
// not pulled, why the night did not run to its end (the Machinefile
// does not resolve, another run held the machine, the lock not written),
// and, beside a night that did not, the last one of its Machinefile that
// did: what that one moved and left waiting still holds.

type Night struct {
	At          time.Time `json:"at"`
	Machinefile string    `json:"machinefile"`
	Moved       []Moved   `json:"moved,omitempty"`
	Waiting     []Waiting `json:"waiting,omitempty"`
	Left        []string  `json:"left,omitempty"`
	Failed      []string  `json:"failed,omitempty"`
	Interrupted bool      `json:"interrupted,omitempty"`
	Unpulled    string    `json:"unpulled,omitempty"`
	Error       string    `json:"error,omitempty"`
	Last        *Night    `json:"last,omitempty"`
}

// value: one app a night moved: from the version it read to the one it
// holds now, within the range its Machinefile asks.

type Moved struct {
	App   string `json:"app"`
	Was   string `json:"was"`
	Now   string `json:"now"`
	Range string `json:"range"`
}

func NightOf(u Upgraded, failed error, name string, at time.Time) Night {
	// invariant: an app moved is one the run updated to another version
	// than it read; an app left is one whose line waits for the owner's run;
	// an app failed is one whose update ended broken. A night that failed
	// before its end keeps its error, so the owner reads it.
	n := Night{At: at.UTC(), Machinefile: cmp.Or(u.Plan.Name, name), Waiting: u.Waiting, Interrupted: u.Interrupted, Unpulled: u.Unpulled}
	if failed != nil && !u.Interrupted {
		n.Error = failed.Error()
	}
	n.seen(u)
	return n
}

func (n Night) Complete() bool { return !n.Interrupted && n.Error == "" }

func (n Night) After(last Night, ran bool) Night {
	// invariant: a night that did not run to its end keeps beside it the
	// last night of its Machinefile that did, so a failed, busy or
	// interrupted night never erases what that one moved and left waiting;
	// a night that ran to its end keeps none.
	n.Last = nil
	switch {
	case n.Complete() || !ran || last.Machinefile != n.Machinefile:
	case last.Complete():
		last.Last = nil
		n.Last = &last
	default:
		n.Last = last.Last
	}
	return n
}

func (n *Night) seen(u Upgraded) {
	n.Left, n.Failed = nil, nil
	for _, d := range u.Done {
		switch {
		case d.Reason == WaitsForYou:
			n.Left = append(n.Left, d.Line.Step.Item)
		case d.Reason == Interrupted || d.Reason == NotRun:
		case d.State == catalog.StateBroken:
			n.Failed = append(n.Failed, d.Line.Step.Item)
		case d.Updated && d.Version != "" && d.Version != d.Line.Seen.Version:
			n.Moved = append(n.Moved, Moved{App: d.Line.Step.Item, Was: d.Line.Seen.Version, Now: d.Version, Range: d.Line.Step.Version.String()})
		}
	}
}

func (n Night) Refreshed(u Upgraded) Night {
	// invariant: the owner's own upgrade of the night's Machinefile, run to
	// its end, is what the machine now reads: what waits, what is left,
	// what failed and whether the source was pulled are its own, and the
	// night's moves stay the night's; it ran to its end, so it keeps no
	// last night beside it.
	moved := n.Moved
	n.Waiting, n.Unpulled, n.Error, n.Interrupted, n.Last = u.Waiting, u.Unpulled, "", false, nil
	n.seen(u)
	n.Moved = moved
	return n
}

func (n Night) Reread(p Plan) Night {
	// invariant: a version waits while its app's line, as the plan reads
	// it now, holds a range it is past; one whose range was moved to hold
	// it, or whose app is listed no longer, waits no more, and the one
	// kept beside a night that did not end is read the same way.
	ranges := map[string]Line{}
	for _, l := range p.Lines {
		ranges[l.Step.Item] = l
	}
	var waiting []Waiting
	for _, w := range n.Waiting {
		l, ok := ranges[w.App]
		if r := l.Step.Version; ok && r.Past(w.Newest) {
			waiting = append(waiting, Waiting{App: w.App, Newest: w.Newest, Range: r.String(), Moved: r.Moved(w.Newest), File: l.Step.Origin.File})
		}
	}
	n.Waiting = waiting
	if n.Last != nil {
		last := n.Last.Reread(p)
		n.Last = &last
	}
	return n
}

func LastNight(now time.Time) time.Time {
	// value: the most recent NightHour at or before now, on now's clock.
	y, m, d := now.Date()
	at := time.Date(y, m, d, NightHour, 0, 0, 0, now.Location())
	if at.After(now) {
		at = at.AddDate(0, 0, -1)
	}
	return at
}

func NightDue(now, since time.Time, last Night, ran bool) bool {
	// invariant: a night is due once a NightHour has passed since the last
	// night that completed, or, with none, since since (the engine's start
	// or the binding, the later): a Mac asleep or off at that hour upgrades
	// when it is back. A night interrupted or failed did not complete, so
	// the next run takes it again.
	if ran && (last.Interrupted || last.Error != "") {
		return true
	}
	from := since
	if ran {
		from = last.At
	}
	return LastNight(now).After(from)
}

func (p Plan) Waits() []string {
	// value: the apps an unattended upgrade leaves for the owner's run: those
	// whose update asks the owner something here (a password).
	var out []string
	for _, a := range p.UpgradeNeeds() {
		for _, it := range a.Items {
			if !slices.Contains(out, it) {
				out = append(out, it)
			}
		}
	}
	return out
}
