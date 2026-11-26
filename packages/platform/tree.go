package platform

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// ErrOutside is the error of a path a Tree refuses because it leaves the
// tree's folder: by "..", by an absolute path elsewhere, or by a NUL.
var ErrOutside = errors.New("leaves the project")

// ErrNotAFile is the error of a read a Tree refuses because the name holds
// neither a file nor a folder: a FIFO, a device or a socket.
var ErrNotAFile = errors.New("is neither a file nor a folder")

// ErrLink is the error of a name a Tree refuses because it, or a folder
// on its way, is a link whose target leaves the folder the Tree reads.
var ErrLink = errors.New("is a link out of its folder, refused: copy what it names into the folder")

// Tree is a project's folder, read through a root it cannot leave: a path
// that climbs out by "..", an absolute path elsewhere and a link whose
// target is outside are refused before anything outside is opened, and a
// read opens only a file or a folder, never waiting on a FIFO. Paths are
// absolute or relative to the folder, as the loaders build them.
type Tree struct {
	dir  string
	real string
	root *os.Root
	err  error
}

// Within opens dir once, as the root every read of the Tree goes through;
// dir itself may be a link, and an absolute path is under the Tree by
// dir or by the folder dir resolves to. A dir that does not open gives a
// Tree each read of which fails with that error, as a read of a missing
// folder would.
func Within(dir string) *Tree {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return &Tree{dir: dir, real: dir, err: err}
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		resolved = abs
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		return &Tree{dir: abs, real: resolved, err: &fs.PathError{Op: "open", Path: abs, Err: syscall.ENOTDIR}}
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return &Tree{dir: abs, real: resolved, err: err}
	}
	return &Tree{dir: abs, real: resolved, root: root}
}

// Close releases the Tree's root.
func (t *Tree) Close() error {
	if t.root == nil {
		return nil
	}
	return t.root.Close()
}

// Dir is the absolute folder the Tree reads.
func (t *Tree) Dir() string { return t.dir }

// Holds reports whether p names something under the Tree's folder, by its
// name alone: p is never opened to say so.
func (t *Tree) Holds(p string) bool {
	_, ok := t.under(p)
	return ok
}

func (t *Tree) under(p string) (string, bool) {
	if strings.ContainsRune(p, 0) {
		return "", false
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(t.dir, p)
	}
	if rel, ok := beneath(t.dir, p); ok {
		return rel, true
	}
	if t.real != t.dir {
		return beneath(t.real, p)
	}
	return "", false
}

