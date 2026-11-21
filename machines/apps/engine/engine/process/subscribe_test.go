package process

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/engine"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
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

// value: the engine's readings as a stream sees them: a store the test
// puts into, and the gestures it counts.
type fakeReading struct {
	st    *store.Store
	ready atomic.Int32
	due   atomic.Int32
}

func newReading(clock *manualClock) *fakeReading {
	return &fakeReading{st: store.New(store.EpochOf(clock.Now(), 1), clock.Now(), func(err error) *fail.Error { return &fail.Error{Message: err.Error()} })}
}

func (r *fakeReading) Tick(context.Context, bool) {}
func (r *fakeReading) Moved()                     {}
func (r *fakeReading) DueAll()                    { r.due.Add(1) }
func (r *fakeReading) Remember(engine.Held)       {}
func (r *fakeReading) Held() engine.Held          { return engine.Held{} }
func (r *fakeReading) Store() *store.Store        { return r.st }
func (r *fakeReading) Ready(context.Context)      { r.ready.Add(1) }

var homeKey = store.Of(store.FamilyOverview, "this-mac")

func (r *fakeReading) put(t *testing.T, v any) {
	t.Helper()
	if err := r.st.Put(store.Put{Key: homeKey, Value: v}); err != nil {
		t.Fatal(err)
	}
}

// value: a page that says the value of This Mac's overview slice, or
// "unread"; projections counts the pages drawn.
type homePage struct{ projections atomic.Int32 }

func (h *homePage) project(s *store.Snapshot, _ string) any {
	h.projections.Add(1)
	if sl, ok := s.Slices[homeKey]; ok {
		return sl.Value
	}
	return "unread"
}

