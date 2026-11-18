package contract

import (
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

// value: the links a source's folders may hold: none, in a shared
// source's clone (its content is the remote's, and a link there is a door
// to any file of the machine); a link to a regular file, in the owner's
// own folders, which owners keep in a dotfiles repository.

type Links string

const (
	NoLinks   Links = "none"
	FileLinks Links = "to a file"
)

func ReadFolders(dir folders, links Links, catalogAt, machinefilesAt string) ([]machinefile.File, error) {
	// invariant: a source on folders holds its catalog's packages one folder
	// down, `<category>/<name>.yaml`, and its Machinefiles at the top of
	// theirs; a folder that is not there holds no file. The files come in
	// path order.
	var out []machinefile.File
	folders, err := list(dir, catalogAt)
	if err != nil {
		return nil, err
	}
	for _, folder := range folders {
		if !folder.IsDir() {
			if !meant(folder.Name()) {
				continue
			}
			if refused := regular(dir, links, catalogAt, folder); refused != nil {
				return nil, refused
			}
			continue
		}
		at := path.Join(catalogAt, folder.Name())
		entries, listed := list(dir, at)
		if listed != nil {
			return nil, listed
		}
		if out, listed = add(out, dir, links, "catalog/"+folder.Name(), at, entries); listed != nil {
			return nil, listed
		}
	}
	entries, err := list(dir, machinefilesAt)
	if err != nil {
		return nil, err
	}
	if out, err = add(out, dir, links, "machinefiles", machinefilesAt, entries); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b machinefile.File) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

// invariant: an entry is judged (its type, its content) only when its
// owner meant it to be read: a `.yaml` file, or a `.yml` one, which the
// decoder refuses rather than skips. Anything else beside them, whatever
// its type, is skipped.
func meant(name string) bool {
	ext := path.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

// value: the platform's files, or an embedded tree.
type folders interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Stat(name string) (fs.FileInfo, error)
}

func list(dir folders, at string) ([]fs.DirEntry, error) {
	entries, err := dir.ReadDir(at)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}

// invariant: nothing but a regular file is read (a device or a pipe never
// answers as a file does). A link is followed only where the caller allows
// it, and only to a regular file; a link to a folder, or any link of a
// source that allows none, is refused. Each refusal names its path.
func regular(dir folders, links Links, at string, e fs.DirEntry) error {
	p := path.Join(at, e.Name())
	switch t := e.Type(); {
	case t&fs.ModeSymlink != 0 && links == FileLinks:
		if info, err := dir.Stat(p); err != nil || !info.Mode().IsRegular() {
			return &catalog.FileError{File: p, Problem: "is a link to no file: a link in your folders points at a regular file"}
		}
	case t&fs.ModeSymlink != 0:
		return &catalog.FileError{File: p, Problem: "is a link: station reads a shared source's files as they are, never through a link"}
	case !t.IsRegular():
		return &catalog.FileError{File: p, Problem: "is no regular file: a source holds files and folders alone"}
	}
	return nil
}

func add(out []machinefile.File, dir folders, links Links, under, at string, entries []fs.DirEntry) ([]machinefile.File, error) {
	for _, e := range entries {
		if e.IsDir() || !meant(e.Name()) {
			continue
		}
		if refused := regular(dir, links, at, e); refused != nil {
			return nil, refused
		}
		body, err := dir.ReadFile(path.Join(at, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, machinefile.File{Path: under + "/" + e.Name(), Body: body})
	}
	return out, nil
}
