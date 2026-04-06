//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/universe/tests/e2e/setup"
)

func TestAgentLifecycle_SurvivesUniverseDestruction(t *testing.T) {
	// GIVEN a universe with an agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("lifecycle-agent")

	chain := tc.Spawn().
		WithAgent("lifecycle-agent").
		Execute()

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("core")
		m.HasFile("core/default.md")
	})

	// WHEN the universe is destroyed
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// THEN the agent Mind should still exist on the host
	info, err := agentDomain.InspectAgent("lifecycle-agent")
	if err != nil {
		t.Fatalf("Agent should survive world destruction: %v", err)
	}
	if _, ok := info.Layers["core"]; !ok {
		t.Fatal("Agent Mind should still have core layer after world destruction")
	}
}

func TestAgentLifecycle_SpawnInDifferentUniverses(t *testing.T) {
	// GIVEN an agent spawned in world A
	tc := setup.NewTestContext(t)
	tc.InitAgent("roaming-agent")

	chainA := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	universeAID := chainA.Universe().ID

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// WHEN world A is destroyed and the agent is spawned in world B
	chainA.Destroy()

	chainB := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	universeBID := chainB.Universe().ID

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// THEN the universe IDs should differ
	if universeAID == universeBID {
		t.Fatalf("Expected different world IDs, both are %s", universeAID)
	}

	// AND the agent Mind should persist across both
	info, err := agentDomain.InspectAgent("roaming-agent")
	if err != nil {
		t.Fatalf("Agent inspect failed: %v", err)
	}
	if _, ok := info.Layers["core"]; !ok {
		t.Fatal("Agent should retain Mind layers after spanning multiple worlds (core layer check)")
	}
}

func TestAgentLifecycle_JournalAcrossUniverses(t *testing.T) {
	// GIVEN an agent that runs to completion in a first world
	tc := setup.NewTestContext(t)
	tc.InitAgent("journal-multi-agent")

	chain1 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	chain1.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(1)
		j.LatestWorldID(chain1.Universe().ID)
	})

	// WHEN the agent runs to completion in a second world
	chain2 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	// THEN the journal should have entries from both worlds
	chain2.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(2)
		j.LatestWorldID(chain2.Universe().ID)
	})

	// AND the entries should reference different worlds
	mindPath := agentDomain.AgentDir("journal-multi-agent")
	entries, err := agentDomain.ListJournal(mindPath, 0)
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

	knowledgePath := filepath.Join(agentDomain.AgentDir("export-src-agent"), "knowledge")
	os.MkdirAll(knowledgePath, 0755)
	os.WriteFile(filepath.Join(knowledgePath, "custom.md"), []byte("# Custom Knowledge\nThis is unique."), 0644)

	// WHEN the agent is exported and imported into a new agent
	outputDir := t.TempDir()
	archivePath, err := agentDomain.ExportMind("export-src-agent", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	err = agentDomain.ImportMind("export-dst-agent", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// THEN the imported agent should have the same layer structure
	srcInfo, err := agentDomain.InspectAgent("export-src-agent")
	if err != nil {
		t.Fatalf("Inspect source failed: %v", err)
	}
	dstInfo, err := agentDomain.InspectAgent("export-dst-agent")
	if err != nil {
		t.Fatalf("Inspect destination failed: %v", err)
	}

	for layer := range srcInfo.Layers {
		if _, ok := dstInfo.Layers[layer]; !ok {
			t.Fatalf("Imported agent missing layer %q", layer)
		}
	}

	// AND the custom knowledge file should be preserved
	customPath := filepath.Join(agentDomain.AgentDir("export-dst-agent"), "knowledge", "custom.md")
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Custom knowledge file not found in imported agent: %v", err)
	}
	if string(content) != "# Custom Knowledge\nThis is unique." {
		t.Fatalf("Custom knowledge content mismatch: %q", string(content))
	}
}

func TestAgentLifecycle_CustomIdentityFile(t *testing.T) {
	// GIVEN an agent with a custom persona file in the identity layer
	tc := setup.NewTestContext(t)
	tc.InitAgent("persona-agent")

	identityDir := filepath.Join(agentDomain.AgentDir("persona-agent"), "core")
	os.WriteFile(filepath.Join(identityDir, "custom.md"), []byte("# Custom Persona\nYou are a specialist."), 0644)

	// WHEN the agent is spawned in a universe
	chain := tc.Spawn().
		WithAgent("persona-agent").
		Detached().
		Execute()

	// THEN the mock should see the Mind with the custom persona
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.SawMind()
	})

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasFile("identity/custom.md")
	})
}

func TestAgentLifecycle_SessionDiffersPerUniverse(t *testing.T) {
	// GIVEN an agent spawned in world A
	tc := setup.NewTestContext(t)
	tc.InitAgent("session-diff-agent")

	chainA := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// WHEN world A is destroyed and the agent is spawned in world B
	chainA.Destroy()

	chainB := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.HasSessionID()
	})

	// THEN the session IDs should be deterministic but different per universe
	sessionA := agentDomain.DeterministicSessionID("session-diff-agent", chainA.Universe().ID)
	sessionB := agentDomain.DeterministicSessionID("session-diff-agent", chainB.Universe().ID)

	if sessionA == sessionB {
		t.Fatalf("Expected different session IDs for different worlds, both are %q", sessionA)
	}
}
