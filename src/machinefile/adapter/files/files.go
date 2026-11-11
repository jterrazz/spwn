package files

import (
	"context"
	"io/fs"

	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

// value: an embedded tree as the folders a source is read from.

type Embedded struct{ FS fs.FS }

func (e Embedded) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(e.FS, name) }
func (e Embedded) ReadFile(name string) ([]byte, error)       { return fs.ReadFile(e.FS, name) }

// value: a source of folders: where its catalog and its Machinefiles are,
// its layer, and the words a reader names it by.

// value: the platform's files, or an embedded tree.
type folders interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type Source struct {
	dir                   folders
	catalog, machinefiles string
	layer                 machinefile.Layer
	root                  string
}

func New(dir folders, catalogAt, machinefilesAt string, layer machinefile.Layer, root string) Source {
	return Source{dir: dir, catalog: catalogAt, machinefiles: machinefilesAt, layer: layer, root: root}
}

func (s Source) Read(context.Context) (machinefile.Tree, error) {
	files, err := contract.ReadFolders(s.dir, s.catalog, s.machinefiles)
	return machinefile.Tree{Layer: s.layer, Root: s.root, Files: files}, err
}
