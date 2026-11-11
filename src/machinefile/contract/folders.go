package contract

import (
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/machinefile"
)

func ReadFolders(dir folders, catalogAt, machinefilesAt string) ([]machinefile.File, error) {
	// invariant: a source on folders holds its catalog's items one folder
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
			continue
		}
		at := path.Join(catalogAt, folder.Name())
		entries, listed := list(dir, at)
		if listed != nil {
			return nil, listed
		}
		if out, listed = add(out, dir, "catalog/"+folder.Name(), at, entries); listed != nil {
			return nil, listed
		}
	}
	entries, err := list(dir, machinefilesAt)
	if err != nil {
		return nil, err
	}
	if out, err = add(out, dir, "machinefiles", machinefilesAt, entries); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b machinefile.File) int { return strings.Compare(a.Path, b.Path) })
	return out, nil
}

// value: the platform's files, or an embedded tree.
type folders interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

func list(dir folders, at string) ([]fs.DirEntry, error) {
	entries, err := dir.ReadDir(at)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return entries, err
}

func add(out []machinefile.File, dir folders, under, at string, entries []fs.DirEntry) ([]machinefile.File, error) {
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		body, err := dir.ReadFile(path.Join(at, e.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, machinefile.File{Path: under + "/" + e.Name(), Body: body})
	}
	return out, nil
}
