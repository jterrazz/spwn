package process

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/setup"
)

type sleeper struct {
	at      time.Time
	asked   time.Duration
	release chan struct{}
}

type manualClock struct {
	mu       sync.Mutex
	now      time.Time
	sleepers map[*sleeper]bool
}

func newManualClock() *manualClock {
	return &manualClock{now: time.Date(2026, 11, 3, 10, 30, 0, 0, time.UTC), sleepers: map[*sleeper]bool{}}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) Sleep(ctx context.Context, d time.Duration) error {
	c.mu.Lock()
	s := &sleeper{at: c.now.Add(d), asked: d, release: make(chan struct{})}
	c.sleepers[s] = true
	c.mu.Unlock()
	select {
	case <-s.release:
		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.sleepers, s)
		c.mu.Unlock()
		return ctx.Err()
	}
}

func (c *manualClock) await(d time.Duration) {
	for {
		c.mu.Lock()
		for s := range c.sleepers {
			if s.asked == d {
				c.mu.Unlock()
				return
			}
		}
		c.mu.Unlock()
		runtime.Gosched()
	}
}

func (c *manualClock) advance(d time.Duration) {
	c.await(d)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	for s := range c.sleepers {
		if !s.at.After(c.now) {
			delete(c.sleepers, s)
			close(s.release)
		}
	}
}

type streamed struct {
	events chan engine.Event
	ended  chan struct{}
}

