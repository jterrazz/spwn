package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDefaultHierarchy(t *testing.T) {
	dir := t.TempDir()

	if err := EnsureDefaultHierarchy.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "hierarchies", "default.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := os.ReadFile(testdataPath(filepath.Join("003_after", "hierarchies", "default.yaml")))
	if string(got) != string(want) {
		t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestEnsureDefaultHierarchy_Idempotent(t *testing.T) {
	dir := t.TempDir()

	ctx := context.Background()
	if err := EnsureDefaultHierarchy.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	// Write a custom content to verify it does NOT overwrite.
	path := filepath.Join(dir, "hierarchies", "default.yaml")
	custom := []byte("custom: true\n")
	os.WriteFile(path, custom, 0644)

	if err := EnsureDefaultHierarchy.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != string(custom) {
		t.Error("second run overwrote existing file")
	}
}

func TestEnsureDefaultHierarchy_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	hDir := filepath.Join(dir, "hierarchies")
	os.MkdirAll(hDir, 0755)
	os.WriteFile(filepath.Join(hDir, "default.yaml"), []byte("existing"), 0644)

	if err := EnsureDefaultHierarchy.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(hDir, "default.yaml"))
	if string(got) != "existing" {
		t.Error("migration overwrote existing hierarchy file")
	}
}
