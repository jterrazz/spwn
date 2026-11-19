package process

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type Notifier interface {
	Notify(ctx context.Context, n engine.Notification)
}

func (d *Engine) notify(ctx context.Context, n engine.Notification) {
	if d.notifier != nil {
		d.notifier.Notify(ctx, n)
	}
}
