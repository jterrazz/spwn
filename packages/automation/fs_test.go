package automation

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// fsFixture bundles the FakeClock + FSWatcher tests almost always
// need. Tests dispatch events directly via watcher.handle so the
// pump goroutine isn't in the timing path. One integration test
// (TestFSWatcher_PumpDeliversFromSource) covers the source→pump
// edge separately.
type fsFixture struct {
	source  *FakeFSSource
	clock   *FakeClock
	watcher *FSWatcher
	ctx     context.Context
}

func newFSFixture(t *testing.T) *fsFixture {
	t.Helper()
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	watcher := NewFSWatcher(source, clock)
	t.Cleanup(func() {
		_ = watcher.Close()
	})
	return &fsFixture{source: source, clock: clock, watcher: watcher, ctx: context.Background()}
}

// emit dispatches a synthetic event synchronously through the
// watcher's filter chain — no goroutine in the timing path. Use
// this in any test asserting "burst → 1 fire" semantics.
func (f *fsFixture) emit(path, kind string) {
	f.watcher.handle(f.ctx, RawFSEvent{Path: path, Kind: kind})
}

// captureHandler returns a handler closure that pushes events into a
// thread-safe slice plus a getter for the captured events.
func captureHandler() (FSHandler, func() []DebouncedFSEvent) {
	var mu sync.Mutex
	var got []DebouncedFSEvent
	handler := func(ev DebouncedFSEvent) {
		mu.Lock()
		got = append(got, ev)
		mu.Unlock()
	}
	getter := func() []DebouncedFSEvent {
		mu.Lock()
		defer mu.Unlock()
		out := make([]DebouncedFSEvent, len(got))
		copy(out, got)
		return out
	}
	return handler, getter
}

// waitForEvents polls until the captured slice has at least n entries
// or the deadline elapses. Used after Advance so the debounce
// goroutine has time to call the handler.
func waitForEvents(t *testing.T, getter func() []DebouncedFSEvent, n int, within time.Duration) {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if len(getter()) >= n {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("expected %d events within %s; got %d", n, within, len(getter()))
}

// ── Single create event ─────────────────────────────────────────────

func TestFSWatcher_FiresOnCreate(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "brain/inbox",
		Path:     "/inbox",
		Events:   []string{"create"},
		Debounce: 100 * time.Millisecond,
	}, handler))

	f.emit("/inbox/foo.md", "create")
	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)

	got := getter()[0]
	if got.SpecID != "brain/inbox" {
		t.Errorf("SpecID = %q", got.SpecID)
	}
	if len(got.Paths) != 1 || got.Paths[0] != "/inbox/foo.md" {
		t.Errorf("Paths = %v", got.Paths)
	}
	if got.Kind != "create" {
		t.Errorf("Kind = %q", got.Kind)
	}
}

// ── Debounce coalesces bursts ───────────────────────────────────────

func TestFSWatcher_BurstCoalescedIntoSingleFire(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "brain/inbox",
		Path:     "/inbox",
		Debounce: 1 * time.Second,
	}, handler))

	for _, name := range []string{"a.md", "b.md", "c.md", "d.md", "e.md"} {
		f.emit("/inbox/"+name, "create")
	}

	// Less than debounce — no fire yet.
	f.clock.Advance(500 * time.Millisecond)
	if got := len(getter()); got != 0 {
		t.Errorf("fired early: got %d events", got)
	}

	// Cross debounce — single fire with all 5 paths.
	f.clock.Advance(600 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)

	if got := len(getter()); got != 1 {
		t.Fatalf("got %d events, want 1 (debounce should coalesce)", got)
	}
	ev := getter()[0]
	if len(ev.Paths) != 5 {
		t.Errorf("Paths = %v, want 5 entries", ev.Paths)
	}
	// Paths must arrive sorted for stable receipts.
	for i := 1; i < len(ev.Paths); i++ {
		if ev.Paths[i-1] > ev.Paths[i] {
			t.Errorf("Paths not sorted: %v", ev.Paths)
			break
		}
	}
}

