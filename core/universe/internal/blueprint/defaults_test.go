package blueprint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitBlueprint_CreatesDefaults(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check default files exist
	for relPath := range DefaultFiles {
		absPath := filepath.Join(basePath, relPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Errorf("expected default file %s to exist", relPath)
		}
	}

	// Check default directories exist
	for _, dirName := range DefaultDirs {
		dirPath := filepath.Join(basePath, dirName)
		info, err := os.Stat(dirPath)
		if os.IsNotExist(err) {
			t.Errorf("expected default dir %s to exist", dirName)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dirName)
		}
	}
}

func TestInitBlueprint_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	// First init
	if err := Init(basePath); err != nil {
		t.Fatalf("first Init failed: %v", err)
	}

	// Overwrite overview.md with custom content
	customContent := "# Custom Overview"
	overviewPath := filepath.Join(basePath, "overview.md")
	if err := os.WriteFile(overviewPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write custom overview: %v", err)
	}

	// Second init should NOT overwrite
	if err := Init(basePath); err != nil {
		t.Fatalf("second Init failed: %v", err)
	}

	data, err := os.ReadFile(overviewPath)
	if err != nil {
		t.Fatalf("failed to read overview.md: %v", err)
	}
	if string(data) != customContent {
		t.Errorf("Init overwrote existing file: got %q, want %q", string(data), customContent)
	}
}

func TestDefaultFilesContent(t *testing.T) {
	// Verify each default file has non-empty content
	for relPath, content := range DefaultFiles {
		if content == "" {
			t.Errorf("default file %s has empty content", relPath)
		}
	}

	// Check specific expected content
	overview, ok := DefaultFiles["overview.md"]
	if !ok {
		t.Fatal("overview.md not in DefaultFiles")
	}
	if len(overview) < 10 {
		t.Error("overview.md content is too short")
	}

	glossary, ok := DefaultFiles["glossary.md"]
	if !ok {
		t.Fatal("glossary.md not in DefaultFiles")
	}
	if len(glossary) < 10 {
		t.Error("glossary.md content is too short")
	}

	roadmap, ok := DefaultFiles["roadmap.md"]
	if !ok {
		t.Fatal("roadmap.md not in DefaultFiles")
	}
	if len(roadmap) < 10 {
		t.Error("roadmap.md content is too short")
	}
}

func TestListFiles(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	files, err := ListFiles(basePath)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(files) < len(DefaultFiles) {
		t.Errorf("expected at least %d files, got %d", len(DefaultFiles), len(files))
	}

	// Check overview.md is in the list
	found := false
	for _, f := range files {
		if f.Path == "overview.md" {
			found = true
			break
		}
	}
	if !found {
		t.Error("overview.md not found in ListFiles result")
	}
}

func TestReadFile(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	content, err := ReadFile(basePath, "overview.md")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if content == "" {
		t.Error("ReadFile returned empty content for overview.md")
	}
}

func TestReadFile_TraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	_, err := ReadFile(basePath, "../../../etc/passwd")
	if err == nil {
		t.Error("expected error for directory traversal, got nil")
	}
}

func TestSearch(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	results, err := Search(basePath, "Blueprint")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("expected search results for 'Blueprint', got none")
	}
}
