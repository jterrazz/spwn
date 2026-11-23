package local

import (
	"crypto/rand"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

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

func (Beneath) ReadFile(base, rel string) ([]byte, error) {
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return nil, err
	}
	defer closeFD(dir)
	fd, err := openat(dir, name, unix.O_RDONLY, at)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), at)
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, &fs.PathError{Op: "read", Path: at, Err: errNotFile}
	}
	return io.ReadAll(f)
}

var errNotFile = errors.New("is not a file")

func (Beneath) Replace(base, rel string, data []byte, perm fs.FileMode, o Owner) error {
	// invariant: rel is replaced whole, at once and durably: data goes to a
	// file of its own made beside it, synced, then renamed over it, and the
	// folder synced. A file there keeps its mode and owner; one not there
	// yet is made with perm and given o; a link or anything but a file
	// there is refused.
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return err
	}
	defer closeFD(dir)
	var st unix.Stat_t
	err = unix.Fstatat(dir, name, &st, unix.AT_SYMLINK_NOFOLLOW)
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
	fd, err := unix.Openat(dir, tmp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return &fs.PathError{Op: "replace", Path: at, Err: err}
	}
	err = fill(os.NewFile(uintptr(fd), path.Join(path.Dir(at), tmp)), data, perm, o)
	if err == nil {
		err = unix.Renameat(dir, tmp, dir, name)
	}
	if err != nil {
		_ = unix.Unlinkat(dir, tmp, 0)
		return &fs.PathError{Op: "replace", Path: at, Err: err}
	}
	return unix.Fsync(dir)
}

func fill(f *os.File, data []byte, perm fs.FileMode, o Owner) error {
	_, err := f.Write(data)
	if err == nil {
		err = f.Chmod(perm.Perm())
	}
	if err == nil {
		err = f.Chown(o.UID, o.GID)
	}
	if err == nil {
		err = f.Sync()
	}
	return errors.Join(err, f.Close())
}

func (Beneath) Remove(base, rel string) error {
	// value: rel taken away: a file, or a folder only while it is empty
	// (ErrNotEmpty otherwise); a link there is refused, never removed.
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return err
	}
	defer closeFD(dir)
	var st unix.Stat_t
	err = unix.Fstatat(dir, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	if err != nil {
		return &fs.PathError{Op: "remove", Path: at, Err: err}
	}
	flags := 0
	switch m := entryOf(&st).Mode; {
	case m&fs.ModeSymlink != 0:
		return &fs.PathError{Op: "remove", Path: at, Err: ErrLink}
	case m.IsDir():
		flags = unix.AT_REMOVEDIR
	case !m.IsRegular():
		return &fs.PathError{Op: "remove", Path: at, Err: errNotFile}
	}
	err = unix.Unlinkat(dir, name, flags)
	if errors.Is(err, unix.ENOTEMPTY) || errors.Is(err, unix.EEXIST) {
		err = ErrNotEmpty
	}
	if err != nil {
		return &fs.PathError{Op: "remove", Path: at, Err: err}
	}
	return nil
}

func (Beneath) Chmod(base, rel string, perm fs.FileMode) error {
	return opened(base, rel, "chmod", func(fd int) error { return unix.Fchmod(fd, uint32(perm.Perm())) })
}

// value: do run on the descriptor of rel, a file or a folder, opened with
// O_NOFOLLOW.
func opened(base, rel, op string, do func(fd int) error) error {
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return err
	}
	defer closeFD(dir)
	fd, err := openat(dir, name, unix.O_RDONLY, at)
	if err != nil {
		return err
	}
	defer closeFD(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return &fs.PathError{Op: op, Path: at, Err: err}
	}
	if m := entryOf(&st).Mode; !m.IsDir() && !m.IsRegular() {
		return &fs.PathError{Op: op, Path: at, Err: errors.New("is not a file or a folder")}
	}
	if err := do(fd); err != nil {
		return &fs.PathError{Op: op, Path: at, Err: err}
	}
	return nil
}

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
