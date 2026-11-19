package fake

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Listeners struct {
	// invariant: what listens on one machine of the world, a Mac or a VM, so
	// a listen event of the world is read at the next Read.
	w   *world.World
	mac string
}

func ListenersOf(w *world.World, mac string) *Listeners { return &Listeners{w: w, mac: mac} }

func NewListeners(listening ...ports.Listener) *Listeners {
	w := alone()
	w.Mac(thisMac).Listening = listening
	return ListenersOf(w, thisMac)
}

// value: what a scripted fault of Listeners.Read answers, as a listeners
// read that fails would.
var errListeners = errors.New("what listens could not be read")

func (l *Listeners) Read(context.Context) ([]ports.Listener, error) {
	l.w.Lock()
	defer l.w.Unlock()
	if l.w.Fails("Listeners.Read") {
		return nil, errListeners
	}
	return slices.Clone(l.w.Listening(l.mac)), nil
}
