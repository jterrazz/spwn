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
