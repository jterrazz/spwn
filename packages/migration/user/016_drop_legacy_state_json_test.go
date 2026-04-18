package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestDropLegacyStateJSON_RemovesLegacyArtifacts covers the real
// pre-labels install shape: state.json + state.json.bak + runtime/
// all populated. After the migration none should exist.
func TestDropLegacyStateJSON_RemovesLegacyArtifacts(t *testing.T) {
	baseDir := t.TempDir()

	// Seed the three legacy artifacts.
	mustWrite(t, filepath.Join(baseDir, "state.json"), `{"worlds":[]}`)
	mustWrite(t, filepath.Join(baseDir, "state.json.bak"), `{"worlds":[]}`)
	runtimeDir := filepath.Join(baseDir, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(runtimeDir, "w-1.json"), `{}`)

	if err := DropLegacyStateJSON.Apply(context.Background(), baseDir); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	for _, path := range []string{"state.json", "state.json.bak", "runtime"} {
		if _, err := os.Stat(filepath.Join(baseDir, path)); !os.IsNotExist(err) {
			t.Errorf("%s should be gone, stat err = %v", path, err)
		}
	}
}

// TestDropLegacyStateJSON_NoopOnFreshInstall verifies the migration
// is safe on a fresh install where none of the three legacy paths
// exist. This is the shape every new install hits.
func TestDropLegacyStateJSON_NoopOnFreshInstall(t *testing.T) {
	baseDir := t.TempDir()
	if err := DropLegacyStateJSON.Apply(context.Background(), baseDir); err != nil {
		t.Errorf("Apply on empty baseDir should be no-op; got %v", err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
