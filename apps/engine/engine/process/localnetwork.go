package process

import (
	"context"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	localNetworkBound = time.Second
	localNetworkEvery = 30 * time.Second
)

// invariant: the engine reads its Mac's Local Network approval before its
// socket opens, bounded to a second, so its first hello carries it; macOS
// asks the owner on that Mac's screen at the first read it refuses. A read
// that does not end in time reads as one that cannot tell.
func (d *Engine) readLocalNetwork(ctx context.Context) {
	if d.o.LocalNetwork == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, localNetworkBound)
	defer cancel()
	read := d.o.LocalNetwork(ctx)
	if ctx.Err() != nil {
		read = ""
	}
	d.mu.Lock()
	d.localNetwork, d.localNetworkAt = read, d.h.Now()
	d.mu.Unlock()
}

// invariant: while its Mac has not approved station, the engine reads the
// approval again every 30 s, so the owner's click is seen without a
// restart; once approved, or where it cannot tell, it reads no more.
func (d *Engine) rereadLocalNetwork(ctx context.Context) {
	d.mu.Lock()
	due := d.localNetwork == model.LocalNetworkNotApproved && d.h.Now().Sub(d.localNetworkAt) >= localNetworkEvery
	d.mu.Unlock()
	if due {
		d.readLocalNetwork(ctx)
	}
}
