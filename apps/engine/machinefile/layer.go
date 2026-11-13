package machinefile

import (
	"errors"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// value: where a value was read: the owner's overlay, the owner's own
// files, a shared source, station's built-in files; this machine, for a
// system a Machinefile does not say; the catalog, for an item a step
// brings in; and the lock, for the method and version it holds an app by.

type Layer string

const (
	Overlay     Layer = "overlay"
	User        Layer = "user"
	Shared      Layer = "shared"
	Builtin     Layer = catalog.BuiltinLayer
	ThisMachine Layer = "this machine"
	Catalog     Layer = "catalog"
	Locked      Layer = "lock"
)

// value: one source's files: its layer, the words a reader names it by
// (`~/.config/station`, `studio@1a2b3c4`, `built-in`), and its files by
// slash path under it, `catalog/<category>/<name>.yaml` and
// `machinefiles/<name>.yaml`.

type Tree struct {
	Layer Layer
	Root  string
	Files []File
	// value: a shared source's name and the commit its files were read
	// at, as the lock records it; nil for every other layer.
	Source *catalog.LockSource
}

type File struct {
	Path string
	Body []byte
	// value: where a reader finds the file when it is not under its tree's
	// root: a Machinefile named by its own path, read where it is.
	At string
}

const (
	catalogDir      = "catalog"
	machinefilesDir = "machinefiles"
)

// value: a Machinefile and where it was read.

type Placed struct {
	File  catalog.Machinefile
	Layer Layer
	// value: the file as a reader finds it: its tree's root and its path;
	// and its bytes, as its source holds them.
	Path string
	Body []byte
}

// value: every source read: the catalog in its layers, the Machinefiles by
// name (the strongest layer's), and the overlays by the name they override.

type Library struct {
	Catalog      catalog.Catalog
	Machinefiles map[string]Placed
	Overlays     map[string]Placed
}

// invariant: a refusal of a file names which format it is in, so a
// catalog item refused reads apart from a Machinefile refused.

type RefusedFileError struct {
	// value: "catalog" for an item file, "machinefile" for a Machinefile.
	Format string
	Err    error
}

func (e *RefusedFileError) Error() string { return e.Err.Error() }
func (e *RefusedFileError) Unwrap() error { return e.Err }

func refusedFile(format string, err error) error {
	var file *catalog.FileError
	var newer *catalog.NewerSchemaError
	if errors.As(err, &file) || errors.As(err, &newer) {
		return &RefusedFileError{Format: format, Err: err}
	}
	return err
}

func (t Tree) display(f File) string {
	if f.At != "" {
		return f.At
	}
	return t.Root + "/" + f.Path
}

// invariant: a source's folders hold its YAML files and whatever else sits
// beside them (a README, a lockfile a later station writes): only a
// `.yaml` file is read, and a `.yml` one is refused rather than skipped,
// since its owner meant it to be read.
func read(p string) bool {
	ext := path.Ext(p)
	return ext == ".yaml" || ext == ".yml"
}

func CatalogOf(trees []Tree) (catalog.Catalog, error) {
	// invariant: the trees strongest first; a same-name item of a stronger
	// layer replaces a weaker one whole, as a same-name Machinefile does.
	layers := make([][]catalog.Placed, 0, len(trees))
	for _, t := range trees {
		var items []catalog.Placed
		for _, f := range t.Files {
			rest, ok := strings.CutPrefix(f.Path, catalogDir+"/")
			if !ok {
				continue
			}
			if !read(rest) {
				continue
			}
			folder, name, ok := strings.Cut(rest, "/")
			if !ok || strings.Contains(name, "/") || !strings.HasSuffix(name, ".yaml") {
				return catalog.Catalog{}, refusedFile(catalogDir, &catalog.FileError{File: t.display(f), Problem: "is no item file: an item is catalog/<category>/<name>.yaml"})
			}
			p, err := catalog.Read(t.display(f), folder, string(t.Layer), f.Body)
			if err != nil {
				return catalog.Catalog{}, refusedFile(catalogDir, err)
			}
			items = append(items, p)
		}
		layers = append(layers, items)
	}
	c, err := catalog.Assemble(catalog.Over(layers...))
	return c, refusedFile(catalogDir, err)
}

func Load(trees []Tree) (Library, error) {
	c, err := CatalogOf(trees)
	if err != nil {
		return Library{}, err
	}
	lib := Library{Catalog: c, Machinefiles: map[string]Placed{}, Overlays: map[string]Placed{}}
	for _, t := range trees {
		for _, f := range t.Files {
			rest, ok := strings.CutPrefix(f.Path, machinefilesDir+"/")
			if !ok {
				continue
			}
			if !read(rest) {
				continue
			}
			file := t.display(f)
			if strings.Contains(rest, "/") || !strings.HasSuffix(rest, ".yaml") {
				return Library{}, refusedFile("machinefile", &catalog.FileError{File: file, Problem: "is no Machinefile: a Machinefile is machinefiles/<name>.yaml"})
			}
			mf, err := catalog.DecodeMachinefile(file, f.Body)
			if err != nil {
				return Library{}, refusedFile("machinefile", err)
			}
			name, overlay := catalog.MachinefileOf(path.Base(f.Path))
			placed := Placed{File: mf, Layer: t.Layer, Path: file, Body: f.Body}
			switch {
			case overlay && t.Layer != User:
				return Library{}, refusedFile("machinefile", &catalog.FileError{File: file, Problem: "an overlay is the owner's alone, in ~/.config/station/machinefiles"})
			case overlay:
				placed.Layer = Overlay
				lib.Overlays[name] = placed
			case lib.Machinefiles[name].Path == "":
				lib.Machinefiles[name] = placed
			}
		}
	}
	return lib, nil
}
