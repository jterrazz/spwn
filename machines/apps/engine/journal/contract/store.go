package contract

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
)

type Journal interface {
	// invariant: the lines kept at or after since and before before (a zero
	// bound is none), newest first; a store never written has none, and no
	// error.
	Lines(ctx context.Context, since, before time.Time) ([]journal.Line, error)
	// invariant: the lines are kept beside the others, and every line
	// older than journal.Horizon before now is dropped in the same write.
	Write(ctx context.Context, now time.Time, lines ...journal.Line) error
}
