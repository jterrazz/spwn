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

// Tree is a project's folder, read through a root it cannot leave: a path
// that climbs out by "..", an absolute path elsewhere and a link whose
// target is outside are refused before anything outside is opened, and a
// read opens only a file or a folder, never waiting on a FIFO. Paths are
// absolute or relative to the folder, as the loaders build them.
type Tree struct {
	dir  string
	root *os.Root
	err  error
}

// Within opens dir once, as the root every read of the Tree goes through;
// dir itself may be a link. A dir that does not open gives a Tree each
// read of which fails with that error, as a read of a missing folder would.
func Within(dir string) *Tree {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return &Tree{dir: dir, err: err}
	}
	root, err := os.OpenRoot(abs)
	if err != nil {
		return &Tree{dir: abs, err: err}
	}
	return &Tree{dir: abs, root: root}
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
	rel, err := filepath.Rel(t.dir, filepath.Clean(p))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
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
	return t.root.Stat(rel)
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
		return nil, err
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
	return f, nil
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

// Open opens the file p inside the Tree for reading, as ReadFile refuses
// what it refuses, so a caller sizes and reads one opened file.
func (t *Tree) Open(p string) (fs.File, error) { return t.open(p, false) }
