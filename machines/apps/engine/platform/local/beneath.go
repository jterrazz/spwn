package local

import (
	"context"
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type Owner struct{ UID, GID int }

func Self() Owner {
	// value: the ids this process acts with, its effective user and group.
	return Owner{UID: os.Geteuid(), GID: os.Getegid()}
}

type Entry struct {
	// value: what a name is itself, a link never followed: its type and
	// permission bits, and its owner.
	Mode fs.FileMode
	Owner
}

var (
	ErrLink     = errors.New("is a link")
	ErrNotEmpty = errors.New("is a folder that is not empty")
)

type Beneath struct {
	// invariant: a path under base is reached from base one name at a time,
	// each opened with O_NOFOLLOW, so a link anywhere below base is refused
	// (ErrLink), never followed, even one swapped in between two calls; a
	// mode or an owner is changed through the descriptor of what was opened,
	// never through a path. base itself is the caller's to trust.
}

func (Beneath) Lstat(base, rel string) (Entry, error) {
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return Entry{}, err
	}
	defer closeFD(dir)
	var st unix.Stat_t
	if err := unix.Fstatat(dir, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return Entry{}, &fs.PathError{Op: "lstat", Path: at, Err: err}
	}
	return entryOf(&st), nil
}

func (Beneath) Mkdir(base, rel string, perm fs.FileMode, o Owner) error {
	// value: the folder rel made with perm and given o; one there already is
	// refused (fs.ErrExist).
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return err
	}
	defer closeFD(dir)
	err = unix.Mkdirat(dir, name, 0o700)
	if err != nil {
		return &fs.PathError{Op: "mkdir", Path: at, Err: err}
	}
	fd, err := openat(dir, name, unix.O_RDONLY|unix.O_DIRECTORY, at)
	if err != nil {
		return err
	}
	defer closeFD(fd)
	if err := unix.Fchmod(fd, uint32(perm.Perm())); err != nil {
		return &fs.PathError{Op: "chmod", Path: at, Err: err}
	}
	if err := unix.Fchown(fd, o.UID, o.GID); err != nil {
		return &fs.PathError{Op: "chown", Path: at, Err: err}
	}
	return nil
}

type Folder struct {
	// value: a folder opened once, from base one name at a time with
	// O_NOFOLLOW, and the folder that holds it: every name in it is then
	// reached through its descriptor, never through a path, so a rename of the
	// folder or of a folder above it after the open moves nothing it reads or
	// writes.
	fd, up   int
	name, at string
}

func (Beneath) Open(base, rel string) (*Folder, error) {
	up, name, at, err := parent(base, rel)
	if err != nil {
		return nil, err
	}
	fd, err := openat(up, name, unix.O_RDONLY|unix.O_DIRECTORY, at)
	if err != nil {
		closeFD(up)
		return nil, err
	}
	return &Folder{fd: fd, up: up, name: name, at: at}, nil
}

func (b Beneath) Hold(ctx context.Context, base, rel string, within time.Duration) (folder *Folder, held bool, err error) {
	// invariant: the lock is flock(2)'s on the folder itself, exclusive and
	// advisory, held until Close: root and the folder's owner both take it,
	// a rename in the folder leaves the folder standing, and a process that
	// dies lets go of it. It is tried every tenth of a second; one another
	// holder keeps past within is answered held, and a wait ctx ends first
	// answers ctx's error.
	held, err = tried(ctx, within, func() (bool, error) {
		f, opened := b.Open(base, rel)
		if opened != nil {
			return false, opened
		}
		locked := unix.Flock(f.fd, unix.LOCK_EX|unix.LOCK_NB)
		switch {
		case errors.Is(locked, unix.EWOULDBLOCK):
			f.Close()
			return true, nil
		case locked != nil:
			f.Close()
			return false, &fs.PathError{Op: "lock", Path: f.at, Err: locked}
		}
		folder = f
		return false, nil
	})
	return folder, held, err
}

func (f *Folder) Close() {
	closeFD(f.fd)
	closeFD(f.up)
}

func (f *Folder) Path() string { return f.at }

