package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/agents"
)

type Repo interface {
	// invariant: the change of folder's work since base, counted locally:
	// files, lines added, lines taken out; no change is the zero Change.
	Change(ctx context.Context, folder, base string) (agents.Change, error)
}
