package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	activitycalls "github.com/jterrazz/studio-station/apps/engine/activity/calls"
)

const defaultBudget = 2 * time.Second

type Reader[Item any] struct{ d Deps[Item] }

func New[Item any](d Deps[Item]) *Reader[Item] {
	return &Reader[Item]{d: d}
}

func (r *Reader[Item]) Sample(ctx context.Context) (activity.Reading, error) {
	return activitycalls.Measure(ctx, r.d.Meter)
}

func (r *Reader[Item]) Forget() {
	// invariant: a reload of the whole machine drops every shared reading of
	// the run (the inventory, the walk, the shared commands), not only its own.
	r.d.Pool.Forget("")
}
