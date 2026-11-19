package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

type ReleaseIn struct{}

var ReadRelease = call.Call[Deps, ReleaseIn, model.Release]{
	Name: "engine.read-release", Where: call.InEngine, Applies: Applies,
	Run: func(ctx context.Context, d Deps, _ ReleaseIn, _ call.Progress) (model.Release, error) {
		// invariant: a read asks nothing of GitHub: it answers what the
		// engine kept of its last asks, a refused one beside the last answer.
		kept, ok, err := d.Cache.Read(ctx)
		if err != nil {
			return model.Release{}, err
		}
		return engine.ReleaseOf(d.Version, kept, ok), nil
	},
}

var RefreshRelease = call.Call[Deps, ReleaseIn, model.Release]{
	Name: "engine.refresh-release", Where: call.InEngine, Applies: Applies,
	Errors: []fail.Code{ReleaseUnreachable},
	Run: func(ctx context.Context, d Deps, _ ReleaseIn, _ call.Progress) (model.Release, error) {
		// invariant: GitHub is asked now and what it answered is kept; a
		// GitHub that does not answer is kept as the refusal, beside the
		// last answer, and refused.
		kept, _, err := d.Cache.Read(ctx)
		if err != nil {
			kept = engine.ReleaseKept{}
		}
		now := d.Now()
		latest, asked := d.Releases.Latest(ctx)
		if asked != nil {
			if err := d.Cache.Write(ctx, kept.Failed(now, asked.Error())); err != nil {
				return model.Release{}, err
			}
			return model.Release{}, &fail.Error{Type: fail.Failed, Code: ReleaseUnreachable, Subject: "release", Detail: map[string]string{"cause": asked.Error()}, Cause: asked}
		}
		kept = kept.Answered(now, latest, d.Arch)
		if err := d.Cache.Write(ctx, kept); err != nil {
			return model.Release{}, err
		}
		return engine.ReleaseOf(d.Version, kept, true), nil
	},
}
