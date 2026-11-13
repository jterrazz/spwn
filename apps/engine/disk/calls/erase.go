package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

type EraseIn struct {
	App     string
	Confirm string
}

var Erase = call.Call[Deps, EraseIn, disk.Outcome]{
	Name: "disk.erase", Section: section.Disk, Where: call.AtTarget, Applies: disk.Applies, Confirm: confirm.Erase,
	Errors: []fail.Code{Unknown, Elsewhere, NeverTouched, NothingToErase, AppRunning, confirm.Needed, confirm.Mismatch},
	Run: func(ctx context.Context, d Deps, in EraseIn, _ call.Progress) (disk.Outcome, error) {
		// invariant: an erase refuses what it never erases and an app that
		// runs before it asks the name typed, so nothing is typed in vain.
		if err := d.Disk.CheckRunning(ctx, in.App); err != nil {
			return disk.Outcome{}, coded(err)
		}
		if err := confirm.Require(confirm.For(confirm.Erase, d.Kind), in.App, in.Confirm); err != nil {
			return disk.Outcome{}, err
		}
		out, err := d.Disk.Erase(ctx, in.App)
		return out, coded(err)
	},
}
