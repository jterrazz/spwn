package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/awake"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
)

var Keep = call.Call[Deps, awake.Request, awake.Awoken]{
	Name: "awake.keep", Where: call.InEngine, Applies: awake.Applies,
	Run: func(ctx context.Context, d Deps, in awake.Request, _ call.Progress) (awake.Awoken, error) {
		// invariant: a keep-awake adds nothing to a Mac that never sleeps;
		// its options are the ones the Mac reports it can take.
		if in.For == 0 && in.Run == "" {
			in.For = awake.DefaultFor
		}
		if _, err := awake.Bounded(in.For); err != nil && in.For != 0 {
			return awake.Awoken{}, d.refused(err, in)
		}
		if err := awake.Options(d.Caps, in.Lid, in.Battery); err != nil {
			return awake.Awoken{}, d.refused(err, in)
		}
		if err := d.Holder.Check(); err != nil {
			return awake.Awoken{}, err
		}
		if d.never() {
			return awake.Awoken{Mac: d.Mac, NeverSleeps: true}, nil
		}
		a, err := d.Holder.Awake(ctx, in)
		a.Mac = d.Mac
		return a, err
	},
}
