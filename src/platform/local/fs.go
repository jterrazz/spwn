package local

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type disk struct{}

func (disk) Open(name string) (fs.File, error)            { return os.Open(name) }
func (disk) ReadFile(path string) ([]byte, error)         { return os.ReadFile(path) }
func (disk) Readlink(path string) (string, error)         { return os.Readlink(path) }
func (disk) Stat(path string) (fs.FileInfo, error)        { return os.Stat(path) }
func (disk) ReadDir(path string) ([]fs.DirEntry, error)   { return os.ReadDir(path) }
func (disk) Glob(pattern string) ([]string, error)        { return filepath.Glob(pattern) }
func (disk) Remove(path string) error                     { return os.Remove(path) }
func (disk) RemoveAll(path string) error                  { return os.RemoveAll(path) }
func (disk) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }
func (disk) Chmod(path string, perm fs.FileMode) error    { return os.Chmod(path, perm) }
func (disk) Chown(path string, uid, gid int) error        { return os.Chown(path, uid, gid) }

func (disk) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (disk) WalkDir(root string, fn fs.WalkDirFunc) error { return filepath.WalkDir(root, fn) }

// invariant: the lock is flock(2)'s on its own file, exclusive and
// advisory, held until unlock: two processes of one Mac, or two
// goroutines of one, never hold it at once, and a process that dies lets
// go of it.
func (disk) Lock(path string) (unlock func(), err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

func ExclusiveWithin(ctx context.Context, path string, within time.Duration) (release func(), held bool, err error) {
	// invariant: Exclusive's lock, its folder made when missing, tried every
	// tenth of a second until it is free; one another holder keeps past
	// within is answered held, never waited for longer, and a wait ctx
	// ends first answers ctx's error: the run was interrupted, not refused.
	wait, cancel := context.WithTimeout(ctx, within)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		release, held, err := tryExclusive(path)
		if err != nil || !held {
			return release, false, err
		}
		select {
		case <-wait.Done():
			if err := ctx.Err(); err != nil {
				return nil, false, err
			}
			return nil, true, nil
		case <-tick.C:
		}
	}
}

func tryExclusive(path string) (release func(), held bool, err error) {
	f, err := openMade(path)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, true, nil
		}
		return nil, false, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, false, nil
}

func openMade(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // a lock file of station's own state folder.
}

func (disk) Rename(from, to string) error {
	err := os.Rename(from, to)
	if errors.Is(err, syscall.EXDEV) {
		return errors.Join(ErrCrossDevice, err)
	}
	return err
}

func Move(from, to string) error {
	// invariant: one rename(2), never a copy: the destination's parent is
	// made, a destination that holds anything is refused, another volume too.
	if entries, err := os.ReadDir(to); err == nil && len(entries) > 0 {
		return &fs.PathError{Op: "move", Path: to, Err: fs.ErrExist}
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil { //nolint:gosec // outside: a folder the owner's own apps read.
		return err
	}
	_ = os.Remove(to)
	return disk{}.Rename(from, to)
}

func Erase(paths []string) error {
	// invariant: a tree its tool wrote read-only (Go's module cache) is made
	// writable and removed again; every path is tried, and the first failure is
	// the one returned.
	var failed error
	for _, p := range paths {
		if os.RemoveAll(p) == nil {
			continue
		}
		_ = filepath.WalkDir(p, func(sub string, d fs.DirEntry, err error) error {
			if err == nil && d.IsDir() {
				_ = os.Chmod(sub, 0o700) //nolint:gosec // outside: a directory needs its search bit to be emptied.
			}
			return nil
		})
		if err := os.RemoveAll(p); err != nil && failed == nil {
			failed = err
		}
	}
	return failed
}

func Exclusive(path string) (release func(), err error) {
	// invariant: an exclusive flock(2) on path, made when missing, held until
	// release: a second holder, in this process or another, waits for it,
	// and the kernel lets it go when its process ends, so a crash never
	// leaves it held.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // a lock file of station's own state folder.
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