func beneath(dir, p string) (string, bool) {
	rel, err := filepath.Rel(dir, filepath.Clean(p))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func namesAFolder(p string) bool {
	return strings.HasSuffix(p, string(filepath.Separator)) || p == "." || strings.HasSuffix(p, string(filepath.Separator)+".")
}

// Rel is p as a name under the Tree's folder, or ErrOutside when p leaves
// it.
func (t *Tree) Rel(p string) (string, error) {
	rel, ok := t.under(p)
	switch {
	case !ok:
		return "", &fs.PathError{Op: "open", Path: p, Err: ErrOutside}
	case t.root == nil:
		return "", &fs.PathError{Op: "open", Path: p, Err: t.err}
	}
	return rel, nil
}

// Stat is os.Stat of p inside the Tree.
func (t *Tree) Stat(p string) (fs.FileInfo, error) {
	rel, err := t.Rel(p)
	if err != nil {
		return nil, err
	}
	info, err := t.root.Stat(rel)
	if err == nil && namesAFolder(p) && !info.IsDir() {
		return nil, &fs.PathError{Op: "stat", Path: p, Err: syscall.ENOTDIR}
	}
	return info, err
}

// IsDir reports whether p is a folder inside the Tree.
func (t *Tree) IsDir(p string) bool {
	info, err := t.Stat(p)
	return err == nil && info.IsDir()
}

// IsFile reports whether p is a file inside the Tree.
func (t *Tree) IsFile(p string) bool {
	info, err := t.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

func (t *Tree) open(p string, folder bool) (*os.File, error) {
	rel, err := t.Rel(p)
	if err != nil {
		return nil, err
	}
	f, err := t.root.OpenFile(rel, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, t.linkOut(p, rel, err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() && (!folder || !info.IsDir()) {
		f.Close()
		return nil, &fs.PathError{Op: "open", Path: p, Err: ErrNotAFile}
	}
	if namesAFolder(p) && !info.IsDir() {
		f.Close()
		return nil, &fs.PathError{Op: "open", Path: p, Err: syscall.ENOTDIR}
	}
	return f, nil
}

// FS is the folder dir as an fs.FS whose every call reads through a Tree
// opened at dir for that call and closed after it, so the FS holds no
// root open between calls and refuses what the Tree refuses.
func FS(dir string) fs.FS { return treeFS{dir: dir} }

// FSUnder is the folder sub of dir as an fs.FS: each call opens a Tree at
// dir and, through it, a root at sub, so a link that leaves sub is refused
// even when its target is elsewhere under dir, and a sub that is a link
// out of dir is refused by dir's root.
func FSUnder(dir, sub string) fs.FS { return treeFS{dir: dir, sub: sub} }

type treeFS struct{ dir, sub string }

func (d treeFS) tree() *Tree {
	t := Within(d.dir)
	if d.sub == "" {
		return t
	}
	defer t.Close()
	return t.Sub(d.sub)
}

// Sub is the folder p of the Tree as a Tree of its own, opened through
// the Tree's root: p cannot leave the Tree, and a read of the Sub cannot
// leave p.
func (t *Tree) Sub(p string) *Tree {
	rel, err := t.Rel(p)
	if err != nil {
		return &Tree{dir: p, real: p, err: err}
	}
	dir := filepath.Join(t.dir, rel)
	root, err := t.root.OpenRoot(rel)
	if err != nil {
		return &Tree{dir: dir, real: dir, err: t.linkOut(p, rel, err)}
	}
	return &Tree{dir: dir, real: dir, root: root}
}

func (d treeFS) name(op, name string) (string, error) {
	if !fs.ValidPath(name) {
		return "", &fs.PathError{Op: op, Path: name, Err: fs.ErrInvalid}
	}
	return filepath.FromSlash(name), nil
}

func (d treeFS) Open(name string) (fs.File, error) {
	p, err := d.name("open", name)
	if err != nil {
		return nil, err
	}
	t := d.tree()
	defer t.Close()
	return t.open(p, true)
}

func (d treeFS) ReadFile(name string) ([]byte, error) {
	p, err := d.name("open", name)
	if err != nil {
		return nil, err
	}
	t := d.tree()
	defer t.Close()
	return t.ReadFile(p)
}

func (d treeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	p, err := d.name("open", name)
	if err != nil {
		return nil, err
	}
	t := d.tree()
	defer t.Close()
	return t.ReadDir(p)
}

func (d treeFS) Stat(name string) (fs.FileInfo, error) {
	p, err := d.name("stat", name)
	if err != nil {
		return nil, err
	}
	t := d.tree()
	defer t.Close()
	return t.Stat(p)
}

// ReadFile is os.ReadFile of the file p inside the Tree.
func (t *Tree) ReadFile(p string) ([]byte, error) {
	f, err := t.open(p, false)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// OpenFile is os.OpenFile of the file p inside the Tree, never blocking
// on a FIFO: a link whose target is outside is refused, never opened
// through, a dangling one included, and what opens is refused unless it
// is a regular file.
func (t *Tree) OpenFile(p string, flag int, perm fs.FileMode) (*os.File, error) {
	rel, err := t.Rel(p)
	if err != nil {
		return nil, err
	}
	f, err := t.root.OpenFile(rel, flag|syscall.O_NONBLOCK, perm)
	if err != nil {
		return nil, t.linkOut(p, rel, err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		f.Close()
		return nil, &fs.PathError{Op: "open", Path: p, Err: ErrNotAFile}
	}
	return f, nil
}

// WriteFile is os.WriteFile of the file p inside the Tree, opened by
// OpenFile.
func (t *Tree) WriteFile(p string, data []byte, perm fs.FileMode) error {
	f, err := t.OpenFile(p, os.O_WRONLY|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

// AppendFile appends data to the file p inside the Tree, creating it,
// opened by OpenFile.
func (t *Tree) AppendFile(p string, data []byte, perm fs.FileMode) error {
	f, err := t.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_APPEND, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// Rename is os.Rename of from to to, both inside the Tree: a link is
// renamed as itself, never followed.
func (t *Tree) Rename(from, to string) error {
	relFrom, err := t.Rel(from)
	if err != nil {
		return err
	}
	relTo, err := t.Rel(to)
	if err != nil {
		return err
	}
	return t.root.Rename(relFrom, relTo)
}

// Remove is os.Remove of p inside the Tree: a link is removed as itself.
func (t *Tree) Remove(p string) error {
	rel, err := t.Rel(p)
	if err != nil {
		return err
	}
	return t.root.Remove(rel)
}

// Lstat is os.Lstat of p inside the Tree: a link is described as itself.
func (t *Tree) Lstat(p string) (fs.FileInfo, error) {
	rel, err := t.Rel(p)
	if err != nil {
		return nil, err
	}
	return t.root.Lstat(rel)
}

// MkdirAll is os.MkdirAll of the folder p inside the Tree.
func (t *Tree) MkdirAll(p string, perm fs.FileMode) error {
	rel, err := t.Rel(p)
	if err != nil {
		return err
	}
	if err := t.root.MkdirAll(rel, perm); err != nil {
		return t.linkOut(p, rel, err)
	}
	return nil
}

// invariant: linkOut reads the links on rel alone, never their targets,
// so the refusal it names reads the same whatever is outside.
func (t *Tree) linkOut(p, rel string, err error) error {
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		return err
	}
	parts := strings.Split(filepath.Clean(rel), string(filepath.Separator))
	for i := range parts {
		prefix := filepath.Join(parts[:i+1]...)
		info, lerr := t.root.Lstat(prefix)
		if lerr != nil {
			return err
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return &fs.PathError{Op: "open", Path: prefix, Err: ErrLink}
		}
	}
	return err
}

// ReadDir is os.ReadDir of the folder p inside the Tree, sorted by name.
func (t *Tree) ReadDir(p string) ([]fs.DirEntry, error) {
	f, err := t.open(p, true)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, err
}

// WalkDir is filepath.WalkDir of p inside the Tree: fn sees each path as
// joined from p.
func (t *Tree) WalkDir(p string, fn fs.WalkDirFunc) error {
	info, err := t.Stat(p)
	if err != nil {
		err = fn(p, nil, err)
	} else {
		err = t.walk(p, fs.FileInfoToDirEntry(info), fn)
	}
	if errors.Is(err, fs.SkipDir) || errors.Is(err, fs.SkipAll) {
		return nil
	}
	return err
}

func (t *Tree) walk(p string, d fs.DirEntry, fn fs.WalkDirFunc) error {
	if err := fn(p, d, nil); err != nil || !d.IsDir() {
		if errors.Is(err, fs.SkipDir) && d.IsDir() {
			err = nil
		}
		return err
	}
	entries, err := t.ReadDir(p)
	if err != nil {
		if err = fn(p, d, err); err != nil {
			if errors.Is(err, fs.SkipDir) {
				err = nil
			}
			return err
		}
	}
	for _, e := range entries {
		if err := t.walk(filepath.Join(p, e.Name()), e, fn); err != nil {
			if errors.Is(err, fs.SkipDir) {
				break
			}
			return err
		}
	}
	return nil
}
