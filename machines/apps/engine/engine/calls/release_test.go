package calls

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type githubStub struct {
	asks, inside, most atomic.Int32
	answer             func(ctx context.Context) (engine.Tagged, error)
}

func (g *githubStub) Latest(ctx context.Context) (engine.Tagged, error) {
	g.asks.Add(1)
	in := g.inside.Add(1)
	defer g.inside.Add(-1)
	for {
		m := g.most.Load()
		if in <= m || g.most.CompareAndSwap(m, in) {
			break
		}
	}
	for range 50 {
		runtime.Gosched()
	}
	return g.answer(ctx)
}

type cacheStub struct {
	mu    sync.Mutex
	kept  *engine.ReleaseKept
	wrote int
}

func (c *cacheStub) Read(context.Context) (engine.ReleaseKept, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kept == nil {
		return engine.ReleaseKept{}, false, nil
	}
	return *c.kept, true, nil
}

func (c *cacheStub) Write(_ context.Context, k engine.ReleaseKept) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.kept, c.wrote = &k, c.wrote+1
	return nil
}

func depsOf(g *githubStub, c *cacheStub) Deps {
	return Deps{Releases: g, Cache: c, Version: "v0.21.0", Arch: "arm64", Now: func() time.Time { return time.Unix(1000, 0) }, Asking: &sync.Mutex{}}
}

func TestTwoRacingAsksOfTheLatestReleaseRunOneAtATime(t *testing.T) {
	g := &githubStub{answer: func(context.Context) (engine.Tagged, error) {
		return engine.Tagged{Tag: "v0.22.0", URL: "https://github.com/jterrazz/spwn/releases/tag/v0.22.0", Assets: []string{"station-darwin-arm64"}}, nil
	}}
	c := &cacheStub{}
	d := depsOf(g, c)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := RefreshRelease.Run(context.Background(), d, ReleaseIn{}, nil); err != nil {
				t.Errorf("refresh: %v", err)
			}
		}()
	}
	wg.Wait()
	if got := g.most.Load(); got != 1 {
		t.Errorf("%d asks of GitHub ran at once; want 1", got)
	}
	if g.asks.Load() != 8 || c.wrote != 8 {
		t.Errorf("%d asks, %d writes; want 8 and 8", g.asks.Load(), c.wrote)
	}
}

func TestAnAskCutShortByItsContextIsNotKeptAsARefusal(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	g := &githubStub{answer: func(ctx context.Context) (engine.Tagged, error) {
		stop()
		return engine.Tagged{}, errors.New("GitHub: " + ctx.Err().Error())
	}}
	read := time.Unix(50, 0)
	c := &cacheStub{kept: &engine.ReleaseKept{Read: read}}
	if _, err := RefreshRelease.Run(ctx, depsOf(g, c), ReleaseIn{}, nil); err == nil {
		t.Fatal("refresh of a cancelled context answered")
	}
	if c.wrote != 0 || c.kept.Refused != nil {
		t.Errorf("cache written %d times, kept %+v; want what was kept left as it was", c.wrote, c.kept)
	}
}
