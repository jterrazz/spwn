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

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Write a new file
	content := "# Test File\n\nThis is test content.\n"
	if err := WriteFile(basePath, "test-write.md", content); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Read it back
	readBack, err := ReadFile(basePath, "test-write.md")
	if err != nil {
		t.Fatalf("ReadFile after write failed: %v", err)
	}

	if readBack != content {
		t.Errorf("content mismatch: got %q, want %q", readBack, content)
	}
}

func TestWriteFile_CreatesSubdirs(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Write to a nested path that doesn't exist yet
	content := "# Nested Architecture Doc\n\nDeep nested file.\n"
	if err := WriteFile(basePath, "projects/backend/architecture.md", content); err != nil {
		t.Fatalf("WriteFile to nested path failed: %v", err)
	}

	// Verify the file exists
	absPath := filepath.Join(basePath, "projects", "backend", "architecture.md")
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Error("expected nested file to exist after WriteFile")
	}

	// Read it back via the API
	readBack, err := ReadFile(basePath, "projects/backend/architecture.md")
	if err != nil {
		t.Fatalf("ReadFile for nested path failed: %v", err)
	}

	if readBack != content {
		t.Errorf("nested content mismatch: got %q, want %q", readBack, content)
	}
}

func TestWriteFile_TraversalBlocked(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	err := WriteFile(basePath, "../../../etc/evil", "malicious content")
	if err == nil {
		t.Error("expected error for directory traversal in WriteFile, got nil")
	}
}

func TestSearch_MultipleResults(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Write files with a common search term
	if err := WriteFile(basePath, "auth.md", "# Authentication\n\nJWT token validation flow.\n"); err != nil {
		t.Fatalf("WriteFile auth.md failed: %v", err)
	}
	if err := WriteFile(basePath, "security.md", "# Security\n\nAuthentication and authorization.\n"); err != nil {
		t.Fatalf("WriteFile security.md failed: %v", err)
	}
	if err := WriteFile(basePath, "performance.md", "# Performance\n\nCaching strategies.\n"); err != nil {
		t.Fatalf("WriteFile performance.md failed: %v", err)
	}

	results, err := Search(basePath, "authentication")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Should find matches in auth.md and security.md but not performance.md
	if len(results) < 2 {
		t.Errorf("expected at least 2 files with matches, got %d", len(results))
	}

	if _, ok := results["auth.md"]; !ok {
		t.Error("expected search results to include auth.md")
	}
	if _, ok := results["security.md"]; !ok {
		t.Error("expected search results to include security.md")
	}
	if _, ok := results["performance.md"]; ok {
		t.Error("expected search results to NOT include performance.md")
	}
}

func TestListFiles_WithSubdirs(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "blueprint")

	if err := Init(basePath); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Create nested files
	if err := WriteFile(basePath, "projects/api.md", "# API"); err != nil {
		t.Fatalf("WriteFile projects/api.md failed: %v", err)
	}
	if err := WriteFile(basePath, "projects/web/frontend.md", "# Frontend"); err != nil {
		t.Fatalf("WriteFile projects/web/frontend.md failed: %v", err)
	}
	if err := WriteFile(basePath, "decisions/adr-001.md", "# ADR 001"); err != nil {
		t.Fatalf("WriteFile decisions/adr-001.md failed: %v", err)
	}

	files, err := ListFiles(basePath)
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	// Build a set of file paths for easy lookup
	pathSet := make(map[string]bool)
	for _, f := range files {
		pathSet[f.Path] = true
	}

	// Verify nested files appear in the listing
	expectedPaths := []string{
		"projects/api.md",
		"projects/web/frontend.md",
		"decisions/adr-001.md",
		"overview.md", // from Init defaults
	}

	for _, expected := range expectedPaths {
		if !pathSet[expected] {
			t.Errorf("expected ListFiles to include %q, got paths: %v", expected, pathSet)
		}
	}

	// Verify total count is reasonable (defaults + custom files)
	if len(files) < len(expectedPaths) {
		t.Errorf("expected at least %d files, got %d", len(expectedPaths), len(files))
	}
}
