package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveOrphanedUniverses(t *testing.T) {
	dir := t.TempDir()
	universesDir := filepath.Join(dir, "universes")
	os.MkdirAll(filepath.Join(universesDir, "sub"), 0755)
	os.WriteFile(filepath.Join(universesDir, "old.json"), []byte("{}"), 0644)

	if err := RemoveOrphanedUniverses.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	if _, err := os.Stat(universesDir); !os.IsNotExist(err) {
		t.Error("expected universes/ to be removed")
	}
}

func TestRemoveOrphanedUniverses_NoDir(t *testing.T) {
	dir := t.TempDir()
	if err := RemoveOrphanedUniverses.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error on missing dir, got: %v", err)
	}
}

func TestRemoveOrphanedUniverses_Idempotent(t *testing.T) {
	dir := t.TempDir()
	universesDir := filepath.Join(dir, "universes")
	os.MkdirAll(universesDir, 0755)

	ctx := context.Background()
	if err := RemoveOrphanedUniverses.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	// Second run on already-removed dir
	if err := RemoveOrphanedUniverses.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
}
