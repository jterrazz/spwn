package process

import (
	"context"
	"time"

	model "github.com/jterrazz/studio-station/apps/engine/machines"
)

const (
	localNetworkBound    = time.Second
	localNetworkEvery    = 30 * time.Second
	localNetworkSettling = 90 * time.Second
	localNetworkRetry    = 2 * time.Second
)

// invariant: the engine reads its Mac's Local Network approval before its
// socket opens, bounded to a second, so its first hello carries it; macOS
// asks the owner on that Mac's screen at the first read it refuses. A read
// that does not end in time reads as one that cannot tell.
//
// hazard: macOS refuses the first datagram of a bundle it has just seen
// replaced, approved or not, and lets later ones through once it has
// caught up, within about a minute. So a refusal in the engine's first 90 s
// reads checking, never not approved: only a refusal past them is the
// Mac's verdict.
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
	now := d.h.Now()
	if read == model.LocalNetworkNotApproved && now.Sub(d.started) < localNetworkSettling {
		read = model.LocalNetworkChecking
	}
	d.mu.Lock()
	d.localNetwork, d.localNetworkAt = read, now
	d.mu.Unlock()
}

// invariant: while its read checks, the engine reads the approval again
// every 2 s; while its Mac has not approved station, every 30 s, so the
// owner's click is seen without a restart; once approved, or where it
// cannot tell, it reads no more.
func (d *Engine) rereadLocalNetwork(ctx context.Context) {
	due := false
	d.locked(func() {
		since := d.h.Now().Sub(d.localNetworkAt)
		due = d.localNetwork == model.LocalNetworkChecking && since >= localNetworkRetry ||
			d.localNetwork == model.LocalNetworkNotApproved && since >= localNetworkEvery
	})
	if due {
		d.readLocalNetwork(ctx)
	}
}
