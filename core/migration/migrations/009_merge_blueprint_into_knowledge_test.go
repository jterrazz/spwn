package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMergeBlueprintIntoKnowledge(t *testing.T) {
	dir := t.TempDir()

	// Create blueprint with files
	bp := filepath.Join(dir, "blueprint")
	os.MkdirAll(filepath.Join(bp, "agents"), 0755)
	os.WriteFile(filepath.Join(bp, "overview.md"), []byte("# Overview"), 0644)
	os.WriteFile(filepath.Join(bp, "agents", "qa.md"), []byte("# QA Agent"), 0644)

	// Create knowledge with one overlapping file
	kn := filepath.Join(dir, "knowledge")
	os.MkdirAll(kn, 0755)
	os.WriteFile(filepath.Join(kn, "overview.md"), []byte("# Existing Overview"), 0644)

	if err := MergeBlueprintIntoKnowledge.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	// blueprint/ should be gone
	if _, err := os.Stat(bp); !os.IsNotExist(err) {
		t.Error("blueprint/ should have been removed")
	}

	// overview.md should NOT be overwritten (existing wins)
	data, _ := os.ReadFile(filepath.Join(kn, "overview.md"))
	if string(data) != "# Existing Overview" {
		t.Errorf("existing file was overwritten: %s", data)
	}

	// agents/qa.md should have been copied
	data, err := os.ReadFile(filepath.Join(kn, "agents", "qa.md"))
	if err != nil {
		t.Fatalf("agents/qa.md missing: %v", err)
	}
	if string(data) != "# QA Agent" {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestMergeBlueprintIntoKnowledge_NoBlueprintDir(t *testing.T) {
	dir := t.TempDir()
	if err := MergeBlueprintIntoKnowledge.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
}

func TestMergeBlueprintIntoKnowledge_Idempotent(t *testing.T) {
	dir := t.TempDir()
	bp := filepath.Join(dir, "blueprint")
	os.MkdirAll(bp, 0755)
	os.WriteFile(filepath.Join(bp, "readme.md"), []byte("hello"), 0644)

	MergeBlueprintIntoKnowledge.Apply(context.Background(), dir)
	// Second run — blueprint/ already gone
	if err := MergeBlueprintIntoKnowledge.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, "knowledge", "readme.md"))
	if string(data) != "hello" {
		t.Errorf("unexpected: %s", data)
	}
}