// value: the nth subscription's feed reads the nth channel.
func streamEngine(t *testing.T, clock *manualClock, feeds ...chan any) *Engine {
	t.Helper()
	h := &fakeMac{clock: clock}
	paths := setup.Paths{Prefs: "/c", State: "/s"}
	var n atomic.Int32
	return New(h, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: paths.State, ThisMachine: catalog.Mac,
		Feeds: map[string]engine.Feed{engine.PageMenubar: func(ctx context.Context, _ engine.FeedInput) <-chan any {
			feed := feeds[n.Add(1)-1]
			out := make(chan any)
			go func() {
				defer close(out)
				for {
					select {
					case <-ctx.Done():
						return
					case p := <-feed:
						select {
						case out <- p:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
			return out
		}}})
}

// value: a watch of the menubar page; a subscriber not reading blocks each
// of its sends until its watch ends.
func subscribe(ctx context.Context, t *testing.T, d *Engine, reading bool) streamed {
	t.Helper()
	s := streamed{events: make(chan engine.Event, 64), ended: make(chan struct{})}
	opened := make(chan struct{})
	go func() {
		defer close(s.ended)
		defer close(s.events)
		_ = d.Stream(ctx, []string{engine.PageMenubar}, func(v any) error {
			if _, ok := v.(engine.Opened); ok {
				close(opened)
				return nil
			}
			if !reading {
				<-ctx.Done()
				return ctx.Err()
			}
			ev, _ := v.(engine.Event)
			s.events <- ev
			return nil
		})
	}()
	<-opened
	return s
}

func (s streamed) next(t *testing.T, kind string) string {
	t.Helper()
	for ev := range s.events {
		if ev.Event == kind {
			if kind == "page" {
				return string(ev.Page)
			}
			return ev.At
		}
	}
	t.Fatalf("the stream ended before a %s", kind)
	return ""
}

// value: the third send returns only once the feed's loop is back for more,
// so the first one is stored.
func settle(feed chan any, v any) {
	for range 3 {
		feed <- v
	}
}

func TestAStreamSendsTheFrameThenOnePageForABurstAndNeverTheSamePageTwice(t *testing.T) {
	clock, feed := newManualClock(), make(chan any)
	s := subscribe(t.Context(), t, streamEngine(t, clock, feed), true)
	feed <- "frame"
	if got := s.next(t, "page"); got != `"frame"` {
		t.Fatalf("first page %s, want the frame", got)
	}
	feed <- "a"
	feed <- "b"
	settle(feed, "c")
	clock.advance(coalesce)
	if got := s.next(t, "page"); got != `"c"` {
		t.Fatalf("after a burst %s, want only its last page", got)
	}
	settle(feed, "c")
	settle(feed, "d")
	clock.advance(coalesce)
	if got := s.next(t, "page"); got != `"d"` {
		t.Fatalf("next page %s, want d: an equal page is never resent", got)
	}
}

func TestAStreamTicksEvery15Seconds(t *testing.T) {
	clock, feed := newManualClock(), make(chan any)
	s := subscribe(t.Context(), t, streamEngine(t, clock, feed), true)
	feed <- "frame"
	s.next(t, "page")
	clock.advance(tickEvery)
	if got := s.next(t, "tick"); got != "2026-11-03T10:30:15Z" {
		t.Fatalf("tick at %s, want 15 s after the subscription", got)
	}
}

func TestASubscriberThatDoesNotReadNeverHoldsAnotherBack(t *testing.T) {
	clock, feedA, feedB := newManualClock(), make(chan any), make(chan any)
	d := streamEngine(t, clock, feedA, feedB)
	reader := subscribe(t.Context(), t, d, true)
	feedA <- "frame"
	reader.next(t, "page")
	subscribe(t.Context(), t, d, false)
	feedB <- "frame"
	settle(feedA, "after")
	clock.advance(coalesce)
	if got := reader.next(t, "page"); got != `"after"` {
		t.Fatalf("the reader got %s, want its page after the other was dropped", got)
	}
}

// invariant: MB-14: a machine's page is watched by its name after the
// page's (`menubar.machine:jterrazz-mac-mini`): the machine feed streams it
// for that machine; an unknown page is still refused.
func TestAMachinesPageIsWatchedByItsName(t *testing.T) {
	h := &fakeMac{clock: newManualClock()}
	subjects := make(chan string, 1)
	d := New(h, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac,
		Feeds: map[string]engine.Feed{engine.PageMachine: func(_ context.Context, in engine.FeedInput) <-chan any {
			subjects <- in.Subject
			out := make(chan any, 1)
			out <- map[string]string{"infra": "restarting"}
			close(out)
			return out
		}}})
	if err := d.CheckPages([]string{"menubar.machine:jterrazz-mac-mini"}); err != nil {
		t.Fatalf("the machine's page is refused: %v", err)
	}
	if err := d.CheckPages([]string{"menubar.nothing:jterrazz-mac-mini"}); err == nil {
		t.Fatal("an unknown page is watched")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var pages []string
	_ = d.Stream(ctx, []string{"menubar.machine:jterrazz-mac-mini"}, func(v any) error {
		if ev, ok := v.(engine.Event); ok && ev.Event == "page" {
			pages = append(pages, string(ev.Page))
			cancel()
		}
		return nil
	})
	if got := <-subjects; got != "jterrazz-mac-mini" {
		t.Fatalf("the feed was asked for %q", got)
	}
	if len(pages) != 1 || pages[0] != `{"infra":"restarting"}` {
		t.Fatalf("streamed %q, want the mini's page", pages)
	}
}

func TestARefreshReadsAgainOnlyThePageItNames(t *testing.T) {
	h := &fakeMac{clock: newManualClock()}
	woke := make(chan string, 8)
	reread := func(name string) engine.Feed {
		return func(ctx context.Context, in engine.FeedInput) <-chan any {
			out := make(chan any, 1)
			out <- map[string]string{"page": name}
			go func() {
				defer close(out)
				for {
					select {
					case <-in.Refresh:
						woke <- name
					case <-ctx.Done():
						select {
						case <-in.Refresh:
							woke <- name
						default:
						}
						return
					}
				}
			}()
			return out
		}
	}
	d := New(h, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac,
		Feeds: map[string]engine.Feed{engine.PageMenubar: reread("home"), engine.PageMachine: reread("linux-1")}})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	framed := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		n := 0
		_ = d.Stream(ctx, []string{engine.PageMenubar, "menubar.machine:linux-1"}, func(v any) error {
			if ev, ok := v.(engine.Event); ok && ev.Event == "page" {
				n++
				if n == 2 {
					close(framed)
				}
			}
			return nil
		})
	}()
	<-framed
	if err := d.Refresh([]string{"menubar.machine:linux-1"}); err != nil {
		t.Fatalf("linux-1's page is not refreshed: %v", err)
	}
	if got := <-woke; got != "linux-1" {
		t.Fatalf("a refresh of linux-1's page read %q again", got)
	}
	cancel()
	<-done
	close(woke)
	for got := range woke {
		t.Errorf("a refresh of linux-1's page also read %q again", got)
	}
}
