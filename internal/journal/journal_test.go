package journal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestMind(t *testing.T, mindID string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	journalDir := filepath.Join(tmp, ".universe", "minds", mindID, "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("creating test journal dir: %v", err)
	}
}

func TestAppend(t *testing.T) {
	setupTestMind(t, "test-mind")

	entry := Entry{
		UniverseID: "abcd1234-5678-9012-3456-789012345678",
		Image:      "ubuntu:22.04",
		Outcome:    "completed",
		ExitCode:   0,
		Duration:   5 * time.Minute,
		Timestamp:  time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC),
	}

	if err := Append("test-mind", entry); err != nil {
		t.Fatalf("Append: %v", err)
	}

	names, err := List("test-mind")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("expected 1 journal entry, got %d", len(names))
	}
	if names[0] != "2025-06-15_143000_abcd1234.md" {
		t.Errorf("unexpected filename: %s", names[0])
	}
}

func TestAppendMultiple(t *testing.T) {
	setupTestMind(t, "test-mind")

	for i, uid := range []string{"aaaa1111-0000-0000-0000-000000000000", "bbbb2222-0000-0000-0000-000000000000"} {
		entry := Entry{
			UniverseID: uid,
			Image:      "alpine:3.19",
			Outcome:    "completed",
			ExitCode:   0,
			Duration:   time.Minute,
			Timestamp:  time.Date(2025, 6, 15, 10+i, 0, 0, 0, time.UTC),
		}
		if err := Append("test-mind", entry); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	names, err := List("test-mind")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(names))
	}
	// Should be chronologically sorted.
	if names[0] >= names[1] {
		t.Errorf("entries should be sorted chronologically: %v", names)
	}
}

func TestList_Empty(t *testing.T) {
	setupTestMind(t, "test-mind")

	names, err := List("test-mind")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected 0 entries, got %d", len(names))
	}
}

func TestList_NoDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	names, err := List("nonexistent")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if names != nil {
		t.Errorf("expected nil for nonexistent mind, got %v", names)
	}
}
