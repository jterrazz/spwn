package wire

import (
	"context"
	"testing"
	"time"
)

func TestTheSleepTimerIsHeldForItsLifeAndAFreshReadRenewsIt(t *testing.T) {
	clock := &idleClock{at: time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)}
	reads, never := 0, false
	timer := newSleepTimer(clock, func(context.Context) bool { reads++; return never })
	ctx := context.Background()

	timer.Kept(ctx)
	clock.advance(sleepTimerKeep - time.Second)
	never = true
	if timer.Kept(ctx) || reads != 1 {
		t.Fatalf("within its life Kept read again (reads %d): want the held reading", reads)
	}
	if !timer.Fresh(ctx) || reads != 2 {
		t.Fatalf("Fresh did not read the timer now (reads %d)", reads)
	}
	clock.advance(sleepTimerKeep - time.Second)
	never = false
	if !timer.Kept(ctx) || reads != 2 {
		t.Fatalf("Kept after a Fresh read read again (reads %d): the Fresh reading should be held", reads)
	}
	clock.advance(time.Second)
	if timer.Kept(ctx) || reads != 3 {
		t.Fatalf("past its life Kept did not read again (reads %d)", reads)
	}
}
