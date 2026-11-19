package fake

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Notifier struct{ w *world.World }

var _ contract.Notifier = (*Notifier)(nil)

func NewNotifier(w *world.World) *Notifier { return &Notifier{w: w} }

func (n *Notifier) Notify(_ context.Context, told engine.Notification) {
	n.w.Told = append(n.w.Told, told)
}
