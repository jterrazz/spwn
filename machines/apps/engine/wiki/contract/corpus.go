package contract

import (
	"context"
	"errors"
	"path"
	"strings"
)

type Corpus interface {
	// value: the wiki this machine declares, read whole: its wiki.yaml and
	// every page, each by its slash path under the corpus's root.
	// invariant: ErrNone when the machine declares no wiki; a file that is
	// neither wiki.yaml at the root nor a .md page is never read.
	Files(ctx context.Context) (map[string][]byte, error)
}

var ErrNone = errors.New("this machine declares no wiki")

func Reads(slashPath string) bool {
	// value: a file of a corpus's tree the wiki reads: its wiki.yaml at the
	// root, and every Markdown page outside a hidden folder.
	for _, part := range strings.Split(path.Dir(slashPath), "/") {
		if strings.HasPrefix(part, ".") && part != "." {
			return false
		}
	}
	return slashPath == "wiki.yaml" || strings.HasSuffix(slashPath, ".md")
}
