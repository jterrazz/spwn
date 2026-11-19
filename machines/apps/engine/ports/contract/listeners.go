package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Listeners interface {
	// invariant: one listener per listening address, in the order the system
	// reports them; folding them by port is the reader's rule.
	Read(ctx context.Context) ([]ports.Listener, error)
}
