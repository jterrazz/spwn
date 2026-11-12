package calls

import (
	"context"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	"github.com/jterrazz/studio-station/src/core/fail"
	"github.com/jterrazz/studio-station/src/core/section"
)

type UpdateIn struct {
	// value: no name and no method updates every Update available row.
	Names []string
	// value: manager keys, as `--manager` takes them; the field keeps its
	// v3 name so an older station reads it.
	Method []string
	// value: the range a Machinefile lets the one app named move within, as
	// apps.UpgradeRequest's Within; an older station drops it and updates
	// as its manager does.
	Within string
}

func keys(words []string) []catalog.ManagerKey {
	// outside: a v3.0 station asks a manager in its own word (`skills`),
	// read as today's key.
	out := make([]catalog.ManagerKey, 0, len(words))
	for _, w := range words {
		out = append(out, catalog.ManagerKey(catalog.ManagerOfOlder(w)))
	}
	return out
}

var Update = call.Call[Deps, UpdateIn, []apps.Outcome]{
	Name: "apps.update", Section: section.Apps, Where: call.AtTarget, Applies: apps.Applies,
	Errors: []fail.Code{Unknown, Elsewhere, NotInstalled},
	Run: func(ctx context.Context, d Deps, in UpdateIn, p call.Progress) ([]apps.Outcome, error) {
		if len(in.Names) > 0 {
			if err := d.Apps.CheckUpgrade(in.Names); err != nil {
				return nil, coded(err)
			}
		}
		step(p, "Updating "+updating(in))
		out, err := d.Apps.Upgrade(ctx, apps.UpgradeRequest{Names: in.Names, All: len(in.Names) == 0 && len(in.Method) == 0, Via: keys(in.Method), Within: in.Within})
		return interrupted(ctx, out, err)
	},
}
