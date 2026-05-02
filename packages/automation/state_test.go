package automation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── MemoryStateStore ────────────────────────────────────────────────

func TestMemoryStateStore_RoundTrip(t *testing.T) {
	s := NewMemoryStateStore()
	if _, ok := s.LastFired("brain", "morning-brief"); ok {
		t.Errorf("empty store should report not-found")
	}

	scheduled := mustParse(t, "2026-05-01T06:00:00Z")
	if err := s.RecordFire("brain", "morning-brief", scheduled); err != nil {
		t.Fatalf("RecordFire: %v", err)
	}
	got, ok := s.LastFired("brain", "morning-brief")
	if !ok {
		t.Fatalf("LastFired ok = false after RecordFire")
	}
	if !got.Equal(scheduled) {
		t.Errorf("LastFired = %s, want %s", got, scheduled)
	}
}

func TestMemoryStateStore_OverwritesOnRefire(t *testing.T) {
	s := NewMemoryStateStore()
	t1 := mustParse(t, "2026-05-01T06:00:00Z")
	t2 := mustParse(t, "2026-05-02T06:00:00Z")
	must(t, s.RecordFire("brain", "x", t1))
	must(t, s.RecordFire("brain", "x", t2))
	got, _ := s.LastFired("brain", "x")
	if !got.Equal(t2) {
		t.Errorf("LastFired = %s, want %s (second fire should overwrite)", got, t2)
	}
}

func TestMemoryStateStore_IsolatesPerKey(t *testing.T) {
	s := NewMemoryStateStore()
	must(t, s.RecordFire("brain", "x", mustParse(t, "2026-05-01T06:00:00Z")))
	must(t, s.RecordFire("brain", "y", mustParse(t, "2026-05-01T07:00:00Z")))
	must(t, s.RecordFire("scratch", "x", mustParse(t, "2026-05-01T08:00:00Z")))

	if got, _ := s.LastFired("brain", "x"); !got.Equal(mustParse(t, "2026-05-01T06:00:00Z")) {
		t.Errorf("brain/x cross-talked: got %s", got)
	}
	if got, _ := s.LastFired("scratch", "x"); !got.Equal(mustParse(t, "2026-05-01T08:00:00Z")) {
		t.Errorf("scratch/x cross-talked: got %s", got)
	}
}

// ── FileStateStore ──────────────────────────────────────────────────

func TestFileStateStore_PersistsAcrossInstances(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	scheduled := mustParse(t, "2026-05-01T06:00:00Z")
	s1 := NewFileStateStore(path)
	must(t, s1.RecordFire("brain", "morning-brief", scheduled))

	// Fresh store reading the same file (simulating an architect restart).
	s2 := NewFileStateStore(path)
	got, ok := s2.LastFired("brain", "morning-brief")
	if !ok {
		t.Fatalf("second instance found no entry")
	}
	if !got.Equal(scheduled) {
		t.Errorf("read-back = %s, want %s", got, scheduled)
	}
}

func TestFileStateStore_MissingFileEmptyState(t *testing.T) {
	dir := t.TempDir()
	s := NewFileStateStore(filepath.Join(dir, "absent.json"))
	if _, ok := s.LastFired("brain", "x"); ok {
		t.Errorf("missing file should report not-found, not crash")
	}
}

func TestFileStateStore_AtomicReplace(t *testing.T) {
	// We can't easily simulate a crash mid-write, but we can verify
	// the .tmp file is cleaned up on a successful write — a leftover
	// would mean Rename failed silently.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s := NewFileStateStore(path)
	must(t, s.RecordFire("brain", "x", mustParse(t, "2026-05-01T06:00:00Z")))

	// No .tmp leftover — Rename succeeded.
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Errorf(".tmp file leaked after RecordFire")
	}
}

func TestFileStateStore_ParentDirCreated(t *testing.T) {
	dir := t.TempDir()
	// Nested path that doesn't exist yet — RecordFire should mkdir it.
	path := filepath.Join(dir, "deep", "nested", "state.json")
	s := NewFileStateStore(path)
	if err := s.RecordFire("brain", "x", time.Now()); err != nil {
		t.Fatalf("RecordFire on deep path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("state file not created: %v", err)
	}
}
