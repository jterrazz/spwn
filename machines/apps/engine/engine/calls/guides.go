package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/guide"
)

var Guides = call.Call[Deps, struct{}, guide.Guides]{
	// invariant: the acts the engine guides, each as its Guide, answered
	// from the store as the view of the same name projects it: the board's
	// acts of This Mac and the copy of every other Mac's. A door that is
	// not the engine relays the engine's answer.
	Name: "engine.guides", Where: call.InEngine, Applies: Applies,
	Run: func(ctx context.Context, d Deps, _ struct{}, _ call.Progress) (guide.Guides, error) {
		src, ok := ctx.Value(stateKey{}).(State)
		if !ok || src == nil {
			if d.Guided == nil {
				return guide.Guides{}, &DownError{}
			}
			return d.Guided(ctx)
		}
		if d.Views == nil {
			return guide.Guides{}, &DownError{}
		}
		src.Ready(ctx)
		views := d.Views.of.Project(src.Store().Now(), func(name string) bool { return name == "engine.guides" })
		listed, _ := views["engine.guides"].Value.(guide.Guides)
		return listed, nil
	},
}
