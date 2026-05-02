package automation

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FSWatcher is the engine's filesystem trigger source. It composes a
// RawFSSource (production = fsnotify, tests = FakeFSSource) with the
// spwn-flavoured semantics on top:
//
//   - Recursive: walk the watch root once on Add, watch every subdir,
//     and auto-watch new subdirs as they appear.
//   - Pattern filter: filepath.Match against the basename. Empty
//     pattern list = match everything.
//   - Event filter: drop ops not in the allowed set (create / write /
//     rename).
//   - Debounce: coalesce a burst of events into one handler call. The
//     handler sees the de-duplicated path list, sorted for stability.
//
// The two-layer split keeps the spwn semantics testable without
// fsnotify timing quirks. RawFSSource is small enough to fake
// completely; FSWatcher is what every meaningful test exercises.
type FSWatcher struct {
	source RawFSSource
	clock  Clock

	mu      sync.Mutex
	specs   []*registeredFS
	closed  bool
	cancels []context.CancelFunc
	wg      sync.WaitGroup
}

// NewFSWatcher constructs an FSWatcher that pulls raw events from
// source and uses clock for debounce timing. Pass RealClock in
// production; FakeClock in tests so debounce windows are
// deterministic.
func NewFSWatcher(source RawFSSource, clock Clock) *FSWatcher {
	return &FSWatcher{source: source, clock: clock}
}

// FSWatchSpec is one filesystem trigger registration.
type FSWatchSpec struct {
	// ID is the engine's stable identifier for this watch — typically
	// "<world>/<automation>". Surfaces in dispatched events so the
	// handler knows which automation to fire.
	ID string

	// Path is the absolute (or project-relative, pre-resolved by the
	// caller) directory to watch.
	Path string

	// Events is the allowed op set. Empty = ApplyDefaults default
	// (["create"]).
	Events []string

	// Recursive watches every subdirectory of Path. New subdirs
	// created after Add are auto-watched.
	Recursive bool

	// Patterns are filepath.Match globs applied to the basename.
	// Empty = match everything.
	Patterns []string

	// Debounce coalesces bursts. Default 1s.
	Debounce time.Duration

	// IncludeHidden, when true, fires for paths inside directories
	// whose basename starts with `.` (e.g. `.cache/foo.md`). Default
	// false — most recursive watches don't want git/cache-internal
	// noise.
	IncludeHidden bool
}

// FSHandler is called with each debounced event batch. The engine's
// fire() pipeline plugs into this.
type FSHandler func(ev DebouncedFSEvent)

// DebouncedFSEvent is the post-debounce payload the handler receives.
type DebouncedFSEvent struct {
	// SpecID is the FSWatchSpec.ID this event corresponds to.
	SpecID string

	// Paths are the unique paths touched during the burst, sorted
	// alphabetically. Always non-empty (the handler is not invoked
	// for empty batches).
	Paths []string

	// Kind is the dominant op for the burst. When the burst includes
	// mixed ops, "create" wins over "write" wins over "rename" — the
	// rationale is "what's the most actionable label for the agent's
	// receipt". Tests pin the precedence.
	Kind string
}

// registeredFS is the engine's per-spec internal state.
type registeredFS struct {
	spec    FSWatchSpec
	handler FSHandler

	// allowed is the spec.Events set materialised as a map for O(1)
	// rejection.
	allowed map[string]struct{}

	// pending accumulates events during a debounce window. Reset on
	// every emission.
	mu          sync.Mutex
	pendingPaths map[string]struct{}
	pendingKind string
	timer       *fsDebounceTimer
}

// fsDebounceTimer is the FakeClock-friendly debounce timer. We can't
// use time.AfterFunc directly (it ignores our injected clock), so we
// roll a minimal scheduler keyed on Clock.After.
type fsDebounceTimer struct {
	cancel context.CancelFunc
}

