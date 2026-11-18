package local

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
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

func TestAWaitForAHeldLockEndsWithItsContext(t *testing.T) {
	at := filepath.Join(t.TempDir(), "zshrc.lock")
	release, err := Exclusive(t.Context(), at)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	ctx, cancel := context.WithCancel(t.Context())
	ended := make(chan error, 1)
	go func() {
		second, err := Exclusive(ctx, at)
		if err == nil {
			second()
		}
		ended <- err
	}()
	time.Sleep(30 * time.Millisecond)
	select {
	case err := <-ended:
		t.Fatalf("took a held lock: %v", err)
	default:
	}
	cancel()
	select {
	case err := <-ended:
		if err == nil || err.Error() != "interrupted" || !errors.Is(err, context.Canceled) {
			t.Errorf("Exclusive = %v, want interrupted", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the wait outlived its context")
	}
}

// invariant: Append adds to the end of a file another handle has open for
// appending, and loses neither side's lines.
func TestAppendKeepsWhatAnotherAppenderWrote(t *testing.T) {
	at := filepath.Join(t.TempDir(), "engine.log")
	other, err := os.OpenFile(at, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Close() })
	for _, write := range []func() error{
		func() error { _, werr := other.WriteString("other 1\n"); return werr },
		func() error { return disk{}.Append(at, []byte("engine 1\n"), 0o600) },
		func() error { _, werr := other.WriteString("other 2\n"); return werr },
	} {
		if werr := write(); werr != nil {
			t.Fatal(werr)
		}
	}
	got, err := os.ReadFile(at)
	if err != nil {
		t.Fatal(err)
	}
	if want := "other 1\nengine 1\nother 2\n"; string(got) != want {
		t.Fatalf("the log holds %q; want %q", got, want)
	}
}

func TestATransientLockIsDeletedWhenReleasedAndNeverHeldTwice(t *testing.T) {
	at := filepath.Join(t.TempDir(), "x.lock")
	const holders = 16
	var inside, overlap atomic.Int32
	var all sync.WaitGroup
	for range holders {
		all.Go(func() {
			release, err := ExclusiveTransient(t.Context(), at)
			if err != nil {
				t.Error(err)
				return
			}
			if inside.Add(1) > 1 {
				overlap.Add(1)
			}
			time.Sleep(2 * time.Millisecond)
			inside.Add(-1)
			release()
		})
	}
	all.Wait()
	if overlap.Load() != 0 {
		t.Errorf("%d holders overlapped", overlap.Load())
	}
	if _, err := os.Stat(at); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the lock file outlived its last release: %v", err)
	}
}

func TestAReleasedTransientLockReleasedAgainLeavesTheNextHoldersFile(t *testing.T) {
	at := filepath.Join(t.TempDir(), "x.lock")
	first, err := ExclusiveTransient(t.Context(), at)
	if err != nil {
		t.Fatal(err)
	}
	first()
	second, err := ExclusiveTransient(t.Context(), at)
	if err != nil {
		t.Fatal(err)
	}
	defer second()
	first()
	if _, err := os.Stat(at); err != nil {
		t.Errorf("a second release removed the lock file another holder owns: %v", err)
	}
}
