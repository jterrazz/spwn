package process

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/engine"
	"github.com/jterrazz/studio-station/src/setup"
)

const here = "jterrazz-macbook-pro"

type ticks struct {
	mu    sync.Mutex
	asked [][]string
}

func (s *ticks) sample(_ context.Context, keys []string) engine.Sample {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asked = append(s.asked, keys)
	values := map[string]any{}
	for i, k := range keys {
		values[k] = 10*len(s.asked) + i
	}
	return engine.Sample{Values: values, Stale: []string{"cpu:linux-1"}}
}

// value: an engine whose home lists This Mac's CPU and network, and whose
// machine pages list their machine's CPU, each sent once, as its frame.
func liveEngine(clock *manualClock, s *ticks) *Engine {
	listing := func(ctx context.Context, keys ...string) <-chan any {
		out := make(chan any, 1)
		out <- json.RawMessage(`{"live":` + string(mustJSON(keys)) + `}`)
		go func() {
			<-ctx.Done()
			close(out)
		}()
		return out
	}
	paths := setup.Paths{Prefs: "/c", State: "/s"}
	return New(&fakeMac{clock: clock}, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: paths.State, ThisMachine: catalog.Mac, Sample: s.sample,
		Feeds: map[string]engine.Feed{
			engine.PageMenubar: func(ctx context.Context, _ engine.FeedInput) <-chan any {
				return listing(ctx, "cpu:"+here, "net:"+here)
			},
			engine.PageMachine: func(ctx context.Context, in engine.FeedInput) <-chan any { return listing(ctx, "cpu:"+in.Subject) },
		}})
}

func watchPage(ctx context.Context, d *Engine, id string) <-chan engine.Event {
	events := make(chan engine.Event, 16)
	go func() {
		defer close(events)
		_ = d.Stream(ctx, []string{id}, func(v any) error {
			if ev, ok := v.(engine.Event); ok {
				events <- ev
			}
			return nil
		})
	}()
	return events
}

func TestOneTickCarriesHomesTheMacPagesAndTheVMPagesKeysAtTheSameValueAndNoPage(t *testing.T) {
	clock, s := newManualClock(), &ticks{}
	d := liveEngine(clock, s)
	home, mac, vm := watchPage(t.Context(), d, engine.PageMenubar), watchPage(t.Context(), d, engine.PageMachine+":"+here), watchPage(t.Context(), d, engine.PageMachine+":linux-1")
	for _, w := range []<-chan engine.Event{home, mac, vm} {
		if ev := <-w; ev.Event != "page" {
			t.Fatalf("a watch opened on a %s, want its page's frame", ev.Event)
		}
	}
	clock.advance(engine.SampleEvery)
	got := map[string]engine.Event{}
	for name, w := range map[string]<-chan engine.Event{"home": home, "the Mac page": mac, "the VM page": vm} {
		got[name] = <-w
		if got[name].Event != engine.EventSample {
			t.Errorf("%s's watch sent a %s on the tick, want the sample alone: no page is resent unchanged", name, got[name].Event)
		}
	}
	if want := [][]string{{"cpu:" + here, "cpu:linux-1", "net:" + here}}; !slices.EqualFunc(s.asked, want, slices.Equal) {
		t.Errorf("the tick read %q, want every open page's keys read once, together", s.asked)
	}
	for name, want := range map[string][]string{"home": {"cpu:" + here, "net:" + here}, "the Mac page": {"cpu:" + here}, "the VM page": {"cpu:linux-1"}} {
		if keys := slices.Sorted(maps.Keys(got[name].Values)); !slices.Equal(keys, want) {
			t.Errorf("%s's sample holds %q, want its own page's keys %q", name, keys, want)
		}
	}
	if a, b := got["home"].Values["cpu:"+here], got["the Mac page"].Values["cpu:"+here]; a != b || got["home"].At != got["the Mac page"].At {
		t.Errorf("home shows This Mac's CPU %v and the Mac page %v, want one value in one tick", a, b)
	}
	if !slices.Equal(got["the VM page"].Stale, []string{"cpu:linux-1"}) || len(got["home"].Stale) != 0 {
		t.Errorf("the stale keys reached home %q and the VM page %q, want each watch told of its own keys alone", got["home"].Stale, got["the VM page"].Stale)
	}
}

func TestNothingIsSampledOnceTheLastWatchEnds(t *testing.T) {
	clock, s := newManualClock(), &ticks{}
	d := liveEngine(clock, s)
	ctx, cancel := context.WithCancel(t.Context())
	w := watchPage(ctx, d, engine.PageMenubar)
	<-w
	clock.await(engine.SampleEvery)
	cancel()
	for range w {
	}
	d.hub.mu.Lock()
	running := d.hub.live.n
	d.hub.mu.Unlock()
	if running != 0 {
		t.Errorf("%d samplers still run with no watch open, want none", running)
	}
}
