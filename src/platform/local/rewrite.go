package local

import (
	"errors"
	"io/fs"
	"path/filepath"
)

// value: how many links a rewrite follows to the file they name before it
// gives up, as the kernel's own bound on a path does.
const linkHops = 32

func Rewrite(files FS, at string, edit func(old []byte) (data []byte, write bool, err error)) error {
	// invariant: a file rewritten whole lands at once: edit gets its bytes (nil
	// when there is none), and what it answers is written beside the file the
	// path names, a link followed to its target, then renamed over it, with the
	// file's own permission bits, so a reader, a crash or a full disk sees the
	// old file or the new one, never half of one, and a link stays a link.
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
	mode := fs.FileMode(0o644)
	if info, statErr := files.Stat(target); statErr == nil {
		mode = info.Mode().Perm()
	}
	tmp := filepath.Join(filepath.Dir(target), "."+filepath.Base(target)+".station")
	_ = files.Remove(tmp)
	err = files.WriteFile(tmp, data, mode)
	if err == nil {
		err = files.Chmod(tmp, mode)
	}
	if err == nil {
		err = files.Rename(tmp, target)
	}
	if err != nil {
		_ = files.Remove(tmp)
	}
	return err
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