func TestFSWatcher_BurstResetsTimer(t *testing.T) {
	// Each new event during the debounce window resets the timer —
	// the burst doesn't fire until quiet for spec.Debounce.
	f := newFSFixture(t)
	handler, getter := captureHandler()
	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "x",
		Path:     "/x",
		Debounce: 1 * time.Second,
	}, handler))

	f.emit("/x/a.md", "create")
	f.clock.Advance(800 * time.Millisecond)
	// 800ms in — still pending.

	f.emit("/x/b.md", "create")
	// Burst extended. Advance another 800ms — total 1.6s but only
	// 800ms since the last event, so still pending.
	f.clock.Advance(800 * time.Millisecond)
	if got := len(getter()); got != 0 {
		t.Errorf("fired during reset: got %d", got)
	}

	// Advance past the second event's debounce.
	f.clock.Advance(300 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)
	if ev := getter()[0]; len(ev.Paths) != 2 {
		t.Errorf("expected both paths in single fire, got %d", len(ev.Paths))
	}
}

// ── Pattern filter ──────────────────────────────────────────────────

func TestFSWatcher_PatternFilter(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "x",
		Path:     "/x",
		Patterns: []string{"*.md"},
		Debounce: 50 * time.Millisecond,
	}, handler))

	f.emit("/x/keep.md", "create")
	f.emit("/x/drop.txt", "create")
	f.emit("/x/also-keep.md", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)

	ev := getter()[0]
	if len(ev.Paths) != 2 {
		t.Errorf("Paths = %v, want only the .md files", ev.Paths)
	}
	for _, p := range ev.Paths {
		if !endsWith(p, ".md") {
			t.Errorf("non-.md path slipped through: %s", p)
		}
	}
}

func TestFSWatcher_MultiplePatterns(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "x",
		Path:     "/x",
		Patterns: []string{"*.md", "*.txt"},
		Debounce: 50 * time.Millisecond,
	}, handler))

	f.emit("/x/a.md", "create")
	f.emit("/x/b.txt", "create")
	f.emit("/x/c.bin", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)
	if got := len(getter()[0].Paths); got != 2 {
		t.Errorf("Paths len = %d, want 2 (md+txt)", got)
	}
}

// ── Event filter ────────────────────────────────────────────────────

func TestFSWatcher_EventsFilter(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "x",
		Path:     "/x",
		Events:   []string{"create"},
		Debounce: 50 * time.Millisecond,
	}, handler))

	f.emit("/x/a.md", "create")
	f.emit("/x/b.md", "write")  // dropped
	f.emit("/x/c.md", "rename") // dropped

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)
	ev := getter()[0]
	if len(ev.Paths) != 1 || ev.Paths[0] != "/x/a.md" {
		t.Errorf("Paths = %v, want only the create", ev.Paths)
	}
}

func TestFSWatcher_AllEventsAllowed(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:       "x",
		Path:     "/x",
		Events:   []string{"create", "write", "rename"},
		Debounce: 50 * time.Millisecond,
	}, handler))

	f.emit("/x/a.md", "create")
	f.emit("/x/b.md", "write")
	f.emit("/x/c.md", "rename")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)
	if got := len(getter()[0].Paths); got != 3 {
		t.Errorf("Paths = %d, want 3", got)
	}
}

// ── Recursive ───────────────────────────────────────────────────────

func TestFSWatcher_RecursiveCoversSubdirs(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:        "x",
		Path:      "/x",
		Recursive: true,
		Debounce:  50 * time.Millisecond,
	}, handler))

	// Emit events at multiple depths.
	f.emit("/x/top.md", "create")
	f.emit("/x/sub/nested.md", "create")
	f.emit("/x/sub/deep/very-nested.md", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)
	if got := len(getter()[0].Paths); got != 3 {
		t.Errorf("Paths = %d, want 3 (recursive should pick all)", got)
	}
}

