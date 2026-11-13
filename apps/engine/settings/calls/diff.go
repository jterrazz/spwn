package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type DiffIn struct{ Name string }

type DiffOut struct {
	Row settings.Row
	// value: a file setting: the unified diff, station's against this
	// machine's; any other: its values before, and what station would set.
	Diff          string
	Before, After []settings.KeyValue
}

var Diff = call.Call[Deps, DiffIn, DiffOut]{
	Name: "settings.diff", Section: section.Settings, Where: call.AtTarget, Applies: settings.Applies,
	Errors: []fail.Code{Unknown, IsChecked, NotForThisMachine},
	Run: func(ctx context.Context, d Deps, in DiffIn, _ call.Progress) (DiffOut, error) {
		if c, ok := settings.CheckedNamed(in.Name); ok {
			return DiffOut{}, fail.New(fail.Refused, IsChecked, in.Name, map[string]string{"fix": c.Fix})
		}
		it, err := find(d, in.Name, true)
		if err != nil {
			return DiffOut{}, err
		}
		row, now := judge(ctx, d, it, map[string]bool{})
		return DiffOut{Row: row, Diff: now.Diff, Before: now.Before, After: now.After}, nil
	},
}
