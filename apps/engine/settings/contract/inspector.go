package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/settings"
)

type Inspector interface {
	// invariant: a fact the machine could not look at reads settings.Unread;
	// only a read the context ended first is an error.
	Read(ctx context.Context, f catalog.Fact) (settings.Fact, error)
}