func TestFSWatcher_NonRecursiveRejectsSubdirEvents(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:        "x",
		Path:      "/x",
		Recursive: false,
		Debounce:  50 * time.Millisecond,
	}, handler))

	f.emit("/x/top.md", "create")
	f.emit("/x/sub/nested.md", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)
	ev := getter()[0]
	if len(ev.Paths) != 1 || ev.Paths[0] != "/x/top.md" {
		t.Errorf("Paths = %v, want only /x/top.md", ev.Paths)
	}
}

// ── Hidden-dir filtering ────────────────────────────────────────────

func TestFSWatcher_RecursiveExcludesHiddenDirsByDefault(t *testing.T) {
	// `.git/foo.md` and `sub/.cache/bar.md` should be filtered out
	// of a recursive watch unless IncludeHidden is set. Most users
	// don't want their inbox watcher fired by every git commit.
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:        "x",
		Path:      "/x",
		Recursive: true,
		Debounce:  50 * time.Millisecond,
	}, handler))

	f.emit("/x/top.md", "create")          // included
	f.emit("/x/.git/objects/abc", "create") // excluded
	f.emit("/x/sub/.cache/foo.md", "create") // excluded (deep hidden)
	f.emit("/x/sub/visible.md", "create")    // included

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)

	ev := getter()[0]
	if len(ev.Paths) != 2 {
		t.Errorf("Paths = %v, want 2 visible files (hidden dirs filtered)", ev.Paths)
	}
	for _, p := range ev.Paths {
		if strings.Contains(p, "/.") {
			t.Errorf("hidden path slipped through: %s", p)
		}
	}
}

func TestFSWatcher_IncludeHiddenAllowsDotDirs(t *testing.T) {
	f := newFSFixture(t)
	handler, getter := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID:            "x",
		Path:          "/x",
		Recursive:     true,
		IncludeHidden: true,
		Debounce:      50 * time.Millisecond,
	}, handler))

	f.emit("/x/top.md", "create")
	f.emit("/x/.git/objects/abc", "create")
	f.emit("/x/sub/.cache/foo.md", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 200*time.Millisecond)

	ev := getter()[0]
	if len(ev.Paths) != 3 {
		t.Errorf("IncludeHidden=true should pick up all 3, got %d", len(ev.Paths))
	}
}

// ── Spec isolation ──────────────────────────────────────────────────

func TestFSWatcher_TwoSpecsDifferentPathsIsolated(t *testing.T) {
	f := newFSFixture(t)
	hA, getA := captureHandler()
	hB, getB := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID: "a", Path: "/a", Debounce: 50 * time.Millisecond,
	}, hA))
	must(t, f.watcher.Watch(FSWatchSpec{
		ID: "b", Path: "/b", Debounce: 50 * time.Millisecond,
	}, hB))

	f.emit("/a/foo.md", "create")
	f.emit("/b/bar.md", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getA, 1, 200*time.Millisecond)
	waitForEvents(t, getB, 1, 200*time.Millisecond)

	if got := getA()[0].Paths[0]; got != "/a/foo.md" {
		t.Errorf("A path = %q", got)
	}
	if got := getB()[0].Paths[0]; got != "/b/bar.md" {
		t.Errorf("B path = %q", got)
	}
}

func TestFSWatcher_TwoSpecsSamePathDifferentPatternsBothFire(t *testing.T) {
	// The same /shared dir watched twice with different filters —
	// both handlers see the events that match their own pattern.
	f := newFSFixture(t)
	hMD, getMD := captureHandler()
	hAll, getAll := captureHandler()

	must(t, f.watcher.Watch(FSWatchSpec{
		ID: "md-only", Path: "/shared", Patterns: []string{"*.md"}, Debounce: 50 * time.Millisecond,
	}, hMD))
	must(t, f.watcher.Watch(FSWatchSpec{
		ID: "all", Path: "/shared", Debounce: 50 * time.Millisecond,
	}, hAll))

	f.emit("/shared/a.md", "create")
	f.emit("/shared/b.txt", "create")

	f.clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getMD, 1, 200*time.Millisecond)
	waitForEvents(t, getAll, 1, 200*time.Millisecond)

	if got := len(getMD()[0].Paths); got != 1 {
		t.Errorf("md-only got %d paths, want 1", got)
	}
	if got := len(getAll()[0].Paths); got != 2 {
		t.Errorf("all got %d paths, want 2", got)
	}
}

