//go:build !windows

package compile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const tagLockPoll = 50 * time.Millisecond

var tagLockWait = 10 * time.Minute

// invariant: two builds of one tag write the same image, and one build at a time per tag keeps a forced rebuild from
// untagging it under a sibling; the lock lives in the temp folder, the one place whatever SPWN_HOME the builds run under shares.
func lockTag(ctx context.Context, tag string) (func(), error) {
	name := "spwn-image-" + strings.NewReplacer("/", "-", ":", "-").Replace(tag) + ".lock"
	path := filepath.Join(os.TempDir(), name)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open image lock %s: %w", path, err)
	}
	waiting, cancel := context.WithTimeout(ctx, tagLockWait)
	defer cancel()
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
		case <-waiting.Done():
			_ = f.Close()
			if ctx.Err() == nil && errors.Is(waiting.Err(), context.DeadlineExceeded) {
				return nil, fmt.Errorf("another build of %s held it past %s", tag, tagLockWait)
			}
			return nil, ctx.Err()
		case <-time.After(tagLockPoll):
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
