package upgrade

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadVersionMissingFile(t *testing.T) {
	dir := t.TempDir()
	v, err := LoadVersion(dir)
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	if v.Version != 0 {
		t.Errorf("version = %d, want 0", v.Version)
	}
	if len(v.Applied) != 0 {
		t.Errorf("applied = %d, want 0", len(v.Applied))
	}
}

func TestLoadVersionExistingFile(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "version": 5,
  "updated_at": "2025-01-15T10:30:00Z",
  "applied": [
    {"number": 1, "description": "init", "applied_at": "2025-01-15T10:00:00Z"},
    {"number": 5, "description": "five", "applied_at": "2025-01-15T10:30:00Z"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, versionFile), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	v, err := LoadVersion(dir)
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	if v.Version != 5 {
		t.Errorf("version = %d, want 5", v.Version)
	}
	if len(v.Applied) != 2 {
		t.Errorf("applied count = %d, want 2", len(v.Applied))
	}
	if v.Applied[0].Description != "init" {
		t.Errorf("first description = %q, want %q", v.Applied[0].Description, "init")
	}
}

func TestLoadVersionInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, versionFile), []byte(`not json`), 0644)
	_, err := LoadVersion(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSaveVersionWritesValidJSON(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	v := &SchemaVersion{
		Version:   3,
		UpdatedAt: now,
		Applied: []AppliedMigration{
			{Number: 1, Description: "first", AppliedAt: now},
			{Number: 3, Description: "third", AppliedAt: now},
		},
	}

	if err := SaveVersion(dir, v); err != nil {
		t.Fatalf("SaveVersion: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, versionFile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty file")
	}
	// Must end with newline
	if data[len(data)-1] != '\n' {
		t.Error("file does not end with newline")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	original := &SchemaVersion{
		Version:   7,
		UpdatedAt: now,
		Applied: []AppliedMigration{
			{Number: 5, Description: "five", AppliedAt: now},
			{Number: 7, Description: "seven", AppliedAt: now},
		},
	}

	if err := SaveVersion(dir, original); err != nil {
		t.Fatalf("SaveVersion: %v", err)
	}

	loaded, err := LoadVersion(dir)
	if err != nil {
		t.Fatalf("LoadVersion: %v", err)
	}
	if loaded.Version != original.Version {
		t.Errorf("version = %d, want %d", loaded.Version, original.Version)
	}
	if len(loaded.Applied) != len(original.Applied) {
		t.Fatalf("applied count = %d, want %d", len(loaded.Applied), len(original.Applied))
	}
	for i, a := range loaded.Applied {
		if a.Number != original.Applied[i].Number {
			t.Errorf("applied[%d].Number = %d, want %d", i, a.Number, original.Applied[i].Number)
		}
		if a.Description != original.Applied[i].Description {
			t.Errorf("applied[%d].Description = %q, want %q", i, a.Description, original.Applied[i].Description)
		}
	}
}
