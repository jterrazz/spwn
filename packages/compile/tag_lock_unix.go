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

// invariant: the locks live in the temp folder, the one place whatever SPWN_HOME the builds run under shares, named
// for the user so another account's file never blocks this one.
func tagLockPath(tag, use string) string {
	name := fmt.Sprintf("spwn-%d-image-%s.%s.lock", os.Getuid(), strings.NewReplacer("/", "-", ":", "-").Replace(tag), use)
	return filepath.Join(os.TempDir(), name)
}

func lockTag(ctx context.Context, tag string) (func(), error) {
	return flockWait(ctx, tagLockPath(tag, "build"), syscall.LOCK_EX, tag)
}

func holdTag(ctx context.Context, tag string) (func(), error) {
	return flockWait(ctx, tagLockPath(tag, "use"), syscall.LOCK_SH, tag)
}

func claimTag(tag string) (func(), bool, error) {
	f, err := openLock(tagLockPath(tag, "use"))
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lock %s: %w", f.Name(), err)
	}
	return releaser(f), true, nil
}

func openLock(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open image lock %s: %w", path, err)
	}
	return f, nil
}

func flockWait(ctx context.Context, path string, how int, tag string) (func(), error) {
	f, err := openLock(path)
	if err != nil {
		return nil, err
	}
	waiting, cancel := context.WithTimeout(ctx, tagLockWait)
	defer cancel()
	for {
		err := syscall.Flock(int(f.Fd()), how|syscall.LOCK_NB)
		if err == nil {
			return releaser(f), nil
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
}

func releaser(f *os.File) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
		})
	}
}
