package files

import (
	"context"
	"io/fs"
	"path/filepath"

	"spwn.sh/packages/platform"

	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

type Corpus struct {
	// value: a corpus read from a folder of this machine's disk; an empty
	// root is a machine that declares none.
	root string
}

var _ contract.Corpus = (*Corpus)(nil)

func New(root string) *Corpus { return &Corpus{root: root} }

func (c *Corpus) Files(ctx context.Context) (map[string][]byte, error) {
	if c.root == "" {
		return nil, contract.ErrNone
	}
	// invariant: every read of the corpus goes through one platform.Tree of
	// its root, so a page that is a link out of the folder, a ".." or a FIFO
	// is refused before anything outside the corpus is opened or waited on.
	tree := platform.Within(c.root)
	defer func() { _ = tree.Close() }()
	out := map[string][]byte{}
	walked := tree.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if stopped := ctx.Err(); stopped != nil {
			return stopped
		}
		return take(tree, out, p, d)
	})
	if walked != nil {
		return nil, walked
	}
	return out, nil
}

func take(tree *platform.Tree, out map[string][]byte, p string, d fs.DirEntry) error {
	slash := filepath.ToSlash(p)
	if d.IsDir() {
		if slash != "." && !contract.Reads(slash+"/page.md") {
			return fs.SkipDir
		}
		return nil
	}
	if !contract.Reads(slash) {
		return nil
	}
	body, err := tree.ReadFile(p)
	if err != nil {
		return err
	}
	out[slash] = body
	return nil
}
