//go:build !windows

package architect

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const imageLockPoll = 50 * time.Millisecond

// invariant: every world of a machine is built under the one tag platform.WorldImage, so two spawns of different tool sets
// retag it under each other; the lock lives in the temp folder, the one place whatever SPWN_HOME the spawns run under shares.
func lockWorldImage(ctx context.Context) (func(), error) {
	path := filepath.Join(os.TempDir(), "spwn-world-image.lock")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open image lock %s: %w", path, err)
	}
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			break
		}
		if err != syscall.EWOULDBLOCK {
			_ = f.Close()
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, ctx.Err()
		case <-time.After(imageLockPoll):
		}
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
		})
	}, nil
}
