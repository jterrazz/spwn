//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/mind"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestAgentLifecycle_SurvivesWorldDestruction(t *testing.T) {
	// GIVEN a world with an agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("lifecycle-agent")

	chain := tc.Spawn().
		WithAgent("lifecycle-agent").
		Execute()

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("core")
		m.HasFile("core/profile.md")
	})

	// WHEN the world is destroyed
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// THEN the agent Mind should still exist on the host
	info, err := mind.InspectAgent("lifecycle-agent")
	if err != nil {
		t.Fatalf("Agent should survive world destruction: %v", err)
	}
	if _, ok := info.Layers["core"]; !ok {
		t.Fatal("Agent Mind should still have core layer after world destruction")
	}
}

func TestAgentLifecycle_SpawnInDifferentWorlds(t *testing.T) {
	// GIVEN an agent spawned in world A
	tc := setup.NewTestContext(t)
	tc.InitAgent("roaming-agent")

	chainA := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	worldAID := chainA.World().ID

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// WHEN world A is destroyed and the agent is spawned in world B
	chainA.Destroy()

	chainB := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	worldBID := chainB.World().ID

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// THEN the world IDs should differ
	if worldAID == worldBID {
		t.Fatalf("Expected different world IDs, both are %s", worldAID)
	}

	// AND the agent Mind should persist across both
	info, err := mind.InspectAgent("roaming-agent")
	if err != nil {
		t.Fatalf("Agent inspect failed: %v", err)
	}
	if _, ok := info.Layers["core"]; !ok {
		t.Fatal("Agent should retain Mind layers after spanning multiple worlds (core layer check)")
	}
}

func TestAgentLifecycle_JournalAcrossWorlds(t *testing.T) {
	// GIVEN an agent that runs to completion in a first world
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-multi-agent")

	chain1 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	chain1.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(1)
		j.LatestWorldID(chain1.World().ID)
	})

	// WHEN the agent runs to completion in a second world
	chain2 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	// THEN the journal should have entries from both worlds
	chain2.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(2)
		j.LatestWorldID(chain2.World().ID)
	})

	// AND the entries should reference different worlds
	mindPath := mind.AgentDir("journal-multi-agent")
	entries, err := mind.ListJournal(mindPath, 0)
	if err != nil {
		t.Fatalf("Failed to list journal: %v", err)
	}
	if len(entries) < 2 {
		t.Fatalf("Expected at least 2 journal entries, got %d", len(entries))
	}

	worldIDs := make(map[string]bool)
	for _, entry := range entries {
		worldIDs[entry.WorldID] = true
	}
	if len(worldIDs) < 2 {
		t.Fatalf("Expected journal entries from at least 2 worlds, got %d", len(worldIDs))
	}
}

func TestAgentLifecycle_ExportImportMindIdentical(t *testing.T) {
	// GIVEN an agent with a custom knowledge file
	tc := setup.NewTestContext(t)
	tc.InitAgent("export-src-agent")

	knowledgePath := filepath.Join(mind.AgentDir("export-src-agent"), "knowledge")
	os.MkdirAll(knowledgePath, 0755)
	os.WriteFile(filepath.Join(knowledgePath, "custom.md"), []byte("# Custom Knowledge\nThis is unique."), 0644)

	// WHEN the agent is exported and imported into a new agent
	outputDir := t.TempDir()
	archivePath, err := mind.ExportMind("export-src-agent", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	err = mind.ImportMind("export-dst-agent", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// THEN the imported agent should have the same layer structure
	srcInfo, err := mind.InspectAgent("export-src-agent")
	if err != nil {
		t.Fatalf("Inspect source failed: %v", err)
	}
	dstInfo, err := mind.InspectAgent("export-dst-agent")
	if err != nil {
		t.Fatalf("Inspect destination failed: %v", err)
	}

	for layer := range srcInfo.Layers {
		if _, ok := dstInfo.Layers[layer]; !ok {
			t.Fatalf("Imported agent missing layer %q", layer)
		}
	}

	// AND the custom knowledge file should be preserved
	customPath := filepath.Join(mind.AgentDir("export-dst-agent"), "knowledge", "custom.md")
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Custom knowledge file not found in imported agent: %v", err)
	}
	if string(content) != "# Custom Knowledge\nThis is unique." {
		t.Fatalf("Custom knowledge content mismatch: %q", string(content))
	}
}

func TestAgentLifecycle_CustomCoreFile(t *testing.T) {
	// GIVEN an agent with a custom file in the core layer
	tc := setup.NewTestContext(t)
	tc.InitAgent("profile-agent")

	coreDir := filepath.Join(mind.AgentDir("profile-agent"), "core")
	os.WriteFile(filepath.Join(coreDir, "custom.md"), []byte("# Custom Profile\nYou are a specialist."), 0644)

	// WHEN the agent is spawned in a world
	chain := tc.Spawn().
		WithAgent("profile-agent").
		Detached().
		Execute()

	// THEN the mock should see the Mind with the custom profile
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.SawMind()
	})

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasFile("core/custom.md")
	})
}

func TestAgentLifecycle_SessionDiffersPerWorld(t *testing.T) {
	// GIVEN an agent spawned in world A
	tc := setup.NewTestContext(t)
	tc.InitAgent("session-diff-agent")

	chainA := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// WHEN world A is destroyed and the agent is spawned in world B
	chainA.Destroy()

	chainB := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// THEN the deterministic session ID (world ID + agent name) should differ.
	sessionA := mind.DeterministicSessionID("session-diff-agent", chainA.World().ID)
	sessionB := mind.DeterministicSessionID("session-diff-agent", chainB.World().ID)

	if sessionA == sessionB {
		t.Fatalf("Expected different session IDs for different worlds, both are %q", sessionA)
	}
}
