package gate

import (
	"context"
	"log"
	"time"
)

// DefaultRefreshInterval drives the internal token refresher. 15
// minutes lines up with the 5-min refresh leeway used by upstream
// elements: a 1h-TTL token (Notion) gets renewed by ~55min worst case.
const DefaultRefreshInterval = 15 * time.Minute

// Scheduler runs each element's Refresh on a periodic ticker. It's
// the gate-internal replacement for the host-side spwn auth daemon
// (kardianos service) — since the gate is itself a long-running
// container, it owns the cron.
type Scheduler struct {
	registry *Registry
	interval time.Duration
	log      *log.Logger
}

// NewScheduler returns a scheduler that ticks every interval (use 0
// for DefaultRefreshInterval).
func NewScheduler(reg *Registry, interval time.Duration, logger *log.Logger) *Scheduler {
	if interval <= 0 {
		interval = DefaultRefreshInterval
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Scheduler{registry: reg, interval: interval, log: logger}
}

// Run blocks until ctx is cancelled. Calls Refresh on every element
// once at startup, then every Interval.
func (s *Scheduler) Run(ctx context.Context) {
	s.tick(ctx)

	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	s.registry.Each(func(e Element) {
		// Per-element timeout — one slow upstream can't hold up the
		// rest of the refresh sweep.
		eCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := e.Refresh(eCtx); err != nil {
			s.log.Printf("warning: %s refresh: %v", e.Name(), err)
		}
	})
}
