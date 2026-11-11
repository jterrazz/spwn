package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/section"
)

type InstallIn struct {
	Names []string
	// value: whether a setting the app brings may ask its values.
	Interactive bool
}

var Install = call.Call[Deps, InstallIn, []apps.Outcome]{
	Name: "apps.install", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Errors: []fail.Code{Unknown, Elsewhere},
	Run: func(ctx context.Context, d Deps, in InstallIn, p call.Progress) ([]apps.Outcome, error) {
		if err := d.Apps.CheckInstall(ctx, in.Names); err != nil {
			return nil, coded(err)
		}
		out, err := d.Apps.Install(ctx, apps.InstallRequest{Names: in.Names, Interactive: in.Interactive}, p)
		return interrupted(ctx, out, err)
	},
}