func (f *Folder) Stat() (Entry, error) {
	var st unix.Stat_t
	if err := unix.Fstat(f.fd, &st); err != nil {
		return Entry{}, &fs.PathError{Op: "stat", Path: f.at, Err: err}
	}
	return entryOf(&st), nil
}

func (f *Folder) Chmod(perm fs.FileMode) error {
	if err := unix.Fchmod(f.fd, uint32(perm.Perm())); err != nil {
		return &fs.PathError{Op: "chmod", Path: f.at, Err: err}
	}
	return nil
}

func (f *Folder) Lstat(name string) (Entry, error) {
	var st unix.Stat_t
	if err := unix.Fstatat(f.fd, name, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return Entry{}, &fs.PathError{Op: "lstat", Path: path.Join(f.at, name), Err: err}
	}
	return entryOf(&st), nil
}

func (f *Folder) ReadFile(name string) ([]byte, Entry, error) {
	// value: the file name holds, opened with O_NOFOLLOW, and what the same
	// descriptor is: anything but a file is refused.
	at := path.Join(f.at, name)
	fd, err := openat(f.fd, name, unix.O_RDONLY, at)
	if err != nil {
		return nil, Entry{}, err
	}
	file := os.NewFile(uintptr(fd), at)
	defer func() { _ = file.Close() }()
	var st unix.Stat_t
	if err = unix.Fstat(fd, &st); err != nil {
		return nil, Entry{}, &fs.PathError{Op: "read", Path: at, Err: err}
	}
	e := entryOf(&st)
	if !e.Mode.IsRegular() {
		return nil, Entry{}, &fs.PathError{Op: "read", Path: at, Err: errNotFile}
	}
	data, err := io.ReadAll(file)
	return data, e, err
}

var errNotFile = errors.New("is not a file")

func (f *Folder) ChmodFile(name string, perm fs.FileMode) error {
	// value: the mode of the file name holds set to perm through the
	// descriptor of what was opened.
	at := path.Join(f.at, name)
	fd, err := openat(f.fd, name, unix.O_RDONLY, at)
	if err != nil {
		return err
	}
	defer closeFD(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return &fs.PathError{Op: "chmod", Path: at, Err: err}
	}
	if !entryOf(&st).Mode.IsRegular() {
		return &fs.PathError{Op: "chmod", Path: at, Err: errNotFile}
	}
	if err := unix.Fchmod(fd, uint32(perm.Perm())); err != nil {
		return &fs.PathError{Op: "chmod", Path: at, Err: err}
	}
	return nil
}

