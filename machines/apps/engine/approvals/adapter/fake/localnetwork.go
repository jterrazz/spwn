package fake

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type LocalNetwork struct {
	// invariant: the Local Network approval of one Mac of the world: its
	// read is the world's, so the event local-network moves it.
	w   *world.World
	mac string
}

func LocalNetworkOf(w *world.World, mac string) *LocalNetwork {
	return &LocalNetwork{w: w, mac: mac}
}

func (l *LocalNetwork) Read(context.Context) approvals.State {
	l.w.Lock()
	defer l.w.Unlock()
	return approvals.StateOf(l.w.Mac(l.mac).LocalNetwork)
}
