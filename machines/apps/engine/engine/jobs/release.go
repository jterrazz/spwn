package jobs

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type Release struct {
	Clock Clock
	// value: what the engine kept of its last asks, and whether it kept any.
	Kept func(ctx context.Context) (engine.ReleaseKept, bool, error)
	// value: the ask of GitHub, engine.refresh-release, which keeps its answer.
	Refresh func(ctx context.Context) error
	// value: how the ask runs beside the engine's step: in a goroutine of its
	// own, so a step never waits for GitHub.
	Go func(func())

	mu      sync.Mutex
	running bool
	// value: before it no ask is due, so a step reads nothing.
	quiet time.Time
}

func (r *Release) Run(ctx context.Context) {
	// invariant: one ask at a time; it is due at the first step when nothing
	// is kept, then engine.ReleaseEvery after the last answer and
	// engine.ReleaseRetry after a refused ask, both read from what was kept,
	// so an engine restart neither asks again early nor forgets a refusal;
	// whatever the ask did, the next one waits at least ReleaseRetry.
	now := r.Clock.Now()
	if !r.begin(now) {
		return
	}
	kept, ok, err := r.Kept(ctx)
	if err == nil && ok {
		if due := kept.Due(now); now.Before(due) {
			r.end(due)
			return
		}
	}
	r.Go(func() {
		_ = r.Refresh(ctx)
		r.end(now.Add(engine.ReleaseRetry))
	})
}

func (r *Release) begin(now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running || now.Before(r.quiet) {
		return false
	}
	r.running = true
	return true
}

func (r *Release) end(quiet time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.running, r.quiet = false, quiet
}
