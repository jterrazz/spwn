package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: how long a night that did not complete (it failed, or it ended
// without a record) waits before it is tried again.
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
	// value: before it no night can be due, so a step reads nothing: the
	// next NightHour past a check that found none due, or the end of a
	// retry's wait.
	quiet time.Time
}

func (n *Nightly) Run(ctx context.Context) {
	// invariant: one night at a time, never started again while one runs;
	// a night that failed, or a ledger that cannot be read, is tried again
	// retryNight later, and between two NightHours with nothing due no step
	// reads the binding or the ledger.
	now := n.Clock.Now()
	if !n.begin(now) {
		return
	}
	name, bound, ok := n.Bound()
	if !ok {
		n.end(time.Time{}, machinefile.LastNight(now).AddDate(0, 0, 1))
		return
	}
	last, ran, err := n.Last(ctx)
	switch {
	case err != nil:
		n.end(time.Time{}, now.Add(retryNight))
		return
	case !machinefile.NightDue(now, later(n.Started, bound), last, ran):
		n.end(time.Time{}, machinefile.LastNight(now).AddDate(0, 0, 1))
		return
	case n.recently(now):
		n.end(time.Time{}, n.tried.Add(retryNight))
		return
	}
	n.Go(func() {
		_ = n.Upgrade(ctx, name)
		n.end(now, time.Time{})
	})
}

func (n *Nightly) begin(now time.Time) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.running || now.Before(n.quiet) {
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

func (n *Nightly) end(tried, quiet time.Time) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.running, n.quiet = false, quiet
	if !tried.IsZero() {
		n.tried = tried
	}
}
