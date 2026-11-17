package files

import (
	"context"
	"io/fs"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

// value: an embedded tree as the folders a source is read from.

type Embedded struct{ FS fs.FS }

func (e Embedded) ReadDir(name string) ([]fs.DirEntry, error) { return fs.ReadDir(e.FS, name) }
func (e Embedded) ReadFile(name string) ([]byte, error)       { return fs.ReadFile(e.FS, name) }
func (e Embedded) Stat(name string) (fs.FileInfo, error)      { return fs.Stat(e.FS, name) }

// value: a source of folders: where its catalog and its Machinefiles are,
// its origin, and the words a reader names it by.

// value: the platform's files, or an embedded tree.
type folders interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Stat(name string) (fs.FileInfo, error)
}

type Source struct {
	dir folders
	// value: the links its folders may hold, which its caller says: the
	// owner's own allow a link to a file.
	links                 contract.Links
	catalog, machinefiles string
	origin                machinefile.OriginKind
	root                  string
}

func New(dir folders, links contract.Links, catalogAt, machinefilesAt string, origin machinefile.OriginKind, root string) Source {
	return Source{dir: dir, links: links, catalog: catalogAt, machinefiles: machinefilesAt, origin: origin, root: root}
}

func (s Source) Read(context.Context) (machinefile.Tree, error) {
	files, err := contract.ReadFolders(s.dir, s.links, s.catalog, s.machinefiles)
	return machinefile.Tree{Kind: s.origin, Root: s.root, Files: files}, err
}
