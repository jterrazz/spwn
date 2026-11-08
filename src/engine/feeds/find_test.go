package feeds_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/src/awake"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/engine/feeds"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/page"
	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
)

func opened(t *testing.T, w *world, feed func(feeds.Menubar) engine.Feed, ready func(page.Page) bool) page.Page {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	o := feeds.Menubar{Calls: w, Clock: newSteps(), ThisMachine: page.Context{Machine: thisMac, Kind: "mac"}, Report: func(awake.View) *awake.Report { return nil }}
	in := engine.FeedInput{Changed: make(chan struct{}), Refresh: make(chan struct{})}
	in.Awake = nothing(in.Awake)
	return until(t, feed(o)(ctx, in), "every Mac read", ready)
}

func askedEach(w *world) map[string][]string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := map[string][]string{}
	for name, on := range w.asked {
		seen := slices.Clone(on)
		slices.Sort(seen)
		out[name] = slices.Compact(seen)
	}
	return out
}

// invariant: SPEC B5.6: Find is built from home's own reading: opened, it
// asks no call, and no Mac, that home does not ask; it lists both Macs and
// infra on mac-mini from it.
func TestTheFindIndexCostsNoReadHomeDoesNotMake(t *testing.T) {
	home := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	opened(t, home, feeds.Menubar.Feed, func(p page.Page) bool { return row(p, "machines", mini) != nil && len(needs(p)) > 0 })

	find := &world{infra: []machines.VM{infraAt(machines.Running, 0)}, asked: map[string][]string{}, activity: map[string]uint64{}}
	p := opened(t, find, feeds.Menubar.Find, func(p page.Page) bool { return row(p, pagemenubar.FindMachines, mini) != nil })

	if p.ID != pagemenubar.FindID || p.Error != nil {
		t.Fatalf("the Find index is %s, %+v", p.ID, p.Error)
	}
	if row(p, pagemenubar.FindMachines, "infra") == nil {
		t.Errorf("the Find index lists no infra on mac-mini")
	}
	read, asked := askedEach(home), askedEach(find)
	if len(asked) == 0 {
		t.Fatal("the Find index was drawn from no read")
	}
	for _, name := range slices.Sorted(maps.Keys(asked)) {
		for _, on := range asked[name] {
			if !slices.Contains(read[name], on) {
				t.Errorf("Find asked %s of %s, which home never asks", name, on)
			}
		}
	}
}
