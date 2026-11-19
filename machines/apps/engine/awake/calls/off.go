package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

type OffIn struct{}

var Off = call.Call[Deps, OffIn, awake.Slept]{
	Name: "awake.off", Where: call.InEngine, Applies: awake.Applies,
	Run: func(ctx context.Context, d Deps, _ OffIn, _ call.Progress) (awake.Slept, error) {
		// invariant: off ends the keep-awake only: Never sleeps is a
		// setting, turned off as one.
		if err := d.Holder.Check(); err != nil {
			return awake.Slept{}, err
		}
		s, err := d.Holder.Sleep(ctx)
		s.Mac, s.NeverSleeps = d.Mac, d.never()
		return s, err
	},
}
