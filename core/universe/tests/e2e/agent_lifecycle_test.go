//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	agentDomain "github.com/jterrazz/spwn/core/agent"
	"github.com/jterrazz/spwn/core/universe/tests/e2e/setup"
)

func TestAgentLifecycle_SurvivesUniverseDestruction(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("lifecycle-agent")

	chain := tc.Spawn().
		WithAgent("lifecycle-agent").
		Execute()

	// Verify agent is in the universe
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("personas")
		m.HasFile("personas/default.md")
	})

	// Destroy the universe
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.UniverseCount(0)
		})

	// Agent should still exist on host
	info, err := agentDomain.InspectAgent("lifecycle-agent")
	if err != nil {
		t.Fatalf("Agent should survive universe destruction: %v", err)
	}
	if _, ok := info.Layers["personas"]; !ok {
		t.Fatal("Agent Mind should still have personas layer after universe destruction")
	}
}

func TestAgentLifecycle_SpawnInDifferentUniverses(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("roaming-agent")

	// Spawn in universe A
	chainA := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	universeAID := chainA.Universe().ID

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// Destroy universe A
	chainA.Destroy()

	// Spawn in universe B
	chainB := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	universeBID := chainB.Universe().ID

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// Universe IDs must be different
	if universeAID == universeBID {
		t.Fatalf("Expected different universe IDs, both are %s", universeAID)
	}

	// Agent Mind should persist across both
	info, err := agentDomain.InspectAgent("roaming-agent")
	if err != nil {
		t.Fatalf("Agent inspect failed: %v", err)
	}
	if _, ok := info.Layers["personas"]; !ok {
		t.Fatal("Agent should retain Mind layers after spanning multiple universes")
	}
}

func TestAgentLifecycle_JournalAcrossUniverses(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-multi-agent")

	// First universe — RunAgent writes journal entry
	chain1 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	chain1.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(1)
		j.LatestUniverseID(chain1.Universe().ID)
	})

	// Small delay for timestamp ordering
	time.Sleep(100 * time.Millisecond)

	// Second universe — RunAgent writes another journal entry
	chain2 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	chain2.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(2)
		j.LatestUniverseID(chain2.Universe().ID)
	})

	// Verify both entries exist via direct API
	mindPath := agentDomain.AgentDir("journal-multi-agent")
	entries, err := agentDomain.ListJournal(mindPath, 0)
	if err != nil {
		t.Fatalf("Failed to list journal: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 journal entries, got %d", len(entries))
	}

	// Entries should reference different universes
	universeIDs := make(map[string]bool)
	for _, entry := range entries {
		universeIDs[entry.UniverseID] = true
	}
	if len(universeIDs) < 2 {
		t.Fatalf("Expected journal entries from at least 2 universes, got %d", len(universeIDs))
	}
}

func TestAgentLifecycle_ExportImportMindIdentical(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("export-src-agent")

	// Write a custom file into the agent's knowledge layer
	knowledgePath := filepath.Join(agentDomain.AgentDir("export-src-agent"), "knowledge")
	os.MkdirAll(knowledgePath, 0755)
	os.WriteFile(filepath.Join(knowledgePath, "custom.md"), []byte("# Custom Knowledge\nThis is unique."), 0644)

	// Export the agent
	outputDir := t.TempDir()
	archivePath, err := agentDomain.ExportMind("export-src-agent", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Import into a new agent
	err = agentDomain.ImportMind("export-dst-agent", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Verify imported agent has the same structure
	srcInfo, err := agentDomain.InspectAgent("export-src-agent")
	if err != nil {
		t.Fatalf("Inspect source failed: %v", err)
	}
	dstInfo, err := agentDomain.InspectAgent("export-dst-agent")
	if err != nil {
		t.Fatalf("Inspect destination failed: %v", err)
	}

	// Both should have the same layers
	for layer := range srcInfo.Layers {
		if _, ok := dstInfo.Layers[layer]; !ok {
			t.Fatalf("Imported agent missing layer %q", layer)
		}
	}

	// Verify the custom knowledge file was preserved
	customPath := filepath.Join(agentDomain.AgentDir("export-dst-agent"), "knowledge", "custom.md")
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Custom knowledge file not found in imported agent: %v", err)
	}
	if string(content) != "# Custom Knowledge\nThis is unique." {
		t.Fatalf("Custom knowledge content mismatch: %q", string(content))
	}
}

func TestAgentLifecycle_CustomPersonasFile(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("persona-agent")

	// Write a custom personas file
	personasDir := filepath.Join(agentDomain.AgentDir("persona-agent"), "personas")
	os.WriteFile(filepath.Join(personasDir, "custom.md"), []byte("# Custom Persona\nYou are a specialist."), 0644)

	// Spawn with agent and verify mock sees the Mind (which includes custom persona)
	tc.Spawn().
		WithAgent("persona-agent").
		Detached().
		Execute().
		ExpectMock(func(m *setup.MockAssertion) {
			m.WasCalled()
			m.SawMind()
		}).
		ExpectMind(func(m *setup.MindAssertion) {
			m.HasFile("personas/custom.md")
		})
}

func TestAgentLifecycle_SessionDiffersPerUniverse(t *testing.T) {
	tc := setup.NewTestContext(t)
	tc.InitAgent("session-diff-agent")

	// Spawn in universe A
	chainA := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// Destroy universe A
	chainA.Destroy()

	// Spawn in universe B
	chainB := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// Session IDs should be deterministic but different per universe
	sessionA := agentDomain.DeterministicSessionID("session-diff-agent", chainA.Universe().ID)
	sessionB := agentDomain.DeterministicSessionID("session-diff-agent", chainB.Universe().ID)

	if sessionA == sessionB {
		t.Fatalf("Expected different session IDs for different universes, both are %q", sessionA)
	}
}
