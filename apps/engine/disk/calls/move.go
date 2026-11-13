package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
	"github.com/jterrazz/studio-station/apps/engine/disk"
)

type MoveIn struct{ Folder string }

var Move = call.Call[Deps, MoveIn, disk.Outcome]{
	Name: "disk.move", Section: section.Disk, Where: call.AtTarget, Applies: disk.Applies,
	Errors: []fail.Code{NotMisplaced, AppRunning},
	Run: func(ctx context.Context, d Deps, in MoveIn, _ call.Progress) (disk.Outcome, error) {
		out, err := d.Disk.Move(ctx, in.Folder)
		return out, coded(err)
	},
}
