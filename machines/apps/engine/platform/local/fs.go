package local

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

type disk struct{ owner *invoker }

func (disk) Open(name string) (fs.File, error)          { return os.Open(name) }
func (disk) ReadFile(path string) ([]byte, error)       { return os.ReadFile(path) }
func (disk) Readlink(path string) (string, error)       { return os.Readlink(path) }
func (disk) Symlink(target, path string) error          { return os.Symlink(target, path) }
func (disk) Stat(path string) (fs.FileInfo, error)      { return os.Stat(path) }
func (disk) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (disk) Glob(pattern string) ([]string, error)      { return filepath.Glob(pattern) }
func (disk) Remove(path string) error                   { return os.Remove(path) }
func (disk) RemoveAll(path string) error                { return os.RemoveAll(path) }
func (disk) Chmod(path string, perm fs.FileMode) error  { return os.Chmod(path, perm) }
func (disk) Chown(path string, uid, gid int) error      { return os.Chown(path, uid, gid) }

func (d disk) MkdirAll(path string, perm fs.FileMode) error {
	made := missing(path)
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	return d.owner.give(made...)
}

func (d disk) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return d.owner.give(path)
}

func (d disk) Append(path string, data []byte, perm fs.FileMode) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return d.owner.give(path)
}

func (d disk) Replace(path string, data []byte, perm fs.FileMode) error {
	if err := (durable{sync: (*os.File).Sync, link: os.Link}).replace(path, data, perm); err != nil {
		return err
	}
	return d.owner.give(path)
}

func (d disk) Swap(path string, seen Seen, data []byte, perm fs.FileMode) error {
	if err := (durable{sync: (*os.File).Sync, link: os.Link}).swap(path, seen, data, perm); err != nil {
		return err
	}
	return d.owner.give(path)
}

func (disk) WalkDir(root string, fn fs.WalkDirFunc) error { return filepath.WalkDir(root, fn) }

// invariant: the lock is flock(2)'s on its own file, exclusive and
// advisory, held until unlock: two processes of one Mac, or two
// goroutines of one, never hold it at once, and a process that dies lets
// go of it.
func (d disk) Lock(path string) (unlock func(), err error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := d.owner.give(path); err != nil {
		_ = f.Close()
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

func (s System) ExclusiveWithin(ctx context.Context, path string, within time.Duration) (release func(), held bool, err error) {
	// invariant: Exclusive's lock, its folder made when missing, tried until
	// it is free or within passes (tried).
	held, err = tried(ctx, within, func() (bool, error) {
		var busy bool
		var failed error
		release, busy, failed = tryExclusive(path, s.owner)
		return busy, failed
	})
	return release, held, err
}

// invariant: try is called every tenth of a second while it answers held;
// one held past within is answered held, never waited for longer, and a
// wait ctx ends first answers ctx's error: the run was interrupted, not
// refused.
func tried(ctx context.Context, within time.Duration, try func() (held bool, err error)) (bool, error) {
	deadline, cancel := context.WithTimeout(ctx, within)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		held, err := try()
		if err != nil || !held {
			return false, err
		}
		select {
		case <-deadline.Done():
			if err := ctx.Err(); err != nil {
				return false, err
			}
			return true, nil
		case <-tick.C:
		}
	}
}

func tryExclusive(path string, owner *invoker) (release func(), held bool, err error) {
	f, err := openMade(path, owner)
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

func openMade(path string, owner *invoker) (*os.File, error) {
	made := missing(filepath.Dir(path))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // a lock file of station's own state folder.
	if err != nil {
		return nil, err
	}
	if err := owner.give(append(made, path)...); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
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

func Exclusive(ctx context.Context, path string) (release func(), err error) {
	f, err := lockOpen(ctx, path)
	if err != nil {
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}

func ExclusiveTransient(ctx context.Context, path string) (release func(), err error) {
	// invariant: Exclusive's lock, whose file release deletes while the lock is
	// still held. A holder that waited on a file deleted meanwhile has locked
	// a name nobody holds: it finds its file is no longer the one at path and
	// asks again on the new one, so two holders never overlap.
	for {
		f, err := lockOpen(ctx, path)
		if err != nil {
			return nil, err
		}
		held, err := f.Stat()
		at, atErr := os.Stat(path)
		if err == nil && atErr == nil && os.SameFile(held, at) {
			var once sync.Once
			return func() {
				once.Do(func() {
					_ = os.Remove(path)
					_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
					_ = f.Close()
				})
			}, nil
		}
		_ = f.Close()
	}
}

func lockOpen(ctx context.Context, path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // a lock file of station's own state folder.
	if err != nil {
		return nil, err
	}
	// invariant: an exclusive flock(2) on path, made when missing, held until
	// release: a second holder, in this process or another, waits for it,
	// and the kernel lets it go when its process ends, so a crash never
	// leaves it held. The wait asks the lock again and again, without
	// blocking, at a growing pause up to lockPause, and ends when ctx does
	// (it reads interrupted): a blocking flock(2) could not be cut short.
	for pause := time.Millisecond; ; pause = min(2*pause, lockPause) {
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			break
		}
		if err = wait(ctx, pause); err != nil {
			break
		}
	}
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// value: the longest pause between two asks of a lock another holds.
const lockPause = 50 * time.Millisecond

func wait(ctx context.Context, pause time.Duration) error {
	t := time.NewTimer(pause)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return lockInterrupted{cause: ctx.Err()}
	case <-t.C:
		return nil
	}
}

// value: a wait its context ended before the lock was free; it reads
// "interrupted", and is the context's own error underneath.
type lockInterrupted struct{ cause error }

func (lockInterrupted) Error() string   { return "interrupted" }
func (i lockInterrupted) Unwrap() error { return i.cause }
