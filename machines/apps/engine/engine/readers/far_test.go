package readers_test

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/readers"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

// value: a clock whose sleeps end only when the test moves it, and which
// counts the sleeps begun so a test moves it once its sleeper is asleep.
type turned struct {
	mu    sync.Mutex
	now   time.Time
	moved chan struct{}
	begun int
}

func (c *turned) sleeps() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.begun
}

func (c *turned) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *turned) advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	close(c.moved)
	c.moved = make(chan struct{})
	c.mu.Unlock()
}

func (c *turned) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	until := c.now.Add(d)
	c.begun++
	c.mu.Unlock()
	for {
		c.mu.Lock()
		now, moved := c.now, c.moved
		c.mu.Unlock()
		if !now.Before(until) {
			return nil
		}
		select {
		case <-moved:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// value: another Mac's engine.state as the Router reaches it: each watch
// counted, answered by the test's frames, or refused while down.
type farMac struct {
	mu      sync.Mutex
	watches int
	down    bool
	frames  chan call.Update[any]
}

func (f *farMac) Invoke(context.Context, machine.Ref, string, any, any, call.Progress) error {
	return errors.New("not asked")
}

func (f *farMac) Watch(_ context.Context, _ machine.Ref, _ string, _ any, _ call.Progress) (<-chan call.Update[any], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.watches++
	if f.down {
		return nil, &fail.Error{Type: fail.Unreachable, Message: "mac-mini does not answer"}
	}
	f.frames = make(chan call.Update[any], 4)
	return f.frames, nil
}

func (f *farMac) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.watches
}

func (f *farMac) send(t *testing.T, fr engine.StateFrame) {
	t.Helper()
	raw, err := json.Marshal(fr)
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	frames := f.frames
	f.mu.Unlock()
	frames <- call.Update[any]{Value: json.RawMessage(raw)}
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for range 1_000_000 {
		if ok() {
			return
		}
		runtime.Gosched()
	}
	t.Fatalf("never: %s", what)
}

func TestASilentMacGoesStaleOnceAndIsDialledAgainOnTheBackoff(t *testing.T) {
	c := &turned{now: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC), moved: make(chan struct{})}
	st := store.New(store.EpochOf(c.now, 1), c.now, word)
	mini := &farMac{}
	var dropped []string
	var mu sync.Mutex
	far := &readers.Far{Calls: mini, Macs: func() []string { return []string{"mac-mini"} }, Drop: func(mac string) {
		mu.Lock()
		defer mu.Unlock()
		dropped = append(dropped, mac)
	}}
	set := readers.New(readers.Options{Store: st, Clock: c, Run: func(f func()) { f() }, Far: far})
	ctx := t.Context()
	key := store.Of(store.FamilyMac, "mac-mini")

	set.Tick(ctx, false)
	fit := engine.FitOf(engine.StateRevision, engine.StateRevision)
	mini.send(t, engine.StateFrame{Event: engine.StateOpened, Epoch: "e/1", Fit: &fit})
	mini.send(t, engine.StateFrame{Event: engine.StateSnapshot, Rev: 3, Slices: map[string]engine.SliceOut{
		string(key): {Value: json.RawMessage(`{"Name":"mac-mini"}`), State: string(store.Fresh)},
	}})
	eventually(t, "the mini's slice is copied fresh", func() bool { return st.Now().Slices[key].State == store.Fresh })

	mini.mu.Lock()
	mini.down = true
	mini.mu.Unlock()
	eventually(t, "the watchdog is asleep", func() bool { return c.sleeps() == 1 })
	c.advance(45 * time.Second)
	eventually(t, "45 s with no frame marks the mini stale", func() bool { return st.Now().Slices[key].State == store.Stale })
	eventually(t, "the silent mini's link is dropped", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(dropped) > 0
	})
	got := st.Now().Slices[key]
	if got.Why != store.WhyUnreachable || got.Value == nil {
		t.Fatalf("a silent mini's slice: why %q, value %v; want unreachable with its last value", got.Why, got.Value)
	}
	mu.Lock()
	if !slices.Equal(dropped, []string{"mac-mini"}) {
		t.Errorf("links dropped: %v, want the mini's once", dropped)
	}
	mu.Unlock()

	for _, step := range []struct {
		after time.Duration
		dials int
	}{{4 * time.Second, 1}, {time.Second, 2}, {9 * time.Second, 2}, {time.Second, 3}, {29 * time.Second, 3}, {time.Second, 4}} {
		c.advance(step.after)
		set.Tick(ctx, false)
		if n := mini.count(); n != step.dials {
			t.Fatalf("at %s: %d watches, want %d (backoff 5 s, 10 s, 30 s)", c.Now().Sub(time.Date(2026, 11, 21, 9, 0, 45, 0, time.UTC)), n, step.dials)
		}
	}
}

func TestARememberedMacComesBackOnlyWhenItDoesNotAnswer(t *testing.T) {
	key := store.Of(store.FamilyMac, "mac-mini")
	held := engine.Held{"mac-mini": {string(key): {Value: json.RawMessage(`{"Name":"remembered"}`), State: string(store.Fresh), ReadAt: "2026-11-21T08:59:00Z"}}}
	open := func(down bool) (*store.Store, *farMac) {
		c := &turned{now: time.Date(2026, 11, 21, 9, 0, 0, 0, time.UTC), moved: make(chan struct{})}
		st := store.New(store.EpochOf(c.now, 1), c.now, word)
		mini := &farMac{down: down}
		far := &readers.Far{Calls: mini, Macs: func() []string { return []string{"mac-mini"} }}
		set := readers.New(readers.Options{Store: st, Clock: c, Run: func(f func()) { f() }, Far: far})
		set.Remember(held)
		set.Tick(t.Context(), false)
		return st, mini
	}

	st, mini := open(false)
	fit := engine.FitOf(engine.StateRevision, engine.StateRevision)
	mini.send(t, engine.StateFrame{Event: engine.StateOpened, Epoch: "e/1", Fit: &fit})
	mini.send(t, engine.StateFrame{Event: engine.StateSnapshot, Rev: 3, Slices: map[string]engine.SliceOut{
		string(key): {Value: json.RawMessage(`{"Name":"answered"}`), State: string(store.Fresh)},
	}})
	eventually(t, "the mini's answer is copied", func() bool { return st.Now().Slices[key].State == store.Fresh })
	if got := st.Now(); got.Rev != 1 || !strings.Contains(string(got.Slices[key].Raw), "answered") {
		t.Fatalf("a mini that answers: rev %d, slice %s; want its answer alone, at rev 1 as with no memory", got.Rev, got.Slices[key].Raw)
	}

	st, _ = open(true)
	got := st.Now().Slices[key]
	if got.State != store.Stale || got.Why != store.WhyUnreachable || !strings.Contains(string(got.Raw), "remembered") {
		t.Fatalf("a mini that does not answer: %s %q %s; want its remembered slice, stale as unreachable", got.State, got.Why, got.Raw)
	}
}
