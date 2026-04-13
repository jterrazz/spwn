package architect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSpwnRoot_ValidRoot(t *testing.T) {
	// Create a temp directory that looks like a spwn workspace root
	root := t.TempDir()

	// Create go.work
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create apps/cli/cmd/spwn/main.go
	mainDir := filepath.Join(root, "apps", "cli", "cmd", "spwn")
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if !isSpwnRoot(root) {
		t.Error("expected isSpwnRoot to return true for valid root")
	}
}

func TestIsSpwnRoot_InvalidRoot(t *testing.T) {
	root := t.TempDir()
	if isSpwnRoot(root) {
		t.Error("expected isSpwnRoot to return false for empty directory")
	}
}

func TestIsSpwnRoot_OnlyGoWork(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0644)

	if isSpwnRoot(root) {
		t.Error("expected isSpwnRoot to return false with only go.work")
	}
}

func TestFindRootUpward_FindsRoot(t *testing.T) {
	// Create a fake spwn root
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0644)
	mainDir := filepath.Join(root, "apps", "cli", "cmd", "spwn")
	os.MkdirAll(mainDir, 0755)
	os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n"), 0644)

	// Create a nested subdirectory
	nested := filepath.Join(root, "packages", "world", "internal")
	os.MkdirAll(nested, 0755)

	found := findRootUpward(nested)
	if found != root {
		t.Errorf("expected to find root %q, got %q", root, found)
	}
}

func TestFindRootUpward_NotFound(t *testing.T) {
	dir := t.TempDir()
	found := findRootUpward(dir)
	if found != "" {
		t.Errorf("expected empty string for non-spwn directory, got %q", found)
	}
}