// Watch registers spec with the watcher. Multiple specs may target
// the same path with different patterns/handlers — the source is
// asked once, the handler set fans out internally.
//
// Must be called before Start. Watch on a closed watcher returns an
// error.
func (w *FSWatcher) Watch(spec FSWatchSpec, handler FSHandler) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return fmt.Errorf("automation: Watch on closed FSWatcher")
	}
	if spec.ID == "" {
		return fmt.Errorf("automation: FSWatchSpec.ID is required")
	}
	if spec.Path == "" {
		return fmt.Errorf("automation: FSWatchSpec.Path is required")
	}
	allowed := make(map[string]struct{}, len(spec.Events))
	for _, e := range spec.Events {
		allowed[e] = struct{}{}
	}
	if len(allowed) == 0 {
		// Defence-in-depth: ApplyDefaults stamps create at parse
		// time, but a programmatic caller could skip it.
		allowed["create"] = struct{}{}
	}
	if spec.Debounce == 0 {
		spec.Debounce = 1 * time.Second
	}

	r := &registeredFS{
		spec:         spec,
		handler:      handler,
		allowed:      allowed,
		pendingPaths: make(map[string]struct{}),
	}
	w.specs = append(w.specs, r)
	return w.source.Add(spec.Path, spec.Recursive)
}

// Start launches the event-pump goroutine. Returns immediately; events
// drain in the background until ctx is cancelled or Close is called.
//
// Calling Start more than once is a programming error.
func (w *FSWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("automation: Start on closed FSWatcher")
	}
	ctx, cancel := context.WithCancel(ctx)
	w.cancels = append(w.cancels, cancel)
	w.mu.Unlock()

	w.wg.Add(1)
	go w.pump(ctx)
	return nil
}

// Close cancels every pump goroutine, every in-flight debounce
// timer, and frees the source. Safe to call multiple times;
// subsequent calls are no-ops.
//
// Cancelling timer-level contexts (in addition to the pump's) is
// what lets tests use plain context.Background() when calling
// handle() directly: debounce goroutines registered against that
// untouched ctx still exit when Close fires their per-spec cancel.
func (w *FSWatcher) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	cancels := w.cancels
	w.cancels = nil
	specs := append([]*registeredFS(nil), w.specs...)
	w.mu.Unlock()

	for _, c := range cancels {
		c()
	}
	for _, r := range specs {
		r.mu.Lock()
		if r.timer != nil {
			r.timer.cancel()
			r.timer = nil
		}
		r.mu.Unlock()
	}
	w.wg.Wait()
	return w.source.Close()
}

// HandleForTest is a test-only entry point that synchronously drives
// the watcher's filter chain for one event. Skips the source pump
// goroutine so integration tests don't race fsnotify timing. Same
// semantics as a real raw event arriving via the source — passes
// through pathCoveredBy / events filter / pattern match / debounce.
func (w *FSWatcher) HandleForTest(path, kind string) {
	w.handle(context.Background(), RawFSEvent{Path: path, Kind: kind})
}

// pump drains events from the source and dispatches them to the
// matching specs. One source produces one stream; the watcher fans
// out to every spec whose path-prefix matches.
//
// Tests usually bypass the pump and call handle directly — synchronous
// dispatch lets them assert against deterministic event ordering
// without racing the goroutine. One integration test exercises the
// pump's lifecycle (TestFSWatcher_PumpDelivers).
func (w *FSWatcher) pump(ctx context.Context) {
	defer w.wg.Done()
	events := w.source.Events()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			w.handle(ctx, ev)
		}
	}
}

// handle routes one raw event to every spec whose path covers it,
// applies the spec's filters, and adds the path to the pending burst.
func (w *FSWatcher) handle(ctx context.Context, ev RawFSEvent) {
	w.mu.Lock()
	specs := append([]*registeredFS(nil), w.specs...)
	w.mu.Unlock()

	for _, r := range specs {
		if !pathCoveredBy(r.spec, ev.Path) {
			continue
		}
		if _, ok := r.allowed[ev.Kind]; !ok {
			continue
		}
		if !matchPatterns(r.spec.Patterns, filepath.Base(ev.Path)) {
			continue
		}
		w.enqueue(ctx, r, ev)
	}
}

