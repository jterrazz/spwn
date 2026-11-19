package files

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strconv"
	"sync/atomic"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

type renaming interface {
	writable
	Rename(from, to string) error
}

// value: the locks of the owner's Machinefiles folder, `<name>.lock`
// beside `<name>.machine.yaml`, which no source reads.

type Locks struct {
	fs        renaming
	dir, said string
}

var _ contract.Locks = Locks{}

func NewLocks(fsys renaming, dir, said string) Locks { return Locks{fs: fsys, dir: dir, said: said} }

func (l Locks) Where(name string) string { return l.said + "/" + name + catalog.LockSuffix }

func (l Locks) Read(_ context.Context, name string) (catalog.LockFile, bool, error) {
	body, err := l.fs.ReadFile(filepath.Join(l.dir, name+catalog.LockSuffix))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return catalog.LockFile{}, false, nil
	case err != nil:
		return catalog.LockFile{}, false, err
	}
	lock, err := catalog.DecodeLock(l.Where(name), body)
	if err != nil {
		return catalog.LockFile{}, false, &machinefile.RefusedFileError{Format: "machinefile", Err: err}
	}
	return lock, true, nil
}

// value: a temp name no other write of this process shares; across
// processes the file lock is what keeps one write at a time.
var written atomic.Uint64

func (l Locks) Write(ctx context.Context, was, now catalog.LockFile) error {
	if err := l.fs.MkdirAll(l.dir, 0o700); err != nil {
		return err
	}
	if locker, ok := l.fs.(interface {
		Lock(path string) (unlock func(), err error)
	}); ok {
		unlock, err := locker.Lock(filepath.Join(l.dir, ".writing"))
		if err != nil {
			return err
		}
		defer unlock()
	}
	switch current, ok, err := l.Read(ctx, now.Machinefile); {
	case err != nil:
		return err
	case ok:
		now = machinefile.Merged(current, was, now)
	}
	body, err := catalog.EncodeLock(now)
	if err != nil {
		return err
	}
	at := filepath.Join(l.dir, now.Machinefile+catalog.LockSuffix)
	temp := at + "." + strconv.FormatUint(written.Add(1), 10) + ".new"
	if err := l.fs.WriteFile(temp, body, 0o600); err != nil {
		return err
	}
	return l.fs.Rename(temp, at)
}
