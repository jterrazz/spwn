package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type MoveIn struct {
	Name string
	// value: the manager key it moves to, a Machinefile's `manager:` word
	// (`brew`, `mise`...).
	To string
}

var Move = call.Call[Deps, MoveIn, []apps.Outcome]{
	// value: an app installed here switched to another of its methods: it
	// is installed by that one first, then the copy the old one holds is
	// uninstalled, so the command never goes missing in between.
	Name: "apps.move", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Errors: []fail.Code{Unknown, Elsewhere, NotInstalled, NoMethod},
	Run: func(ctx context.Context, d Deps, in MoveIn, p call.Progress) ([]apps.Outcome, error) {
		out, err := d.Apps.Move(ctx, in.Name, catalog.Method(in.To), p)
		return interrupted(ctx, out, err)
	},
}