func streamEngine(t *testing.T, clock *manualClock, r *fakeReading, home *homePage) *Engine {
	t.Helper()
	h := &fakeMac{clock: clock}
	paths := setup.Paths{Prefs: "/c", State: "/s"}
	return New(h, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: paths.State, ThisMachine: catalog.Mac, Reading: r,
		Pages: map[string]Projection{engine.PageMenubar: home.project}})
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
		_ = d.Stream(ctx, engine.PageWatchIn{Pages: []string{engine.PageMenubar}}, func(v any) error {
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

func TestAWatchOpensOnWhatTheStoreHoldsOnceTheEngineIsReady(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	r := newReading(clock)
	r.put(t, "held")
	s := subscribe(t.Context(), t, streamEngine(t, clock, r, home), true)
	if got := s.next(t, "page"); got != `"held"` {
		t.Fatalf("first page %s, want what the store held", got)
	}
	if r.ready.Load() != 1 {
		t.Fatalf("the engine was made ready %d times before the first page, want once", r.ready.Load())
	}
}

func TestAStreamSendsOnePageForABurstAndNeverTheSamePageTwice(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	r := newReading(clock)
	s := subscribe(t.Context(), t, streamEngine(t, clock, r, home), true)
	if got := s.next(t, "page"); got != `"unread"` {
		t.Fatalf("first page %s, want the page of an empty store", got)
	}
	r.put(t, "a")
	r.put(t, "b")
	r.put(t, "c")
	clock.advance(coalesce)
	if got := s.next(t, "page"); got != `"c"` {
		t.Fatalf("after a burst %s, want only its last page", got)
	}
	if err := r.st.Put(store.Put{Key: store.Of(store.FamilyPorts, "this-mac"), Value: "other"}); err != nil {
		t.Fatal(err)
	}
	clock.advance(coalesce)
	clock.advance(tickEvery)
	if ev := <-s.events; ev.Event != "tick" {
		t.Fatalf("a revision that left the page as it was sent a %s, want nothing before the tick", ev.Event)
	}
	r.put(t, "d")
	clock.advance(coalesce)
	if got := s.next(t, "page"); got != `"d"` {
		t.Fatalf("next page %s, want d: an equal page is never resent", got)
	}
}

func TestAPageIsProjectedOncePerRevisionHoweverManyWatchIt(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	r := newReading(clock)
	r.put(t, "held")
	d := streamEngine(t, clock, r, home)
	a := subscribe(t.Context(), t, d, true)
	b := subscribe(t.Context(), t, d, true)
	a.next(t, "page")
	b.next(t, "page")
	if got := home.projections.Load(); got != 1 {
		t.Fatalf("two watches of one revision drew %d pages, want 1", got)
	}
}

// invariant: the page a watch sends last is the store's newest: revisions
// written at once never leave an older page standing after a newer one.
func TestRevisionsWrittenAtOnceLeaveTheNewestPageSentLast(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	r := newReading(clock)
	s := subscribe(t.Context(), t, streamEngine(t, clock, r, home), true)
	s.next(t, "page")
	var writers sync.WaitGroup
	for i := range 32 {
		writers.Go(func() { r.put(t, strconv.Itoa(i)) })
	}
	writers.Wait()
	clock.advance(coalesce)
	want := fmt.Sprintf("%q", r.st.Now().Slices[homeKey].Value)
	if got := s.next(t, "page"); got != want {
		t.Fatalf("the last page sent %s, want the newest revision's %s", got, want)
	}
}

func TestAStreamTicksEvery15Seconds(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	s := subscribe(t.Context(), t, streamEngine(t, clock, newReading(clock), home), true)
	s.next(t, "page")
	clock.advance(tickEvery)
	if got := s.next(t, "tick"); got != "2026-11-03T10:30:15Z" {
		t.Fatalf("tick at %s, want 15 s after the subscription", got)
	}
}

func TestASubscriberThatDoesNotReadNeverHoldsAnotherBack(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	r := newReading(clock)
	d := streamEngine(t, clock, r, home)
	reader := subscribe(t.Context(), t, d, true)
	reader.next(t, "page")
	subscribe(t.Context(), t, d, false)
	r.put(t, "after")
	clock.advance(coalesce)
	if got := reader.next(t, "page"); got != `"after"` {
		t.Fatalf("the reader got %s, want its page after the other was dropped", got)
	}
}

// invariant: MB-14: a machine's page is watched by its name after the
// page's (`menubar.machine:mac-mini`): the machine's projection draws it
// for that machine; an unknown page is still refused.
func TestAMachinesPageIsWatchedByItsName(t *testing.T) {
	clock := newManualClock()
	subjects := make(chan string, 1)
	d := New(&fakeMac{clock: clock}, newHypervisor(), nil, &fakeMachines{}, &fakeTailnet{}, memStores(), Options{State: "/s", ThisMachine: catalog.Mac, Reading: newReading(clock),
		Pages: map[string]Projection{engine.PageMachine: func(_ *store.Snapshot, subject string) any {
			subjects <- subject
			return map[string]string{"infra": "restarting"}
		}}})
	if err := d.CheckPages([]string{"menubar.machine:mac-mini"}); err != nil {
		t.Fatalf("the machine's page is refused: %v", err)
	}
	if err := d.CheckPages([]string{"menubar.nothing:mac-mini"}); err == nil {
		t.Fatal("an unknown page is watched")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var pages []string
	_ = d.Stream(ctx, engine.PageWatchIn{Pages: []string{"menubar.machine:mac-mini"}}, func(v any) error {
		if ev, ok := v.(engine.Event); ok && ev.Event == "page" {
			pages = append(pages, string(ev.Page))
			cancel()
		}
		return nil
	})
	if got := <-subjects; got != "mac-mini" {
		t.Fatalf("the page was drawn for %q", got)
	}
	if len(pages) != 1 || pages[0] != `{"infra":"restarting"}` {
		t.Fatalf("streamed %q, want the mini's page", pages)
	}
}

func TestARefreshMakesEveryReaderDue(t *testing.T) {
	clock := newManualClock()
	r := newReading(clock)
	d := streamEngine(t, clock, r, &homePage{})
	if err := d.Refresh([]string{engine.PageMenubar}); err != nil {
		t.Fatalf("home is not refreshed: %v", err)
	}
	if r.due.Load() != 1 {
		t.Fatalf("a refresh made the readers due %d times, want once", r.due.Load())
	}
}

func TestAWatchLogsItsOpeningOnceItsFirstFullPageWentOut(t *testing.T) {
	clock, home := newManualClock(), &homePage{}
	r := newReading(clock)
	r.put(t, map[string]any{"id": "menubar", "sections": []any{map[string]any{"id": "reading", "mark": "pending"}}})
	d := streamEngine(t, clock, r, home)
	notes := make(chan string, 4)
	d.note = func(line string) { notes <- line }
	var ran atomic.Int64
	d.o.Commands = ran.Load
	s := subscribe(t.Context(), t, d, true)
	s.next(t, "page")
	ran.Add(3)
	r.put(t, map[string]any{"id": "menubar", "header": map[string]any{"title": "This Mac"}})
	clock.advance(coalesce)
	s.next(t, "page")
	if got, want := <-notes, "watch: menubar first=0ms header=250ms full=250ms frames=2 commands=3"; !strings.HasSuffix(got, want) {
		t.Fatalf("logged %q, want it to end %q", got, want)
	}
}