// ── Source registration ─────────────────────────────────────────────

func TestFSWatcher_PassesRecursiveFlagToSource(t *testing.T) {
	f := newFSFixture(t)
	handler, _ := captureHandler()
	must(t, f.watcher.Watch(FSWatchSpec{
		ID: "x", Path: "/x", Recursive: true, Debounce: 50 * time.Millisecond,
	}, handler))

	calls := f.source.AddCalls()
	if len(calls) != 1 {
		t.Fatalf("Add calls = %d, want 1", len(calls))
	}
	if calls[0].Path != "/x" || !calls[0].Recursive {
		t.Errorf("Add call = %+v", calls[0])
	}
}

// ── Pump integration ────────────────────────────────────────────────

func TestFSWatcher_PumpDeliversFromSource(t *testing.T) {
	// One end-to-end test that runs through the channel + pump
	// goroutine. Confirms the lifecycle is wired correctly: source
	// emits → pump dequeues → handle filters → debounce fires.
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	w := NewFSWatcher(source, clock)
	t.Cleanup(func() { _ = w.Close() })

	handler, getter := captureHandler()
	must(t, w.Watch(FSWatchSpec{
		ID: "x", Path: "/x", Debounce: 50 * time.Millisecond,
	}, handler))
	must(t, w.Start(context.Background()))

	source.Emit(RawFSEvent{Path: "/x/a.md", Kind: "create"})

	// Settle: the pump goroutine reads from a buffered channel, so
	// give Go's scheduler a moment to drain before we Advance.
	time.Sleep(20 * time.Millisecond)
	clock.Advance(100 * time.Millisecond)
	waitForEvents(t, getter, 1, 500*time.Millisecond)

	if got := getter()[0].Paths[0]; got != "/x/a.md" {
		t.Errorf("Paths[0] = %q", got)
	}
}

// ── Error / lifecycle ───────────────────────────────────────────────

func TestFSWatcher_WatchOnClosedRejected(t *testing.T) {
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	w := NewFSWatcher(source, clock)
	must(t, w.Close())
	handler, _ := captureHandler()
	if err := w.Watch(FSWatchSpec{ID: "x", Path: "/x"}, handler); err == nil {
		t.Error("expected error from Watch on closed watcher")
	}
}

func TestFSWatcher_WatchRequiresIDAndPath(t *testing.T) {
	source := NewFakeFSSource()
	clock := NewFakeClock(mustParse(t, testEpoch))
	w := NewFSWatcher(source, clock)
	defer w.Close()

	handler, _ := captureHandler()
	if err := w.Watch(FSWatchSpec{Path: "/x"}, handler); err == nil {
		t.Error("missing ID should error")
	}
	if err := w.Watch(FSWatchSpec{ID: "x"}, handler); err == nil {
		t.Error("missing Path should error")
	}
}

// ── mergeKinds (white-box) ──────────────────────────────────────────

func TestMergeKinds(t *testing.T) {
	cases := []struct {
		current, incoming, want string
	}{
		{"", "create", "create"},
		{"", "write", "write"},
		{"", "rename", "rename"},
		{"create", "write", "create"},
		{"write", "create", "create"},
		{"write", "rename", "write"},
		{"rename", "rename", "rename"},
		{"create", "create", "create"},
	}
	for _, c := range cases {
		t.Run(c.current+"+"+c.incoming, func(t *testing.T) {
			if got := mergeKinds(c.current, c.incoming); got != c.want {
				t.Errorf("mergeKinds(%q, %q) = %q, want %q", c.current, c.incoming, got, c.want)
			}
		})
	}
}

// ── helpers ─────────────────────────────────────────────────────────

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
