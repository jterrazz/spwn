package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/journal"
)

type KeptIn struct {
	// value: the oldest instant asked, kept; zero asks from the first line.
	Since time.Time `json:",omitzero"`
	// value: the instant the lines end before; zero asks up to the last.
	Before time.Time `json:",omitzero"`
}

type KeptOut struct {
	// value: this Mac's lines, newest first.
	Lines []journal.Line
}

var Kept = call.Call[Deps, KeptIn, KeptOut]{
	Name: "journal.kept", Where: call.AtTarget, Applies: Applies,
	Errors: []fail.Code{Unreadable},
	Run: func(ctx context.Context, d Deps, in KeptIn, _ call.Progress) (KeptOut, error) {
		lines, err := d.Journal.Lines(ctx, in.Since, in.Before)
		if err != nil {
			return KeptOut{}, coded(err)
		}
		return KeptOut{Lines: journal.Kept(lines, d.Now())}, nil
	},
}
