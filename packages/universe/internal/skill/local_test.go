package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalRegistryListEmpty(t *testing.T) {
	r := NewLocal("/tmp/nonexistent-skills-dir-for-test")
	skills, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if skills != nil {
		t.Errorf("List() = %v, want nil", skills)
	}
}

func TestLocalRegistryListWithSkills(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "skill-a"), 0755)
	os.MkdirAll(filepath.Join(dir, "skill-b"), 0755)
	// Create a file (should be ignored, only dirs count)
	os.WriteFile(filepath.Join(dir, "not-a-skill.txt"), []byte("hi"), 0644)

	r := NewLocal(dir)
	skills, err := r.List(context.Background())
	if err != nil {
		t.Fatalf("List() unexpected error: %v", err)
	}
	if len(skills) != 2 {
		t.Fatalf("List() returned %d skills, want 2", len(skills))
	}
}

func TestLocalRegistryRemove(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "to-remove")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "file.txt"), []byte("data"), 0644)

	r := NewLocal(dir)
	if err := r.Remove(context.Background(), "to-remove"); err != nil {
		t.Fatalf("Remove() unexpected error: %v", err)
	}

	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("skill directory should have been removed")
	}
}

func TestLocalRegistryInstallNotImplemented(t *testing.T) {
	r := NewLocal(t.TempDir())
	err := r.Install(context.Background(), "some-source")
	if err == nil {
		t.Error("Install() expected error, got nil")
	}
}
