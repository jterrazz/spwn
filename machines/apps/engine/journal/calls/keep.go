package calls

import (
	"cmp"
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
)

type Ran struct {
	// value: the call that ran, the machine it ran on and who it was done
	// by, empty for the owner.
	Call, Machine, By string
	// value: the failure it ended on, in the owner's words; empty when it
	// ran to its end.
	Failed string
}

func Keep(ctx context.Context, d Deps, ran Ran, lines ...journal.Line) error {
	// invariant: the lines a gesture keeps are written on this Mac's clock,
	// each done by its own author or the run's, the owner when none is
	// named, on its own machine or the one the call ran on; a line that
	// says no outcome is the run's, failed with its words when the run
	// failed; a retry runs where the call ran unless it names its machine;
	// a line of the night is its night's. Nothing to keep is no write.
	if len(lines) == 0 {
		return nil
	}
	now, by := d.Now(), cmp.Or(ran.By, journal.ByYou)
	for i := range lines {
		l := &lines[i]
		l.At, l.Call, l.By, l.Machine = now, ran.Call, cmp.Or(l.By, by), cmp.Or(l.Machine, ran.Machine)
		if by == journal.ByNight {
			l.Night = now
		}
		if l.Retry != nil {
			l.Retry.On = cmp.Or(l.Retry.On, ran.Machine)
		}
		switch {
		case l.Outcome != "":
		case ran.Failed != "":
			l.Outcome, l.Detail = journal.Failed, cmp.Or(l.Detail, ran.Failed)
		default:
			l.Outcome = journal.Done
		}
	}
	return d.Journal.Write(ctx, now, lines...)
}
