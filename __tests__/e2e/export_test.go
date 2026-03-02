//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/mind"
)

func TestAgent_ExportCreatesArchive(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("export-agent")

	outputDir := t.TempDir()
	archivePath, err := mind.Export("export-agent", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("Archive not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("Expected non-empty archive")
	}
}

func TestAgent_ImportRestoresMind(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("import-src")

	// Export
	outputDir := t.TempDir()
	archivePath, err := mind.Export("import-src", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import into new agent
	err = mind.Import("import-dst", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify imported agent has the same structure
	info, err := mind.Inspect("import-dst")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	// Should have personas/default.md
	files := info.Layers["personas"]
	found := false
	for _, f := range files {
		if f == "default.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected personas/default.md in imported Mind, got: %v", files)
	}
}

func TestAgent_ExportWithExclude(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("exclude-agent")

	// Write a file to journal to ensure it exists
	journalDir := filepath.Join(mind.AgentDir("exclude-agent"), "journal")
	os.WriteFile(filepath.Join(journalDir, "test.md"), []byte("test"), 0644)

	outputDir := t.TempDir()
	archivePath, err := mind.Export("exclude-agent", outputDir, []string{"journal"})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import and verify journal was excluded
	err = mind.Import("exclude-dst", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	info, err := mind.Inspect("exclude-dst")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	// Journal should be empty (excluded from export)
	if len(info.Layers["journal"]) > 0 {
		t.Fatalf("Expected empty journal after exclude, got: %v", info.Layers["journal"])
	}
}
