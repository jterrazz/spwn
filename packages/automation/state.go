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

// stateSchemaVersion is the on-disk format version. Bump when the
// schema changes incompatibly. The store reads any version it
// recognises and writes the current version; a future version's
// file format is detected via the explicit `version` field, which
// lets readers branch on schema generation without guessing.
const stateSchemaVersion = 1

// stateFile is the on-disk shape. Versioned envelope around the
// flat map so future widenings (struct values for entries, etc) can
// land without breaking existing files. Old format (a bare flat
// map without the envelope) is detected at load time and migrated
// in-memory.
type stateFile struct {
	Version int                  `json:"version"`
	Entries map[string]time.Time `json:"entries"`
}

// FileStateStore persists state as JSON. Single file, mutex-guarded,
// rewritten atomically (write-temp + rename) on every RecordFire so a
// crash mid-update can never leave a half-written entry.
//
// On-disk shape (v1):
//
//	{
//	  "version": 1,
//	  "entries": { "<world>/<name>": "<RFC3339>", ... }
//	}
//
// Old shape (no envelope, bare `{ "<world>/<name>": "<RFC3339>" }`)
// is detected at load and migrated in memory; the next RecordFire
// rewrites in v1 shape.
//
// The map is loaded once on first access and held in memory; reads
// are O(1) instead of file-IO per call. Writes update the in-memory
// map AND rewrite the file atomically, so a crash leaves the cache
// and disk consistent on next boot.
//
// Corruption safety: if the on-disk JSON fails to parse, the store
// flips into a "tainted" mode that refuses RecordFire so a subsequent
// successful fire can't atomically overwrite the (presumably
// hand-edited or partially-written) file and lose the rest of the
// project's cursors. The engine surfaces the write error; callers
// see the surface and can intervene before more data is at risk.
type FileStateStore struct {
	Path string

	mu      sync.Mutex
	loaded  bool
	tainted error                // non-nil if the file existed but failed to parse
	entries map[string]time.Time // in-memory cache, mirrors disk
}

// NewFileStateStore constructs a writer rooted at path. The file is
// loaded lazily on first read — boot stays fast even for projects
// with no prior state.
func NewFileStateStore(path string) *FileStateStore {
	return &FileStateStore{Path: path}
}

// LastFired returns the persisted cursor. Reads the in-memory cache;
// the cache is loaded from disk on first call.
func (s *FileStateStore) LastFired(world, name string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	t, ok := s.entries[stateKey(world, name)]
	return t, ok
}

// RecordFire persists scheduled and rewrites the file atomically.
// Returns an error when the on-disk file was corrupt at load — the
// store refuses to write so the existing data isn't clobbered.
// Engine treats other state-write failures as non-fatal (the next
// catch-up re-detects the missed slot), but a tainted store means
// the user has to intervene.
func (s *FileStateStore) RecordFire(world, name string, scheduled time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	if s.tainted != nil {
		return fmt.Errorf("state file is tainted (parse failed at load) — refusing to overwrite: %w", s.tainted)
	}
	if s.entries == nil {
		s.entries = map[string]time.Time{}
	}
	s.entries[stateKey(world, name)] = scheduled
	return s.writeLocked(s.entries)
}

// ensureLoadedLocked populates the in-memory cache from disk on
// first call. Subsequent calls are O(1). Caller holds s.mu.
//
// Tries the v1 envelope shape first; falls back to the legacy bare-
// map shape so existing on-disk files keep working without a
// migration step. The next RecordFire rewrites in v1 shape.
func (s *FileStateStore) ensureLoadedLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	data, err := os.ReadFile(s.Path)
	if err != nil {
		// Missing file is the empty-state case, not an error.
		s.entries = map[string]time.Time{}
		return
	}

	// Try v1 envelope first. If it parses with a non-empty Entries
	// (or with version >= 1), it's a v1 file.
	var v1 stateFile
	if err := json.Unmarshal(data, &v1); err == nil && (v1.Version >= 1 || v1.Entries != nil) {
		// Future-version file: warn but try our best to load. A
		// reader from the future may have added fields per-entry;
		// we only know how to read time.Time values.
		if v1.Version > stateSchemaVersion {
			// Tainted: writing back would lose the future fields.
			s.tainted = fmt.Errorf("state file is version %d, this engine knows version %d", v1.Version, stateSchemaVersion)
		}
		if v1.Entries == nil {
			v1.Entries = map[string]time.Time{}
		}
		s.entries = v1.Entries
		return
	}

	// Legacy bare-map shape (pre-v1). Migrate in memory.
	var legacy map[string]time.Time
	if err := json.Unmarshal(data, &legacy); err != nil {
		// File exists but parses as neither shape. Keep cache empty
		// + mark tainted so the engine won't clobber it.
		s.entries = map[string]time.Time{}
		s.tainted = err
		return
	}
	if legacy == nil {
		legacy = map[string]time.Time{}
	}
	s.entries = legacy
}

// IsTainted reports whether the store refused to load the file due
// to parse error. Returns the underlying parse error or nil.
//
// Callers (CLI status, daemon boot) should check this and surface
// the error to the user — a silent tainted store means catch-up
// won't fire and the user gets no warning.
func (s *FileStateStore) IsTainted() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureLoadedLocked()
	return s.tainted
}

// writeLocked atomically replaces the state file. Caller holds s.mu.
// Always writes the v1 envelope shape, even when the loaded file
// was legacy bare-map — first successful RecordFire upgrades the
// on-disk format.
func (s *FileStateStore) writeLocked(entries map[string]time.Time) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("state dir: %w", err)
	}
	envelope := stateFile{
		Version: stateSchemaVersion,
		Entries: entries,
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
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
