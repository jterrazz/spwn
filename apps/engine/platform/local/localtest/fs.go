package localtest

import (
	"context"
	"io/fs"
	"maps"
	"path"
	"strings"
	"testing/fstest"
	"time"
)

type FS struct {
	Files fstest.MapFS
}

func NewFS() *FS { return &FS{Files: fstest.MapFS{}} }

func key(p string) string { return strings.TrimPrefix(path.Clean(p), "/") }

func (f *FS) Open(name string) (fs.File, error) { return f.Files.Open(key(name)) }

func (f *FS) ReadFile(p string) ([]byte, error) { return f.Files.ReadFile(key(p)) }

func (f *FS) Readlink(string) (string, error) { return "", fs.ErrNotExist }

func (f *FS) WriteFile(p string, data []byte, perm fs.FileMode) error {
	f.Files[key(p)] = &fstest.MapFile{Data: append([]byte{}, data...), Mode: perm, ModTime: time.Unix(0, 0)}
	return nil
}

func (f *FS) Replace(p string, data []byte, perm fs.FileMode) error {
	if file, ok := f.Files[key(p)]; ok {
		perm = file.Mode.Perm()
	}
	return f.WriteFile(p, data, perm)
}

func (f *FS) MkdirAll(p string, perm fs.FileMode) error {
	if _, ok := f.Files[key(p)]; !ok {
		f.Files[key(p)] = &fstest.MapFile{Mode: fs.ModeDir | perm}
	}
	return nil
}

func (f *FS) Remove(p string) error {
	if _, ok := f.Files[key(p)]; !ok {
		return &fs.PathError{Op: "remove", Path: p, Err: fs.ErrNotExist}
	}
	delete(f.Files, key(p))
	return nil
}

func (f *FS) RemoveAll(p string) error {
	k := key(p)
	for name := range f.Files {
		if name == k || strings.HasPrefix(name, k+"/") {
			delete(f.Files, name)
		}
	}
	return nil
}

func (f *FS) Stat(p string) (fs.FileInfo, error) { return f.Files.Stat(key(p)) }

func (f *FS) ReadDir(p string) ([]fs.DirEntry, error) { return f.Files.ReadDir(key(p)) }

func (f *FS) Glob(pattern string) ([]string, error) {
	matches, err := f.Files.Glob(key(pattern))
	for i := range matches {
		matches[i] = "/" + matches[i]
	}
	return matches, err
}

func (f *FS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return fs.WalkDir(f.Files, key(root), func(p string, d fs.DirEntry, err error) error {
		return fn("/"+p, d, err)
	})
}

func (f *FS) Sizes(ctx context.Context, roots []string, each func(int, int64)) {
	for i, p := range roots {
		if ctx.Err() != nil {
			each(i, -1)
			continue
		}
		var total int64
		k := key(p)
		for name, file := range f.Files {
			if strings.HasPrefix(name, k+"/") && !file.Mode.IsDir() {
				total += int64(len(file.Data))
			}
		}
		each(i, total)
	}
}

func (f *FS) Chmod(p string, perm fs.FileMode) error {
	file, ok := f.Files[key(p)]
	if !ok {
		return &fs.PathError{Op: "chmod", Path: p, Err: fs.ErrNotExist}
	}
	file.Mode = file.Mode.Type() | perm
	return nil
}

func (f *FS) Chown(string, int, int) error { return nil }

func (f *FS) Rename(from, to string) error {
	k, dst := key(from), key(to)
	if _, ok := f.Files[k]; !ok {
		return &fs.PathError{Op: "rename", Path: from, Err: fs.ErrNotExist}
	}
	moved := fstest.MapFS{}
	for name, file := range f.Files {
		if name == k || strings.HasPrefix(name, k+"/") {
			moved[dst+strings.TrimPrefix(name, k)] = file
			delete(f.Files, name)
		}
	}
	maps.Copy(f.Files, moved)
	return nil
}

type Disk struct {
	// value: a disk over the in-memory files, for a service that sizes and
	// erases through one; no process runs on it.
	*FS
}

func (d Disk) Volume(context.Context) (total, free int64, err error) { return 1 << 40, 1 << 39, nil }

func (d Disk) Erase(_ context.Context, paths []string) error {
	for _, p := range paths {
		if err := d.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

func (d Disk) Running(context.Context, string) bool { return false }

func (d Disk) Move(_ context.Context, from, to string) error { return d.Rename(from, to) }
