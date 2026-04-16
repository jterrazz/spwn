package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConsolidateJournal_BothExist(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents", "alice")

	// Create legacy journal with two files
	legacyDir := filepath.Join(agentDir, "journal")
	os.MkdirAll(legacyDir, 0755)
	os.WriteFile(filepath.Join(legacyDir, "2025-01-01.md"), []byte("legacy entry"), 0644)
	os.WriteFile(filepath.Join(legacyDir, "2025-01-02.md"), []byte("legacy second"), 0644)

	// Create memory/journal with one overlapping file
	memDir := filepath.Join(agentDir, "memory", "journal")
	os.MkdirAll(memDir, 0755)
	os.WriteFile(filepath.Join(memDir, "2025-01-01.md"), []byte("memory entry"), 0644)

	if err := ConsolidateJournal.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// legacy journal/ should be removed (was emptied)
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Error("expected legacy journal/ to be removed")
	}

	// Overlapping file should keep the memory version
	data, _ := os.ReadFile(filepath.Join(memDir, "2025-01-01.md"))
	if string(data) != "memory entry" {
		t.Errorf("overlapping file was overwritten: %s", data)
	}

	// Non-overlapping file should be moved
	data, err := os.ReadFile(filepath.Join(memDir, "2025-01-02.md"))
	if err != nil {
		t.Fatalf("non-overlapping file not moved: %v", err)
	}
	if string(data) != "legacy second" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestConsolidateJournal_OnlyLegacy(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents", "bob")

	legacyDir := filepath.Join(agentDir, "journal")
	os.MkdirAll(legacyDir, 0755)
	os.WriteFile(filepath.Join(legacyDir, "entry.md"), []byte("data"), 0644)

	if err := ConsolidateJournal.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// legacy should be gone
	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Error("expected legacy journal/ to be renamed away")
	}

	// memory/journal should now exist with the file
	data, err := os.ReadFile(filepath.Join(agentDir, "memory", "journal", "entry.md"))
	if err != nil {
		t.Fatalf("file not moved: %v", err)
	}
	if string(data) != "data" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestConsolidateJournal_NoLegacy(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents", "charlie")
	os.MkdirAll(filepath.Join(agentDir, "memory", "journal"), 0755)

	if err := ConsolidateJournal.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestConsolidateJournal_NoAgentsDir(t *testing.T) {
	dir := t.TempDir()
	if err := ConsolidateJournal.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error on missing agents dir, got: %v", err)
	}
}

func TestConsolidateJournal_Idempotent(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents", "dave")

	legacyDir := filepath.Join(agentDir, "journal")
	os.MkdirAll(legacyDir, 0755)
	os.WriteFile(filepath.Join(legacyDir, "entry.md"), []byte("data"), 0644)

	ctx := context.Background()
	if err := ConsolidateJournal.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if err := ConsolidateJournal.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(agentDir, "memory", "journal", "entry.md"))
	if string(data) != "data" {
		t.Errorf("unexpected content after second run: %s", data)
	}
}
