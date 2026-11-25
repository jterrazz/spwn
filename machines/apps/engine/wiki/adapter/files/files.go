package files

import (
	"context"
	"io/fs"
	"path/filepath"

	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

type Files interface {
	ReadFile(path string) ([]byte, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

type Corpus struct {
	// value: a corpus read from a folder of this machine's disk; an empty
	// root is a machine that declares none.
	files Files
	root  string
}

var _ contract.Corpus = (*Corpus)(nil)

func New(files Files, root string) *Corpus { return &Corpus{files: files, root: root} }

func (c *Corpus) Files(ctx context.Context) (map[string][]byte, error) {
	if c.root == "" {
		return nil, contract.ErrNone
	}
	out := map[string][]byte{}
	walked := c.files.WalkDir(c.root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if stopped := ctx.Err(); stopped != nil {
			return stopped
		}
		return c.take(out, p, d)
	})
	if walked != nil {
		return nil, walked
	}
	return out, nil
}

func (c *Corpus) take(out map[string][]byte, p string, d fs.DirEntry) error {
	rel, err := filepath.Rel(c.root, p)
	if err != nil {
		return err
	}
	slash := filepath.ToSlash(rel)
	if d.IsDir() {
		if slash != "." && !contract.Reads(slash+"/page.md") {
			return fs.SkipDir
		}
		return nil
	}
	if !contract.Reads(slash) {
		return nil
	}
	body, err := c.files.ReadFile(p)
	if err != nil {
		return err
	}
	out[slash] = body
	return nil
}
