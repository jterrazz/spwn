package automation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateStore persists the engine's last-fired cursor across restarts.
// On boot the engine compares LastFired(world, name) against the
// current time to compute catch-up.
//
// Production = FileStateStore (JSON file at <project>/.spwn/automations/state.json).
// Tests = MemoryStateStore.
//
// The cursor is the SCHEDULED time of the last successful fire, not
// the wall-clock Fired time. Recording the scheduled time means a
// catch-up that ran 2h late advances the cursor to the slot it
// covered, not to "now+2h" — preventing the next catch-up from
// double-counting that slot or skipping the next.
type StateStore interface {
	LastFired(world, name string) (time.Time, bool)
	RecordFire(world, name string, scheduled time.Time) error
}

// MemoryStateStore is a test StateStore. Goroutine-safe.
type MemoryStateStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

// NewMemoryStateStore constructs an empty store.
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{entries: make(map[string]time.Time)}
}

// LastFired returns the cursor for (world, name), or zero+false if
// the automation has never recorded a fire.
func (s *MemoryStateStore) LastFired(world, name string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.entries[stateKey(world, name)]
	return t, ok
}

// RecordFire persists scheduled as the cursor for (world, name).
func (s *MemoryStateStore) RecordFire(world, name string, scheduled time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[stateKey(world, name)] = scheduled
	return nil
}

// FileStateStore persists state as JSON. Single file, mutex-guarded,
// rewritten atomically (write-temp + rename) on every RecordFire so a
// crash mid-update can never leave a half-written entry.
//
// Schema-stable: the on-disk shape is a flat map of "<world>/<name>"
// → RFC3339 timestamps. Adding fields later (e.g. "last_run_id" for
// dashboard correlation) means widening to a struct value, which can
// be done without breaking old files via a migration step.
type FileStateStore struct {
	Path string
	mu   sync.Mutex
}

// NewFileStateStore constructs a writer rooted at path. The file is
// loaded lazily on first read — boot stays fast even for projects
// with no prior state.
func NewFileStateStore(path string) *FileStateStore {
	return &FileStateStore{Path: path}
}

// LastFired returns the persisted cursor, reading the file each call.
// JSON re-decode is cheap (the file holds at most one entry per
// automation) and stale-cache hazards beat the perf cost.
func (s *FileStateStore) LastFired(world, name string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, _ := s.readLocked()
	t, ok := entries[stateKey(world, name)]
	return t, ok
}

// RecordFire persists scheduled and rewrites the file atomically.
// Errors propagate so the engine can log them, but the engine treats
// state-write failures as non-fatal: the next catch-up will
// re-detect the missed slot and re-fire, matching the
// "rappels-Apple" semantics of the catch-up design.
func (s *FileStateStore) RecordFire(world, name string, scheduled time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, _ := s.readLocked()
	if entries == nil {
		entries = map[string]time.Time{}
	}
	entries[stateKey(world, name)] = scheduled
	return s.writeLocked(entries)
}

// readLocked parses the on-disk JSON. Caller holds s.mu.
func (s *FileStateStore) readLocked() (map[string]time.Time, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		// Missing file is the empty-state case, not an error.
		return map[string]time.Time{}, nil
	}
	var raw map[string]time.Time
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]time.Time{}, fmt.Errorf("state file %s parse: %w", s.Path, err)
	}
	return raw, nil
}

// writeLocked atomically replaces the state file. Caller holds s.mu.
func (s *FileStateStore) writeLocked(entries map[string]time.Time) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("state dir: %w", err)
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write state tmp: %w", err)
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		return fmt.Errorf("rename state: %w", err)
	}
	return nil
}

// stateKey is the registry key for a (world, name) pair. The
// "<world>/<name>" form mirrors the user-facing CLI form
// (`spwn automation run brain/morning-brief`) so logs and on-disk
// state match what users type.
func stateKey(world, name string) string {
	return world + "/" + name
}
