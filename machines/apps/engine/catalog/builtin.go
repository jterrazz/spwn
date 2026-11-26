package catalog

import (
	"bytes"
	"io/fs"
	"path"
	"sync/atomic"
)

// value: the files this process decoded strictly, package files and
// Machinefiles alike: what a run's start is held to.
var decodes atomic.Int64

func Decoded() int64 { return decodes.Load() }

func ref[T any](v T) *T { return &v }

// value: a reader of package files that takes a built-in file's package
// from the compiled table when its bytes are the embedded file's, and
// decodes every other file: layering the owner's packages over the
// built-in ones decodes only the owner's.

type Reader struct {
	known map[string]Package
	built bool
}

func (r *Reader) Read(file, folder, origin string, body []byte) (Placed, error) {
	if p, ok := r.builtin(folder, path.Base(file), body); ok {
		return Placed{Package: p, Folder: folder, File: file, Origin: origin}, nil
	}
	return Read(file, folder, origin, body)
}

func (r *Reader) builtin(folder, name string, body []byte) (Package, bool) {
	if !r.built {
		r.built = true
		if c, err := Builtin(); err == nil {
			r.known = make(map[string]Package, len(c.Packages))
			for _, p := range c.Packages {
				r.known[p.File] = p.Package
			}
		}
	}
	at := path.Join("packages", folder, name)
	p, ok := r.known[at]
	if !ok {
		return Package{}, false
	}
	if embedded, err := fs.ReadFile(builtin, at); err != nil || !bytes.Equal(embedded, body) {
		return Package{}, false
	}
	return p, true
}
