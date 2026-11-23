package local

import (
	"bytes"
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
	// permission bits, its owner, and how many names the system gives it.
	Mode  fs.FileMode
	Nlink int
	Owner
}

var (
	ErrLink     = errors.New("is a link")
	ErrNotEmpty = errors.New("is a folder that is not empty")
	// value: a file with another name, a hard link, which a write or a mode
	// change through this name would reach too.
	ErrHardLink = errors.New("has another name, a hard link")
	// value: a file whose bytes are no longer those its writer judged its
	// write on.
	ErrChanged = errors.New("changed since it was read")
	// value: a file another user owns, which a Folder never chmods.
	ErrOwner = errors.New("is another user's")
	// value: a file larger than MaxRead, which a Folder never reads.
	ErrTooLarge = errors.New("is larger than 1 MiB")
)

const MaxRead = 1 << 20

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

func (Beneath) Mkdir(base, rel string, perm fs.FileMode, o Owner) (err error) {
	// value: the folder rel made with perm and given o; one there already is
	// refused (fs.ErrExist). Its mode and owner are set only through a
	// descriptor proved to be the folder just made, this process's, empty,
	// the one its name held right after it was made; anything else is
	// refused with nothing touched, and a folder made that fails after
	// that proof is taken away again.
	dir, name, at, err := parent(base, rel)
	if err != nil {
		return err
	}
	defer closeFD(dir)
	err = unix.Mkdirat(dir, name, 0o700)
	if err != nil {
		return &fs.PathError{Op: "mkdir", Path: at, Err: err}
	}
	mine := false
	defer func() {
		if err != nil && mine {
			_ = unix.Unlinkat(dir, name, unix.AT_REMOVEDIR)
		}
	}()
	var made, held unix.Stat_t
	if err = unix.Fstatat(dir, name, &made, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return &fs.PathError{Op: "mkdir", Path: at, Err: err}
	}
	fd, err := openat(dir, name, unix.O_RDONLY|unix.O_DIRECTORY, at)
	if err != nil {
		return err
	}
	defer closeFD(fd)
	if err = unix.Fstat(fd, &held); err != nil {
		return &fs.PathError{Op: "mkdir", Path: at, Err: err}
	}
	if err = fresh(fd, &made, &held); err != nil {
		return &fs.PathError{Op: "mkdir", Path: at, Err: err}
	}
	mine = true
	if err = unix.Fchmod(fd, uint32(perm.Perm())); err != nil {
		return &fs.PathError{Op: "chmod", Path: at, Err: err}
	}
	if err = unix.Fchown(fd, o.UID, o.GID); err != nil {
		return &fs.PathError{Op: "chown", Path: at, Err: err}
	}
	return nil
}

var errNotMade = errors.New("is not the folder just made")

// value: nil when held, opened through fd, is made, the folder its name
// held right after Mkdirat, still this process's and empty.
func fresh(fd int, made, held *unix.Stat_t) error {
	if made.Mode&unix.S_IFMT != unix.S_IFDIR || held.Dev != made.Dev || held.Ino != made.Ino || int(held.Uid) != os.Geteuid() {
		return errNotMade
	}
	dup, err := unix.Dup(fd)
	if err != nil {
		return err
	}
	folder := os.NewFile(uintptr(dup), "")
	defer func() { _ = folder.Close() }()
	if _, err := unix.Seek(dup, 0, io.SeekStart); err != nil {
		return err
	}
	if _, err := folder.Readdirnames(1); !errors.Is(err, io.EOF) {
		return errNotMade
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
	// descriptor is: anything but a file of one name is refused, and so is
	// one larger than MaxRead, never read past it.
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
	if refused := single(e); refused != nil {
		return nil, Entry{}, &fs.PathError{Op: "read", Path: at, Err: refused}
	}
	if st.Size > MaxRead {
		return nil, Entry{}, &fs.PathError{Op: "read", Path: at, Err: ErrTooLarge}
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxRead+1))
	if err == nil && len(data) > MaxRead {
		return nil, Entry{}, &fs.PathError{Op: "read", Path: at, Err: ErrTooLarge}
	}
	return data, e, err
}

var errNotFile = errors.New("is not a file")

func single(e Entry) error {
	switch {
	case !e.Mode.IsRegular():
		return errNotFile
	case e.Nlink != 1:
		return ErrHardLink
	}
	return nil
}

func (f *Folder) ChmodFile(name string, uid int, perm func(fs.FileMode) fs.FileMode) error {
	// value: the mode of the file name holds set to perm of its mode, judged
	// and set through the one descriptor of what was opened, a file of one
	// name that uid owns; anything else is refused, never chmodded.
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
	e := entryOf(&st)
	if err := single(e); err != nil {
		return &fs.PathError{Op: "chmod", Path: at, Err: err}
	}
	if e.UID != uid {
		return &fs.PathError{Op: "chmod", Path: at, Err: ErrOwner}
	}
	want := perm(e.Mode.Perm()).Perm()
	if want == e.Mode.Perm() {
		return nil
	}
	if err := unix.Fchmod(fd, uint32(want)); err != nil {
		return &fs.PathError{Op: "chmod", Path: at, Err: err}
	}
	return nil
}

type Seen struct {
	// value: what a writer read of a name before it judged its write there:
	// its bytes, or that nothing was there.
	Data  []byte
	There bool
}

func (f *Folder) Replace(name string, seen Seen, data []byte, perm fs.FileMode, o Owner) error {
	// invariant: name is replaced whole, at once and durably: data goes to a
	// file of its own made beside it, synced, then renamed over it, and the
	// folder synced. A file there keeps its mode and owner; one not there
	// yet is made with perm and given o; a link, a file with another name
	// or anything but a file there is refused, and so is a name that no
	// longer holds what was seen, read again just before the rename
	// (ErrChanged).
	at := path.Join(f.at, name)
	var st unix.Stat_t
	err := unix.Fstatat(f.fd, name, &st, unix.AT_SYMLINK_NOFOLLOW)
	switch e := entryOf(&st); {
	case err == nil && e.Mode&fs.ModeSymlink != 0:
		return &fs.PathError{Op: "replace", Path: at, Err: ErrLink}
	case err == nil && single(e) != nil:
		return &fs.PathError{Op: "replace", Path: at, Err: single(e)}
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
		err = f.still(name, seen)
	}
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

func (f *Folder) Remove(name string, seen Seen) error {
	// value: the file name holds taken away; a link, anything but a file, or
	// a file that no longer holds what was seen, read again just before the
	// unlink (ErrChanged), is refused, never removed.
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
	if err := f.still(name, seen); err != nil {
		return &fs.PathError{Op: "remove", Path: at, Err: err}
	}
	if err := unix.Unlinkat(f.fd, name, 0); err != nil {
		return &fs.PathError{Op: "remove", Path: at, Err: err}
	}
	return nil
}

// value: nil while name holds what was seen, the same bytes or nothing;
// ErrChanged once it holds anything else.
func (f *Folder) still(name string, seen Seen) error {
	data, _, err := f.ReadFile(name)
	switch {
	case errors.Is(err, fs.ErrNotExist) && !seen.There:
		return nil
	case errors.Is(err, fs.ErrNotExist):
		return ErrChanged
	case err != nil:
		return err
	case !seen.There || !bytes.Equal(data, seen.Data):
		return ErrChanged
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
	return Entry{Mode: m, Nlink: int(st.Nlink), Owner: Owner{UID: int(st.Uid), GID: int(st.Gid)}}
}

func closeFD(fd int) { _ = unix.Close(fd) }
