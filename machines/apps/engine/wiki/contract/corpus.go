package contract

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
)

type Corpus interface {
	// value: the wiki this machine declares, read whole: its wiki.yaml and
	// every page, each by its slash path under the corpus's root.
	// invariant: ErrNone when the machine declares no wiki; a file that is
	// neither wiki.yaml at the root nor a .md page is never read, a page past
	// wiki.MaxPageBytes is listed by its size and never read, and a corpus
	// past wiki.MaxPages or wiki.MaxCorpusBytes stops the read with a
	// *wiki.TooLarge.
	Files(ctx context.Context) (map[string]wiki.File, error)
	// value: what changes when any file Files would read changes, read
	// without reading one: each file's path, size and time, or its bytes'
	// hash where there is no time.
	Signature(ctx context.Context) (string, error)
}

var ErrNone = errors.New("this machine declares no wiki")

type Unreadable struct {
	// value: a file or folder of the corpus its read refused, by its slash
	// path under the corpus's root: a link out of the folder, a FIFO, a
	// permission. Only the path inside the corpus is named.
	Path string
	Err  error
}

func (e *Unreadable) Error() string { return e.Path + " cannot be read" }

func (e *Unreadable) Unwrap() error { return e.Err }

func Reads(slashPath string) bool {
	// value: a file of a corpus's tree the wiki reads: its wiki.yaml at the
	// root, and every Markdown page outside a hidden folder that is not
	// hidden itself.
	for _, part := range strings.Split(path.Dir(slashPath), "/") {
		if strings.HasPrefix(part, ".") && part != "." {
			return false
		}
	}
	if strings.HasPrefix(path.Base(slashPath), ".") {
		return false
	}
	return slashPath == "wiki.yaml" || strings.HasSuffix(slashPath, ".md")
}