// enqueue adds the event to the spec's pending burst and (re)arms
// the debounce timer. If a timer is already running, this resets it
// — the handler doesn't fire until the burst quietens for spec.Debounce.
//
// clock.After is called synchronously (before the goroutine starts)
// so callers observing clock.Pending() immediately after handle()
// returns see the timer registered. Without this, tests racing the
// goroutine's first scheduling slice could Advance past the timer's
// would-be deadline before it was even registered.
func (w *FSWatcher) enqueue(ctx context.Context, r *registeredFS, ev RawFSEvent) {
	r.mu.Lock()
	r.pendingPaths[ev.Path] = struct{}{}
	r.pendingKind = mergeKinds(r.pendingKind, ev.Kind)

	// Cancel any in-flight timer so the new event resets the window.
	if r.timer != nil {
		r.timer.cancel()
		r.timer = nil
	}

	// Snapshot what we'll fire if quiet, then register the timer
	// synchronously (the channel sit on the goroutine, but the
	// clock.Pending side-effect happens here).
	debounce := r.spec.Debounce
	tCtx, tCancel := context.WithCancel(ctx)
	r.timer = &fsDebounceTimer{cancel: tCancel}
	timerCh := w.clock.After(debounce)
	r.mu.Unlock()

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		select {
		case <-tCtx.Done():
			return
		case <-timerCh:
			w.flush(r)
		}
	}()
}

// flush emits the pending burst as a single DebouncedFSEvent and
// resets the spec's accumulator. Called by the debounce goroutine
// when the window expires undisturbed.
func (w *FSWatcher) flush(r *registeredFS) {
	r.mu.Lock()
	if len(r.pendingPaths) == 0 {
		r.mu.Unlock()
		return
	}
	paths := make([]string, 0, len(r.pendingPaths))
	for p := range r.pendingPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	kind := r.pendingKind
	r.pendingPaths = make(map[string]struct{})
	r.pendingKind = ""
	r.timer = nil
	handler := r.handler
	specID := r.spec.ID
	r.mu.Unlock()

	handler(DebouncedFSEvent{SpecID: specID, Paths: paths, Kind: kind})
}

// pathCoveredBy reports whether ev.Path falls under spec.Path —
// directly when Recursive is false, or anywhere beneath when
// Recursive is true. The watcher trusts the source to only emit
// events from registered subtrees, but this guard catches bugs in
// fake sources and prevents cross-spec leakage when two specs share
// a parent.
//
// Hidden-directory filtering: when Recursive is true and
// IncludeHidden is false (default), paths whose relative dir starts
// with `.` are excluded — git internal writes, editor swap files,
// .cache directories. Most recursive watches want this. Authors who
// genuinely need to watch dotfiles (configs, dotfile editors) set
// IncludeHidden: true.
func pathCoveredBy(spec FSWatchSpec, eventPath string) bool {
	if eventPath == "" {
		return false
	}
	dir := filepath.Dir(eventPath)
	if dir == spec.Path {
		return true
	}
	if !spec.Recursive {
		return false
	}
	// Recursive: ensure dir is *under* spec.Path. filepath.Rel is the
	// portable check; reject paths that escape via "..".
	rel, err := filepath.Rel(spec.Path, dir)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || len(rel) == 0 {
		return false
	}
	// Hidden-dir filter (default-on). Skip when the relative path
	// has any component starting with `.` (e.g. `.git/objects` or
	// `sub/.cache`).
	if !spec.IncludeHidden {
		for _, segment := range strings.Split(rel, string(filepath.Separator)) {
			if len(segment) > 0 && segment[0] == '.' {
				return false
			}
		}
	}
	return true
}

// matchPatterns returns true if patterns is empty (match all) or any
// pattern matches the basename via filepath.Match. Per-pattern errors
// are treated as "no match" rather than crashing the pump.
func matchPatterns(patterns []string, base string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		ok, err := filepath.Match(p, base)
		if err == nil && ok {
			return true
		}
	}
	return false
}

// mergeKinds picks the dominant op label for a burst. Precedence:
// create > write > rename. Once a burst contains a Create event the
// kind locks to "create" — that's the most actionable label for the
// receipt and matches user mental model ("a new file appeared, with
// some writes during the appearing").
func mergeKinds(current, incoming string) string {
	if current == "create" || incoming == "create" {
		return "create"
	}
	if current == "write" || incoming == "write" {
		return "write"
	}
	if incoming != "" {
		return incoming
	}
	return current
}
