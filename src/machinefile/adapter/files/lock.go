package files

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

type renaming interface {
	writable
	Rename(from, to string) error
}

// value: the locks of the owner's Machinefiles folder, `<name>.lock`
// beside `<name>.yaml`, which no source reads.

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

func (l Locks) Write(_ context.Context, lock catalog.LockFile) error {
	body, err := catalog.EncodeLock(lock)
	if err != nil {
		return err
	}
	if err := l.fs.MkdirAll(l.dir, 0o700); err != nil {
		return err
	}
	at := filepath.Join(l.dir, lock.Machinefile+catalog.LockSuffix)
	if err := l.fs.WriteFile(at+".new", body, 0o600); err != nil {
		return err
	}
	return l.fs.Rename(at+".new", at)
}
