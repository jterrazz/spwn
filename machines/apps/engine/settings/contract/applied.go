package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type AppliedStore interface {
	// invariant: a setting never applied answers store.ErrNotFound.
	Get(ctx context.Context, setting string) (settings.Applied, error)
	// invariant: applying a setting again replaces what was kept of it.
	Put(ctx context.Context, a settings.Applied) error
	// invariant: every setting applied, by name.
	List(ctx context.Context) ([]settings.Applied, error)
	// invariant: deleting a setting never applied changes nothing.
	Delete(ctx context.Context, setting string) error
}
