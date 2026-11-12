package local

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// invariant: the disk's lock is held by one taker at a time: a second
// waits until the first lets go, then takes it.
func TestTheDiskLockIsHeldByOneTakerAtATime(t *testing.T) {
	at := filepath.Join(t.TempDir(), ".records.lock")
	unlock, err := disk{}.Lock(at)
	if err != nil {
		t.Fatal(err)
	}
	taken := make(chan func(), 1)
	go func() {
		second, err := disk{}.Lock(at)
		if err != nil {
			t.Error(err)
		}
		taken <- second
	}()
	select {
	case <-taken:
		t.Fatal("a second taker took the lock while the first held it")
	case <-time.After(100 * time.Millisecond):
	}
	unlock()
	select {
	case second := <-taken:
		second()
	case <-time.After(5 * time.Second):
		t.Fatal("the second taker never took the lock the first let go of")
	}
}

// invariant: a turn another holder keeps is answered held once within
// ends, and interrupted, never held, when the run's ctx ends first.
func TestAWaitForAHeldLockEndsHeldOrInterrupted(t *testing.T) {
	at := filepath.Join(t.TempDir(), "station", "runs.lock")
	release, held, err := ExclusiveWithin(context.Background(), at, time.Second)
	if err != nil || held {
		t.Fatalf("a free lock was not taken: held %v, %v", held, err)
	}
	defer release()
	if _, held, err := ExclusiveWithin(context.Background(), at, 200*time.Millisecond); err != nil || !held {
		t.Errorf("a lock held past within answered held %v, %v", held, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(150*time.Millisecond, cancel)
	if _, held, err := ExclusiveWithin(ctx, at, 10*time.Second); held || !errors.Is(err, context.Canceled) {
		t.Errorf("a wait the run's ctx ended answered held %v, %v", held, err)
	}
}
