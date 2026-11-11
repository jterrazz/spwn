package local

import (
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
