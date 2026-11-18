package local

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// value: how many links a rewrite follows to the file they name before it
// gives up, as the kernel's own bound on a path does.
const linkHops = 32

func Rewrite(files FS, at string, edit func(old []byte) (data []byte, write bool, err error)) error {
	// invariant: Rewrite takes no lock: its caller holds the one lock every
	// writer of that file takes, from its read to its write, or two writers
	// at once each write the text they read and one drops the other's change.
	// invariant: edit gets the bytes of the file the path names, a link
	// followed to its target (nil when there is none), and what it answers
	// replaces that target whole (FS.Replace), so a link stays a link.
	// Nothing is written when edit answers write false.
	target, err := linkTarget(files, at)
	if err != nil {
		return err
	}
	old, err := files.ReadFile(target)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	data, write, err := edit(old)
	if err != nil || !write {
		return err
	}
	return files.Replace(target, data, 0o644)
}

// value: how a replace pushes a file, then its folder, to the disk itself:
// (*os.File).Sync, which is fcntl(F_FULLFSYNC) on darwin (Go's
// internal/poll, fsync where the volume refuses it) and fsync(2) elsewhere.
type durable struct{ sync func(*os.File) error }

// invariant: a replace lands at once and survives a power cut: the data is
// written to a file of its own in the target's folder, given the target's
// mode, uid and gid, synced, closed, renamed over the target, and the
// folder synced, so a reader, a crash or a full disk sees the old file or
// the new one, never half of one. A file that is not there yet is made
// with perm.
func (d durable) replace(path string, data []byte, perm fs.FileMode) error {
	was, err := os.Stat(path)
	switch {
	case err == nil:
		perm = was.Mode().Perm()
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+strings.TrimPrefix(filepath.Base(path), ".")+".*.station")
	if errors.Is(err, fs.ErrPermission) && was != nil {
		return d.overwrite(path, data)
	}
	if err != nil {
		return err
	}
	err = d.fill(tmp, data, perm, was)
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		if errors.Is(err, errOwner) {
			return d.overwrite(path, data)
		}
		return err
	}
	return d.syncFolder(dir)
}

// value: the owner a new file cannot be given, so the target is
// overwritten where it stands, which keeps its own.
var errOwner = errors.New("the file's owner cannot be kept")

func (d durable) fill(f *os.File, data []byte, perm fs.FileMode, was fs.FileInfo) error {
	_, err := f.Write(data)
	if err == nil {
		err = f.Chmod(perm)
	}
	if err == nil {
		err = keepOwner(f, was)
	}
	if err == nil {
		err = d.sync(f)
	}
	return errors.Join(err, f.Close())
}

func keepOwner(f *os.File, was fs.FileInfo) error {
	if was == nil {
		return nil
	}
	want, ok := was.Sys().(*syscall.Stat_t)
	now, err := f.Stat()
	if !ok || err != nil {
		return err
	}
	have, ok := now.Sys().(*syscall.Stat_t)
	if !ok || (have.Uid == want.Uid && have.Gid == want.Gid) {
		return nil
	}
	err = f.Chown(int(want.Uid), int(want.Gid))
	if errors.Is(err, fs.ErrPermission) {
		return errOwner
	}
	return err
}

// hazard: an overwrite is not atomic: a crash in it leaves the file cut
// short. It is taken only where a replace cannot keep the file as it is, a
// folder station may not write in or an owner it may not give, and under
// the caller's lock (Rewrite) like any replace.
func (d durable) overwrite(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0) //nolint:gosec // outside: the file the caller names, there already.
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = d.sync(f)
	}
	return errors.Join(err, f.Close())
}

func (d durable) syncFolder(dir string) error {
	f, err := os.Open(dir) //nolint:gosec // outside: the folder of the file just replaced.
	if err != nil {
		return err
	}
	return errors.Join(d.sync(f), f.Close())
}

func linkTarget(files FS, at string) (string, error) {
	for range linkHops {
		to, link := readlink(files, at)
		if !link {
			return at, nil
		}
		if !filepath.IsAbs(to) {
			to = filepath.Join(filepath.Dir(at), to)
		}
		at = to
	}
	return "", &fs.PathError{Op: "rewrite", Path: at, Err: errors.New("too many links")}
}

func readlink(files FS, at string) (string, bool) {
	to, err := files.Readlink(at)
	return to, err == nil
}
