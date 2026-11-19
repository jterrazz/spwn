package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type Notifier interface {
	// invariant: telling never fails the engine: a notification that cannot
	// be delivered is dropped.
	Notify(ctx context.Context, n engine.Notification)
}
