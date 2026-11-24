package imagetest

import (
	"context"
	"errors"
	"testing"
	"time"
)

func noPause(int) time.Duration { return 0 }

func TestRetry_StopsAtTheFirstSuccess(t *testing.T) {
	calls := 0
	err := retry(context.Background(), 3, noPause, func() error {
		calls++
		if calls < 2 {
			return errors.New("mirror sync in progress")
		}
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatalf("want nil after 2 calls, got %v after %d", err, calls)
	}
}

func TestRetry_FailsWithTheLastErrorAfterTheBound(t *testing.T) {
	calls := 0
	err := retry(context.Background(), 3, noPause, func() error {
		calls++
		return errors.New("failure " + string(rune('0'+calls)))
	})
	if err == nil || err.Error() != "failure 3" || calls != 3 {
		t.Fatalf("want failure 3 after 3 calls, got %v after %d", err, calls)
	}
}

func TestRetry_StopsWaitingWhenTheContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	err := retry(ctx, 3, func(int) time.Duration { return time.Hour }, func() error {
		calls++
		return errors.New("failure")
	})
	if err == nil || calls != 1 {
		t.Fatalf("want the first error after 1 call, got %v after %d", err, calls)
	}
}
