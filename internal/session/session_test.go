package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// setupTestMind creates a temporary mind directory structure for testing.
// It overrides HOME so ResolvePath points to the temp dir.
func setupTestMind(t *testing.T, mindID string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	sessionsDir := filepath.Join(tmp, ".universe", "minds", mindID, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatalf("creating test sessions dir: %v", err)
	}
	return tmp
}

func TestSaveAndLoad(t *testing.T) {
	setupTestMind(t, "test-mind")

	s := &Session{
		SessionID:  "sess-123",
		UniverseID: "uni-456",
		MindID:     "test-mind",
		CreatedAt:  time.Now().Truncate(time.Second),
		UpdatedAt:  time.Now().Truncate(time.Second),
	}

	if err := Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load("test-mind", "uni-456")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil {
		t.Fatal("Load returned nil for existing session")
	}
	if loaded.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", loaded.SessionID, "sess-123")
	}
	if loaded.MindID != "test-mind" {
		t.Errorf("MindID = %q, want %q", loaded.MindID, "test-mind")
	}
}

func TestLoad_NotFound(t *testing.T) {
	setupTestMind(t, "test-mind")

	loaded, err := Load("test-mind", "nonexistent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != nil {
		t.Error("Load should return nil for nonexistent session")
	}
}

func TestList(t *testing.T) {
	setupTestMind(t, "test-mind")

	for _, uid := range []string{"uni-1", "uni-2"} {
		s := &Session{
			SessionID:  "sess-" + uid,
			UniverseID: uid,
			MindID:     "test-mind",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := Save(s); err != nil {
			t.Fatalf("Save %s: %v", uid, err)
		}
	}

	sessions, err := List("test-mind")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestDelete(t *testing.T) {
	setupTestMind(t, "test-mind")

	s := &Session{
		SessionID:  "sess-del",
		UniverseID: "uni-del",
		MindID:     "test-mind",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Delete("test-mind", "uni-del"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	loaded, err := Load("test-mind", "uni-del")
	if err != nil {
		t.Fatalf("Load after delete: %v", err)
	}
	if loaded != nil {
		t.Error("session should be gone after Delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	setupTestMind(t, "test-mind")

	if err := Delete("test-mind", "nonexistent"); err != nil {
		t.Errorf("Delete nonexistent should not error, got: %v", err)
	}
}
