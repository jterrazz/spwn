package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeterministicID(t *testing.T) {
	id1 := DeterministicID("neo", "w-default-12345")
	id2 := DeterministicID("neo", "w-default-12345")
	id3 := DeterministicID("trinity", "w-default-12345")

	if id1 != id2 {
		t.Error("same inputs should produce same ID")
	}
	if id1 == id3 {
		t.Error("different agents should produce different IDs")
	}
	// UUID format: xxxxxxxx-xxxx-4xxx-axxx-xxxxxxxxxxxx
	if len(id1) != 36 {
		t.Errorf("expected UUID length 36, got %d", len(id1))
	}
	if id1[8] != '-' || id1[13] != '-' || id1[18] != '-' || id1[23] != '-' {
		t.Error("invalid UUID format")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	mindPath := tmp

	s := &Session{
		ID:         "test-session-id",
		AgentName:  "neo",
		UniverseID: "w-default-12345",
		Resumed:    false,
	}

	if err := Save(mindPath, s); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(mindPath, "w-default-12345")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded session is nil")
	}
	if loaded.ID != "test-session-id" {
		t.Errorf("expected ID test-session-id, got %s", loaded.ID)
	}
	if loaded.AgentName != "neo" {
		t.Errorf("expected agent neo, got %s", loaded.AgentName)
	}
}

func TestLoadMissing(t *testing.T) {
	tmp := t.TempDir()
	loaded, err := Load(tmp, "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil for missing session")
	}
}

func TestList(t *testing.T) {
	tmp := t.TempDir()

	// Save two sessions
	Save(tmp, &Session{ID: "s1", AgentName: "neo", UniverseID: "w-1"})
	Save(tmp, &Session{ID: "s2", AgentName: "neo", UniverseID: "w-2"})

	sessions, err := List(tmp)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestListEmpty(t *testing.T) {
	tmp := t.TempDir()
	sessions, err := List(tmp)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0, got %d", len(sessions))
	}
}

func TestSessionFilePersistence(t *testing.T) {
	tmp := t.TempDir()
	Save(tmp, &Session{ID: "persist", AgentName: "neo", UniverseID: "w-test"})

	// Verify file exists
	path := filepath.Join(tmp, "sessions", "w-test.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("session file not created")
	}
}
