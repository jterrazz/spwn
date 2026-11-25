package fake

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Corpus struct {
	// value: the wiki one Mac of the world declares: its repository's files
	// at its head.
	w   *world.World
	mac string
}

var _ contract.Corpus = (*Corpus)(nil)

func Of(w *world.World, mac string) *Corpus { return &Corpus{w: w, mac: mac} }

func (c *Corpus) Files(context.Context) (map[string][]byte, error) {
	c.w.Lock()
	defer c.w.Unlock()
	repo := c.w.Mac(c.mac).Wiki
	if repo == nil {
		return nil, contract.ErrNone
	}
	out := map[string][]byte{}
	for p, body := range repo.Commits[repo.Head] {
		if contract.Reads(p) {
			out[p] = []byte(body)
		}
	}
	return out, nil
}
