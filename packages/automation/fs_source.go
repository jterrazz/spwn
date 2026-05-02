package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// RawFSSource is the low-level event source the FSWatcher composes
// over. Production = fsnotifySource (this file); tests use
// FakeFSSource (also this file) so debounce + pattern + recursive
// logic is exercised without real filesystem timing.
//
// Events() returns a single read-only channel — the watcher's pump
// goroutine drains it. Close() must be safe to call multiple times.
type RawFSSource interface {
	// Add registers path with the source. Recursive=true means the
	// source must walk the tree and watch every subdirectory + handle
	// new subdirs created later.
	Add(path string, recursive bool) error
	// Events returns the read side of the source's event stream. The
	// channel closes when Close is called.
	Events() <-chan RawFSEvent
	// Close stops the source and closes the Events channel.
	Close() error
}

// RawFSEvent is the source-side event shape. A subset of fsnotify.Event
// — we drop ChMod (which fires on every metadata touch and is too
// noisy for the spwn use case) and we map fsnotify's CREATE/WRITE/
// REMOVE/RENAME enum to our string allow-list.
type RawFSEvent struct {
	Path string
	Kind string // create | write | rename
}

// ── Production source: fsnotify wrapper ─────────────────────────────

// fsnotifySource is the production RawFSSource. It owns one
// fsnotify.Watcher, walks the tree on Add when recursive, and watches
// new subdirectories as they appear.
type fsnotifySource struct {
	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	out      chan RawFSEvent
	doneOnce sync.Once
	done     chan struct{}
	// recursiveRoots tracks which Add calls were recursive so we
	// know whether to follow new subdirs created under them.
	recursiveRoots map[string]struct{}
}

// NewFSNotifySource constructs a production RawFSSource backed by
// fsnotify. Returns an error if the OS-level watcher can't be
// allocated (rare — typically inotify/kqueue exhaustion).
func NewFSNotifySource() (RawFSSource, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("fsnotify: %w", err)
	}
	s := &fsnotifySource{
		watcher:        w,
		out:            make(chan RawFSEvent, 64),
		done:           make(chan struct{}),
		recursiveRoots: make(map[string]struct{}),
	}
	go s.pump()
	return s, nil
}

// Add registers path. When recursive, walks the tree and adds every
// directory; new subdirs created later get auto-watched in pump().
func (s *fsnotifySource) Add(path string, recursive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if recursive {
		s.recursiveRoots[path] = struct{}{}
		return filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return s.watcher.Add(p)
			}
			return nil
		})
	}
	return s.watcher.Add(path)
}

// pump translates fsnotify events into RawFSEvent shape and forwards
// them. Auto-watches newly-created subdirs that fall under a
// recursive root.
func (s *fsnotifySource) pump() {
	defer close(s.out)
	for {
		select {
		case <-s.done:
			return
		case ev, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			kind := mapFSNotifyOp(ev.Op)
			if kind == "" {
				continue
			}
			// Auto-watch new directories under a recursive root. The
			// fsnotify lib doesn't recurse natively; we do it here.
			if kind == "create" && s.shouldRecurseInto(ev.Name) {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					_ = s.watcher.Add(ev.Name)
				}
			}
			select {
			case s.out <- RawFSEvent{Path: ev.Name, Kind: kind}:
			case <-s.done:
				return
			}
		case <-s.watcher.Errors:
			// fsnotify errors are non-fatal for our purposes — the
			// watcher continues. Drop the error rather than crashing
			// the engine; surfacing it would require an out-of-band
			// channel we don't have wired yet.
		}
	}
}

// shouldRecurseInto reports whether path is a descendant of any
// recursive root.
func (s *fsnotifySource) shouldRecurseInto(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for root := range s.recursiveRoots {
		if rel, err := filepath.Rel(root, path); err == nil && rel != ".." && len(rel) > 0 && rel[0] != '.' {
			return true
		}
	}
	return false
}

func (s *fsnotifySource) Events() <-chan RawFSEvent { return s.out }

func (s *fsnotifySource) Close() error {
	var err error
	s.doneOnce.Do(func() {
		close(s.done)
		err = s.watcher.Close()
	})
	return err
}

// mapFSNotifyOp folds fsnotify's bitmask into our string enum. Only
// translates the three ops the engine cares about; chmod-only events
// produce empty string and get dropped upstream.
func mapFSNotifyOp(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create == fsnotify.Create:
		return "create"
	case op&fsnotify.Write == fsnotify.Write:
		return "write"
	case op&fsnotify.Rename == fsnotify.Rename:
		return "rename"
	default:
		return ""
	}
}

// ── Test source: manually-driven emitter ────────────────────────────

// FakeFSSource is a RawFSSource for tests. Events are pushed via
// Emit; the watcher pump reads them as if they came from fsnotify.
//
// Goroutine-safe: Emit and Close can be called concurrently with the
// watcher's pump.
type FakeFSSource struct {
	mu     sync.Mutex
	out    chan RawFSEvent
	closed bool
	added  []addCall
}

// addCall records each Add invocation so tests can assert on the
// watcher's setup behaviour (e.g. "the recursive root was registered").
type addCall struct {
	Path      string
	Recursive bool
}

// NewFakeFSSource constructs an empty FakeFSSource. The output
// channel is buffered to 64 so tests can Emit a burst without
// blocking on the watcher's pump.
func NewFakeFSSource() *FakeFSSource {
	return &FakeFSSource{out: make(chan RawFSEvent, 64)}
}

// Add records the call and returns nil. Tests use AddCalls to assert.
func (s *FakeFSSource) Add(path string, recursive bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.added = append(s.added, addCall{Path: path, Recursive: recursive})
	return nil
}

// AddCalls returns a snapshot of every Add call made on this source.
func (s *FakeFSSource) AddCalls() []addCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]addCall, len(s.added))
	copy(out, s.added)
	return out
}

// Events returns the read-only channel.
func (s *FakeFSSource) Events() <-chan RawFSEvent { return s.out }

// Emit pushes a synthetic event to the pump. Blocks if the buffer is
// full — tests should consume promptly or raise the buffer.
func (s *FakeFSSource) Emit(ev RawFSEvent) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	s.out <- ev
}

// Close idempotently closes the event channel.
func (s *FakeFSSource) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	close(s.out)
	return nil
}
