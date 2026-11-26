package process

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
)

// invariant: the engine reads its Mac's Local Network approval before its
// socket opens, bounded to a second, so its first hello carries it; macOS
// asks the owner on that Mac's screen at the first read it refuses. A read
// that does not end in time reads as one that cannot tell. How a refusal's
// age is judged is the approvals family's (approvals.Settled).
func (d *Engine) readLocalNetwork(ctx context.Context) {
	if d.o.LocalNetwork == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, approvals.LocalNetworkBound)
	defer cancel()
	read := d.o.LocalNetwork(ctx)
	if ctx.Err() != nil {
		read = approvals.Unknown
	}
	now := d.h.Now()
	read = approvals.Settled(read, now.Sub(d.started))
	d.mu.Lock()
	d.localNetwork, d.localNetworkAt = read.Hello(), now
	d.mu.Unlock()
}

// invariant: the engine reads the approval again when the family says it
// is due (approvals.Due).
func (d *Engine) rereadLocalNetwork(ctx context.Context) {
	d.mu.Lock()
	due := approvals.Due(approvals.StateOf(d.localNetwork), d.h.Now().Sub(d.localNetworkAt))
	d.mu.Unlock()
	if due {
		d.readLocalNetwork(ctx)
	}
}
