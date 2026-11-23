package local

import (
	"errors"
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

var ErrLink = errors.New("is a link")

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

func (Beneath) Chmod(base, rel string, perm fs.FileMode) error {
	return opened(base, rel, "chmod", func(fd int) error { return unix.Fchmod(fd, uint32(perm.Perm())) })
}

func (Beneath) Chown(base, rel string, o Owner) error {
	return opened(base, rel, "chown", func(fd int) error { return unix.Fchown(fd, o.UID, o.GID) })
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
