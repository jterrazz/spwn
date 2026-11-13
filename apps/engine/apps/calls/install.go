package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/apps"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type InstallIn struct {
	Names []string
	// value: whether a setting the app brings may ask its values.
	Interactive bool
	// value: the method the apps are installed by and the version asked,
	// as a Machinefile's plan chose them; empty, this machine's own.
	Method  string
	Version string
}

var Install = call.Call[Deps, InstallIn, []apps.Outcome]{
	Name: "apps.install", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Errors: []fail.Code{Unknown, Elsewhere},
	Run: func(ctx context.Context, d Deps, in InstallIn, p call.Progress) ([]apps.Outcome, error) {
		r := apps.InstallRequest{Names: in.Names, Interactive: in.Interactive, Method: catalog.Method(in.Method), Version: in.Version}
		if err := d.Apps.CheckInstall(ctx, r); err != nil {
			return nil, coded(err)
		}
		out, err := d.Apps.Install(ctx, r, p)
		return interrupted(ctx, out, err)
	},
}