func (f *Folder) Replace(name string, data []byte, perm fs.FileMode, o Owner) error {
	// invariant: name is replaced whole, at once and durably: data goes to a
	// file of its own made beside it, synced, then renamed over it, and the
	// folder synced. A file there keeps its mode and owner; one not there
	// yet is made with perm and given o; a link or anything but a file
	// there is refused.
	at := path.Join(f.at, name)
	var st unix.Stat_t
	err := unix.Fstatat(f.fd, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	switch e := entryOf(&st); {
	case err == nil && e.Mode&fs.ModeSymlink != 0:
		return &fs.PathError{Op: "replace", Path: at, Err: ErrLink}
	case err == nil && !e.Mode.IsRegular():
		return &fs.PathError{Op: "replace", Path: at, Err: errNotFile}
	case err == nil:
		perm, o = e.Mode.Perm(), e.Owner
	case !errors.Is(err, unix.ENOENT):
		return &fs.PathError{Op: "replace", Path: at, Err: err}
	}
	tmp := "." + name + "." + rand.Text() + ".station"
	fd, err := unix.Openat(f.fd, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return &fs.PathError{Op: "replace", Path: at, Err: err}
	}
	err = fill(os.NewFile(uintptr(fd), path.Join(f.at, tmp)), data, perm, o)
	if err == nil {
		err = unix.Renameat(f.fd, tmp, f.fd, name)
	}
	if err != nil {
		_ = unix.Unlinkat(f.fd, tmp, 0)
		return &fs.PathError{Op: "replace", Path: at, Err: err}
	}
	return unix.Fsync(f.fd)
}

func fill(file *os.File, data []byte, perm fs.FileMode, o Owner) error {
	_, err := file.Write(data)
	if err == nil {
		err = file.Chmod(perm.Perm())
	}
	if err == nil {
		err = file.Chown(o.UID, o.GID)
	}
	if err == nil {
		err = file.Sync()
	}
	return errors.Join(err, file.Close())
}

func (f *Folder) Remove(name string) error {
	// value: the file name holds taken away; a link or anything but a file
	// there is refused, never removed.
	at := path.Join(f.at, name)
	e, err := f.Lstat(name)
	switch {
	case err != nil:
		return err
	case e.Mode&fs.ModeSymlink != 0:
		return &fs.PathError{Op: "remove", Path: at, Err: ErrLink}
	case !e.Mode.IsRegular():
		return &fs.PathError{Op: "remove", Path: at, Err: errNotFile}
	}
	if err := unix.Unlinkat(f.fd, name, 0); err != nil {
		return &fs.PathError{Op: "remove", Path: at, Err: err}
	}
	return nil
}

func (f *Folder) RemoveSelf() error {
	// value: the folder itself taken away while it is empty (ErrNotEmpty
	// otherwise), only while its name still holds the folder that was opened.
	var held, there unix.Stat_t
	if err := unix.Fstat(f.fd, &held); err != nil {
		return &fs.PathError{Op: "remove", Path: f.at, Err: err}
	}
	if err := unix.Fstatat(f.up, f.name, &there, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return &fs.PathError{Op: "remove", Path: f.at, Err: err}
	}
	if held.Dev != there.Dev || held.Ino != there.Ino {
		return &fs.PathError{Op: "remove", Path: f.at, Err: errMoved}
	}
	err := unix.Unlinkat(f.up, f.name, unix.AT_REMOVEDIR)
	if errors.Is(err, unix.ENOTEMPTY) || errors.Is(err, unix.EEXIST) {
		err = ErrNotEmpty
	}
	if err != nil {
		return &fs.PathError{Op: "remove", Path: f.at, Err: err}
	}
	return nil
}

var errMoved = errors.New("is no longer the folder that was opened")

// value: the folder that holds rel's last name, opened from base through
// every name before it, and that last name.
func parent(base, rel string) (dir int, name, at string, err error) {
	names, err := namesOf(rel)
	if err != nil {
		return -1, "", "", &fs.PathError{Op: "open", Path: path.Join(base, rel), Err: err}
	}
	dir, err = unix.Open(base, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, "", "", &fs.PathError{Op: "open", Path: base, Err: err}
	}
	at = base
	for _, n := range names[:len(names)-1] {
		at = path.Join(at, n)
		next, err := openat(dir, n, unix.O_RDONLY|unix.O_DIRECTORY, at)
		closeFD(dir)
		if err != nil {
			return -1, "", "", err
		}
		dir = next
	}
	last := names[len(names)-1]
	return dir, last, path.Join(at, last), nil
}

func namesOf(rel string) ([]string, error) {
	var out []string
	for n := range strings.SplitSeq(rel, "/") {
		switch n {
		case "", ".":
			continue
		case "..":
			return nil, fs.ErrInvalid
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fs.ErrInvalid
	}
	return out, nil
}

// value: name opened in dir with O_NOFOLLOW; a link answers ErrLink.
func openat(dir int, name string, flags int, at string) (int, error) {
	fd, err := unix.Openat(dir, name, flags|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err == nil {
		return fd, nil
	}
	var st unix.Stat_t
	if (errors.Is(err, unix.ELOOP) || errors.Is(err, unix.ENOTDIR)) && unix.Fstatat(dir, name, &st, unix.AT_SYMLINK_NOFOLLOW) == nil && entryOf(&st).Mode&fs.ModeSymlink != 0 {
		err = ErrLink
	}
	return -1, &fs.PathError{Op: "open", Path: at, Err: err}
}

func entryOf(st *unix.Stat_t) Entry {
	m := fs.FileMode(st.Mode & 0o777)
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFDIR:
		m |= fs.ModeDir
	case unix.S_IFLNK:
		m |= fs.ModeSymlink
	case unix.S_IFREG:
	default:
		m |= fs.ModeIrregular
	}
	return Entry{Mode: m, Owner: Owner{UID: int(st.Uid), GID: int(st.Gid)}}
}

func closeFD(fd int) { _ = unix.Close(fd) }
