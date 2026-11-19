package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type LogIn struct{}

var ReadLog = call.Call[Deps, LogIn, engine.Log]{
	Name: "engine.log", Where: call.AtTarget, Applies: Applies,
	Run: func(ctx context.Context, d Deps, _ LogIn, _ call.Progress) (engine.Log, error) {
		// invariant: what station wrote on this Mac, its last lines as the
		// engine kept them, each with its secrets withheld (ADR-048).
		log, err := d.Control.Log(ctx)
		for i, line := range log.Lines {
			log.Lines[i] = secret.Kept(line)
		}
		return log, err
	},
}
