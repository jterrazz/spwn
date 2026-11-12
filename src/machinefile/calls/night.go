package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/machinefile"
)

type NightlyIn struct{}

// value: the last night the engine's upgrade recorded here, and whether
// one ran at all.

type Nightly struct {
	Night machinefile.Night `json:"night"`
	Ran   bool              `json:"ran"`
}

var LastNight = call.Call[Deps, NightlyIn, Nightly]{
	// value: what the engine's last night did here: the apps it moved
	// within their range, the versions waiting past one, what it left for
	// the owner's run. It never acts.
	Name: "machinefile.nightly", Where: call.AtTarget, Applies: machine.Anywhere,
	Run: func(ctx context.Context, d Deps, _ NightlyIn, _ call.Progress) (Nightly, error) {
		n, ran, err := d.Ledger.Night(ctx)
		return Nightly{Night: n, Ran: ran}, err
	},
}
