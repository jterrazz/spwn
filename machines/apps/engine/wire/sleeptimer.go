package wire

import (
	"context"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

// value: how old the engine's own reading of the Mac's sleep timer may be
// before its slices read it again: This Mac's shortest cadence, attached
// (ADR-070), since the timer changes only when the owner sets it in System
// Settings.
const sleepTimerKeep = 30 * time.Second

// invariant: the sleep timer's reading is one `pmset -g` kept for
// sleepTimerKeep; a call that reads it for the owner (awake.read) asks
// Fresh, which also renews the kept reading, so the engine's slices never
// hold an older one than the owner was shown.
type sleepTimer struct {
	mu    sync.Mutex
	clock local.Clock
	read  func(context.Context) bool
	at    time.Time
	never bool
	held  bool
}

func newSleepTimer(clock local.Clock, read func(context.Context) bool) *sleepTimer {
	return &sleepTimer{clock: clock, read: read}
}

func (s *sleepTimer) Kept(ctx context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held && s.clock.Now().Sub(s.at) < sleepTimerKeep {
		return s.never
	}
	return s.readLocked(ctx)
}

func (s *sleepTimer) Fresh(ctx context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readLocked(ctx)
}

func (s *sleepTimer) readLocked(ctx context.Context) bool {
	s.never, s.at, s.held = s.read(ctx), s.clock.Now(), true
	return s.never
}
