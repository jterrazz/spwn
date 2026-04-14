//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestAgent_ExportCreatesArchive(t *testing.T) {
	// Given - an initialized agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("export-agent")

	// When - the agent's Mind is exported
	outputDir := t.TempDir()
	archivePath, err := agent.ExportMind("export-agent", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Then - the archive should exist and be non-empty
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("Archive not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("Expected non-empty archive")
	}
}

func TestAgent_ImportRestoresMind(t *testing.T) {
	// Given - an exported agent archive
	tc := setup.NewTestContext(t)
	tc.InitAgent("import-src")

	outputDir := t.TempDir()
	archivePath, err := agent.ExportMind("import-src", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// When - importing the archive into a new agent
	err = agent.ImportMind("import-dst", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Then - the imported agent should have core/profile.md
	info, err := agent.InspectAgent("import-dst")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	files := info.Layers["core"]
	found := false
	for _, f := range files {
		if f == "profile.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Expected core/profile.md in imported Mind, got: %v", files)
	}
}

func TestAgent_ExportWithExclude(t *testing.T) {
	// Given - an agent with a journal entry
	tc := setup.NewTestContext(t)
	tc.InitAgent("exclude-agent")

	journalDir := filepath.Join(agent.AgentDir("exclude-agent"), "journal")
	os.WriteFile(filepath.Join(journalDir, "test.md"), []byte("test"), 0644)

	// When - exporting with the journal excluded
	outputDir := t.TempDir()
	archivePath, err := agent.ExportMind("exclude-agent", outputDir, []string{"journal"})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// AND importing into a new agent
	err = agent.ImportMind("exclude-dst", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Then - the imported agent's journal should be empty
	info, err := agent.InspectAgent("exclude-dst")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if len(info.Layers["journal"]) > 0 {
		t.Fatalf("Expected empty journal after exclude, got: %v", info.Layers["journal"])
	}
}
