package calls

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type RemoveIn struct {
	Label   string
	Confirm string
}

var Remove = call.Call[Deps, RemoveIn, background.Task]{
	Name: "background.remove", Section: section.Background, Where: call.AtTarget, Applies: background.Applies, Confirm: confirm.Remove,
	Errors: []fail.Code{Unknown, IsTheEngine, RootScope, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in RemoveIn, _ call.Progress) (background.Task, error) {
		// invariant: station removes only a task of the owner's own scope, and
		// never runs sudo: a root task is refused with the command that removes
		// it by hand, and the engine's own job with how to stop the engine.
		if in.Label == catalog.EngineLabel {
			return background.Task{}, fail.New(fail.Refused, IsTheEngine, in.Label, nil)
		}
		jobs, judged, err := tasks(ctx, d)
		if err != nil {
			return background.Task{}, err
		}
		i := slices.IndexFunc(jobs, func(j background.Job) bool { return j.Label == in.Label })
		if i < 0 {
			return background.Task{}, fail.New(fail.NotFound, Unknown, in.Label, nil)
		}
		job, task := jobs[i], judged[i]
		if task.Scope == background.ScopeRoot {
			return task, fail.New(fail.Refused, RootScope, in.Label, map[string]string{"command": job.Removal})
		}
		if err := confirm.Require(confirm.For(confirm.Remove, d.Kind), in.Label, in.Confirm); err != nil {
			return task, err
		}
		return task, d.Jobs.Remove(ctx, contract.Ref{Domain: d.Domain, Label: job.Label, File: job.File})
	},
}
