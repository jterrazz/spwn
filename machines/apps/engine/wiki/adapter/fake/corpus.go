package fake

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/wiki"
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

func (c *Corpus) Signature(context.Context) (string, error) {
	c.w.Lock()
	defer c.w.Unlock()
	repo := c.w.Mac(c.mac).Wiki
	if repo == nil {
		return "", contract.ErrNone
	}
	h := sha256.New()
	for _, p := range slices.Sorted(maps.Keys(repo.Commits[repo.Head])) {
		if body := repo.Commits[repo.Head][p]; contract.Reads(p) {
			_, _ = h.Write([]byte(p + " " + strconv.Itoa(len(body)) + "\n" + body + "\n"))
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (c *Corpus) Files(context.Context) (map[string]wiki.File, error) {
	c.w.Lock()
	defer c.w.Unlock()
	repo := c.w.Mac(c.mac).Wiki
	if repo == nil {
		return nil, contract.ErrNone
	}
	out := map[string]wiki.File{}
	pages, bytes := 0, int64(0)
	for _, p := range slices.Sorted(maps.Keys(repo.Commits[repo.Head])) {
		if !contract.Reads(p) {
			continue
		}
		body := repo.Commits[repo.Head][p]
		f := wiki.File{Size: int64(len(body))}
		if f.Size <= wiki.MaxPageBytes {
			f.Body = []byte(body)
		}
		if p != "wiki.yaml" {
			pages++
		}
		bytes += int64(len(f.Body))
		if wiki.Past(pages, bytes) {
			return nil, &wiki.TooLarge{Pages: pages, Bytes: bytes}
		}
		out[p] = f
	}
	return out, nil
}
