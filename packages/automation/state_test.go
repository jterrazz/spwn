package automation

import (
	"os"
	"path/filepath"
	"strings"
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

func TestFileStateStore_CorruptFileTaintsStore(t *testing.T) {
	// Corrupt JSON in state.json must NOT silently produce an empty
	// map that the next RecordFire could atomically overwrite —
	// that would clobber every other automation's cursor with one
	// hand-edit. The store flips into "tainted" mode and refuses
	// RecordFire until the file is repaired.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewFileStateStore(path)

	// LastFired returns not-found (engine treats this as first-boot).
	if _, ok := s.LastFired("brain", "x"); ok {
		t.Errorf("tainted store should report not-found from LastFired")
	}

	// IsTainted surfaces the parse error.
	if err := s.IsTainted(); err == nil {
		t.Errorf("IsTainted should return parse error, got nil")
	}

	// RecordFire refuses to overwrite the corrupt file.
	err := s.RecordFire("brain", "x", mustParse(t, "2026-05-01T06:00:00Z"))
	if err == nil {
		t.Errorf("RecordFire on tainted store should error")
	}

	// File on disk is unchanged.
	data, _ := os.ReadFile(path)
	if string(data) != "{not valid json" {
		t.Errorf("state file should be unchanged on tainted store, got: %s", string(data))
	}
}

func TestFileStateStore_LoadOnceCachesAcrossReads(t *testing.T) {
	// LastFired re-reads the file every call before this fix —
	// 100×5.7ms = 576ms boot stall. After the fix, the cache loads
	// once. We can't time it directly but we can confirm a corrupt
	// file written AFTER load doesn't change behaviour.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	s := NewFileStateStore(path)
	must(t, s.RecordFire("brain", "x", mustParse(t, "2026-05-01T06:00:00Z")))

	// Caller observes the value.
	if _, ok := s.LastFired("brain", "x"); !ok {
		t.Fatal("first LastFired should find the entry")
	}

	// Corrupt the file behind the store's back. Cache should hide
	// this from subsequent reads (well-behaved test fixture: nothing
	// else mutates the store while corruption is in place).
	if err := os.WriteFile(path, []byte("{not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.LastFired("brain", "x"); !ok {
		t.Errorf("cached read should still find entry after on-disk corruption")
	}
}

// ── Schema versioning + migration ───────────────────────────────────

func TestFileStateStore_LegacyBareMapMigratedOnRead(t *testing.T) {
	// Pre-v1 files were a bare flat map without the {version,
	// entries} envelope. The reader still loads them; the next
	// successful RecordFire rewrites in v1 shape.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	legacy := `{"brain/morning":"2026-05-01T06:00:00Z"}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewFileStateStore(path)
	got, ok := s.LastFired("brain", "morning")
	if !ok {
		t.Fatal("legacy file should be readable")
	}
	if !got.Equal(mustParse(t, "2026-05-01T06:00:00Z")) {
		t.Errorf("LastFired = %s", got)
	}

	// First RecordFire should rewrite in v1 envelope.
	must(t, s.RecordFire("brain", "morning", mustParse(t, "2026-05-02T06:00:00Z")))
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"version"`) {
		t.Errorf("after RecordFire, file should be v1 envelope; got: %s", string(data))
	}
	if !strings.Contains(string(data), `"entries"`) {
		t.Errorf("after RecordFire, file should have entries field; got: %s", string(data))
	}
}

func TestFileStateStore_FutureVersionTaintsStore(t *testing.T) {
	// Reading a file from a newer engine version: the store loads
	// what it can but refuses to write back (would drop the future
	// fields). User has to upgrade the engine or delete state.
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	future := `{"version":99,"entries":{"brain/x":"2026-05-01T06:00:00Z"},"future_field":"surprise"}`
	if err := os.WriteFile(path, []byte(future), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewFileStateStore(path)
	if err := s.IsTainted(); err == nil {
		t.Errorf("future-version file should taint the store")
	}
	// Reads still work (we got what we could).
	if _, ok := s.LastFired("brain", "x"); !ok {
		t.Errorf("LastFired should return the entry we loaded")
	}
	// Writes refuse to overwrite.
	if err := s.RecordFire("brain", "x", mustParse(t, "2026-05-02T06:00:00Z")); err == nil {
		t.Errorf("RecordFire on tainted-by-future-version should error")
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
