package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/machinefile"
)

// value: how long a night that ended without a record (the Machinefile
// not found, a store that cannot be written) waits before it is tried again.
const retryNight = time.Hour

type Nightly struct {
	Clock Clock
	// value: the Machinefile this machine is bound to, and since when; false
	// when none is.
	Bound func() (name string, since time.Time, ok bool)
	// value: the last night recorded here, and whether one ran.
	Last func(ctx context.Context) (machinefile.Night, bool, error)
	// value: the unattended upgrade of the Machinefile named, which records
	// the night.
	Upgrade func(ctx context.Context, name string) error
	// value: how the upgrade runs beside the engine's step: in a goroutine
	// of its own, so a step never waits for it.
	Go func(func())

	// value: when the engine started: with no night recorded, the first
	// night is past it and past the binding, so a Machinefile bound in the
	// day is not upgraded the moment after.
	Started time.Time

	mu      sync.Mutex
	running bool
	tried   time.Time
}

func (n *Nightly) Run(ctx context.Context) {
	// invariant: one night at a time, never started again while one runs,
	// and a try that recorded nothing waits retryNight before the next.
	name, bound, ok := n.Bound()
	if !ok || !n.begin() {
		return
	}
	now := n.Clock.Now()
	last, ran, err := n.Last(ctx)
	if err != nil || !machinefile.NightDue(now, later(n.Started, bound), last, ran) || n.recently(now) {
		n.end(time.Time{})
		return
	}
	n.Go(func() {
		_ = n.Upgrade(ctx, name)
		n.end(now)
	})
}

func (n *Nightly) begin() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.running {
		return false
	}
	n.running = true
	return true
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func (n *Nightly) recently(now time.Time) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return !n.tried.IsZero() && now.Sub(n.tried) < retryNight
}

func (n *Nightly) end(tried time.Time) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.running = false
	if !tried.IsZero() {
		n.tried = tried
	}
}
