package mind

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePath(t *testing.T) {
	path, err := ResolvePath("test-mind")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(path, ".universe") {
		t.Errorf("path should contain .universe, got %s", path)
	}
	if !strings.HasSuffix(path, filepath.Join("minds", "test-mind")) {
		t.Errorf("path should end with minds/test-mind, got %s", path)
	}
}

func TestEnsureDir(t *testing.T) {
	tmp := t.TempDir()
	mindPath := filepath.Join(tmp, "test-mind")

	if err := EnsureDir(mindPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, sub := range subdirs {
		dir := filepath.Join(mindPath, sub)
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("subdir %s should exist: %v", sub, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("subdir %s should be a directory", sub)
		}
	}
}

func TestSubdirs(t *testing.T) {
	got := Subdirs()
	if len(got) != 6 {
		t.Fatalf("expected 6 subdirs, got %d", len(got))
	}
	expected := []string{"personas", "skills", "knowledge", "playbooks", "journal", "sessions"}
	for i, name := range expected {
		if got[i] != name {
			t.Errorf("subdirs[%d] = %q, want %q", i, got[i], name)
		}
	}
	// Verify it returns a copy.
	got[0] = "modified"
	if Subdirs()[0] == "modified" {
		t.Error("Subdirs should return a copy, not the original slice")
	}
}

func TestValidate_Valid(t *testing.T) {
	tmp := t.TempDir()
	mindPath := filepath.Join(tmp, "valid-mind")
	if err := EnsureDir(mindPath); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if err := Validate(mindPath); err != nil {
		t.Errorf("Validate should pass for complete mind, got: %v", err)
	}
}

func TestValidate_Missing(t *testing.T) {
	tmp := t.TempDir()
	mindPath := filepath.Join(tmp, "incomplete-mind")
	// Create only some subdirs.
	os.MkdirAll(filepath.Join(mindPath, "personas"), 0o755)
	os.MkdirAll(filepath.Join(mindPath, "skills"), 0o755)

	err := Validate(mindPath)
	if err == nil {
		t.Fatal("Validate should return error for incomplete mind")
	}
	if !strings.Contains(err.Error(), "knowledge") {
		t.Errorf("error should mention missing 'knowledge', got: %v", err)
	}
}

func TestListMinds(t *testing.T) {
	// ListMinds reads from ~/.universe/minds/ which may or may not exist.
	// We just verify it doesn't crash.
	_, err := ListMinds()
	if err != nil {
		t.Fatalf("ListMinds: %v", err)
	}
}

func TestBasePath(t *testing.T) {
	path, err := BasePath()
	if err != nil {
		t.Fatalf("BasePath: %v", err)
	}
	if !strings.HasSuffix(path, filepath.Join(".universe", "minds")) {
		t.Errorf("BasePath should end with .universe/minds, got %s", path)
	}
}
