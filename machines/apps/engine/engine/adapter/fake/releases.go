package fake

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Releases struct{ w *world.World }

var (
	_ contract.Releases     = Releases{}
	_ contract.ReleaseCache = ReleaseCache{}
)

func ReleasesOf(w *world.World) Releases { return Releases{w: w} }

func (r Releases) Latest(context.Context) (engine.Tagged, error) {
	r.w.Lock()
	defer r.w.Unlock()
	if r.w.Fails("Releases.Latest") {
		return engine.Tagged{}, errors.New("GitHub: dial tcp: connection refused")
	}
	if len(r.w.Releases) == 0 {
		return engine.Tagged{}, nil
	}
	last := r.w.Releases[len(r.w.Releases)-1]
	return engine.Tagged{Tag: last.Tag, URL: last.URL, Assets: slices.Clone(last.Assets)}, nil
}

type ReleaseCache struct {
	// invariant: what one Mac of the world keeps of its asks.
	w   *world.World
	mac string
}

func ReleaseCacheOf(w *world.World, mac string) ReleaseCache { return ReleaseCache{w: w, mac: mac} }

func (c ReleaseCache) Read(context.Context) (engine.ReleaseKept, bool, error) {
	c.w.Lock()
	defer c.w.Unlock()
	kept := c.w.Mac(c.mac).Store().Release
	if kept == nil {
		return engine.ReleaseKept{}, false, nil
	}
	return *kept, true, nil
}

func (c ReleaseCache) Write(_ context.Context, k engine.ReleaseKept) error {
	c.w.Lock()
	defer c.w.Unlock()
	c.w.Mac(c.mac).Store().Release = &k
	return nil
}
