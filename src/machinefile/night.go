package machinefile

import (
	"slices"
	"time"
)

// value: the hour of the night the engine upgrades a bound Machinefile at,
// on the machine's own clock.

const NightHour = 3

// value: what the night's upgrade did: when it ended, the Machinefile it
// ran, each app it moved, each version waiting past a range, the apps it
// left for the owner's run (a step that asks a password), and whether it
// was interrupted.

type Night struct {
	At          time.Time `json:"at"`
	Machinefile string    `json:"machinefile"`
	Moved       []Moved   `json:"moved,omitempty"`
	Waiting     []Waiting `json:"waiting,omitempty"`
	Left        []string  `json:"left,omitempty"`
	Interrupted bool      `json:"interrupted,omitempty"`
}

// value: one app a night moved: from the version it read to the one it
// holds now, within the range its Machinefile asks.

type Moved struct {
	App   string `json:"app"`
	Was   string `json:"was"`
	Now   string `json:"now"`
	Range string `json:"range"`
}

func NightOf(u Upgraded, at time.Time) Night {
	// invariant: an app moved is one the run updated to another version
	// than it read; an app left is one whose line waits for the owner's run.
	n := Night{At: at.UTC(), Machinefile: u.Plan.Name, Waiting: u.Waiting, Interrupted: u.Interrupted}
	for _, d := range u.Done {
		switch {
		case d.Reason == WaitsForYou:
			n.Left = append(n.Left, d.Line.Step.Item)
		case d.Updated && d.Version != "" && d.Version != d.Line.Seen.Version:
			n.Moved = append(n.Moved, Moved{App: d.Line.Step.Item, Was: d.Line.Seen.Version, Now: d.Version, Range: d.Line.Step.Version.String()})
		}
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
	// when it is back. A night interrupted did not complete, so the next run
	// takes it again.
	if ran && last.Interrupted {
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
