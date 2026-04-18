//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestAgentLifecycle_SurvivesWorldDestruction(t *testing.T) {
	// Given - a world with an agent
	tc := setup.NewTestContext(t)
	tc.InitAgent("lifecycle-agent")

	chain := tc.Spawn().
		WithAgent("lifecycle-agent").
		Execute()

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasFile("SOUL.md")
	})

	// When - the world is destroyed
	chain.Destroy().
		ExpectState(func(s *setup.StateAssertion) {
			s.WorldCount(0)
		})

	// Then - the agent Mind should still exist on the host
	info, err := agent.InspectAgent("lifecycle-agent")
	if err != nil {
		t.Fatalf("Agent should survive world destruction: %v", err)
	}
	soulPath := filepath.Join(info.Path, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("Agent SOUL.md should still exist after world destruction: %v", err)
	}
}

func TestAgentLifecycle_SpawnInDifferentWorlds(t *testing.T) {
	// Given - an agent spawned in world A
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

	// When - world A is destroyed and the agent is spawned in world B
	chainA.Destroy()

	chainB := tc.Spawn().
		WithAgent("roaming-agent").
		Detached().
		Execute()

	worldBID := chainB.World().ID

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// Then - the world IDs should differ
	if worldAID == worldBID {
		t.Fatalf("Expected different world IDs, both are %s", worldAID)
	}

	// AND the agent Mind should persist across both
	info, err := agent.InspectAgent("roaming-agent")
	if err != nil {
		t.Fatalf("Agent inspect failed: %v", err)
	}
	soulPath := filepath.Join(info.Path, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("Agent should retain its SOUL.md after spanning multiple worlds: %v", err)
	}
}

func TestAgentLifecycle_JournalAcrossWorlds(t *testing.T) {
	// Given - an agent that runs to completion in a first world
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

	// When - the agent runs to completion in a second world
	chain2 := tc.Spawn().
		WithAgent("journal-multi-agent").
		RunAgent().
		Execute()

	// Then - the journal should have entries from both worlds
	chain2.ExpectJournal(func(j *setup.JournalAssertion) {
		j.HasEntries(2)
		j.LatestWorldID(chain2.World().ID)
	})

	// AND the entries should reference different worlds
	mindPath := agent.AgentDir("journal-multi-agent")
	entries, err := agent.ListJournal(mindPath, 0)
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
	// Given - an agent with a custom playbook file (knowledge is
	// world-scoped now, so it's no longer part of the agent's Mind
	// export).
	tc := setup.NewTestContext(t)
	tc.InitAgent("export-src-agent")

	playbooksPath := filepath.Join(agent.AgentDir("export-src-agent"), "playbooks")
	os.MkdirAll(playbooksPath, 0755)
	os.WriteFile(filepath.Join(playbooksPath, "custom.md"), []byte("# Custom Playbook\nThis is unique."), 0644)

	// When - the agent is exported and imported into a new agent
	outputDir := t.TempDir()
	archivePath, err := agent.ExportMind("export-src-agent", outputDir, nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	err = agent.ImportMind("export-dst-agent", archivePath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Then - the imported agent should have the same layer structure
	srcInfo, err := agent.InspectAgent("export-src-agent")
	if err != nil {
		t.Fatalf("Inspect source failed: %v", err)
	}
	dstInfo, err := agent.InspectAgent("export-dst-agent")
	if err != nil {
		t.Fatalf("Inspect destination failed: %v", err)
	}

	for layer := range srcInfo.Layers {
		if _, ok := dstInfo.Layers[layer]; !ok {
			t.Fatalf("Imported agent missing layer %q", layer)
		}
	}

	// AND the custom playbook file should be preserved
	customPath := filepath.Join(agent.AgentDir("export-dst-agent"), "playbooks", "custom.md")
	content, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("Custom playbook file not found in imported agent: %v", err)
	}
	if string(content) != "# Custom Playbook\nThis is unique." {
		t.Fatalf("Custom playbook content mismatch: %q", string(content))
	}
}

func TestAgentLifecycle_CustomCoreFile(t *testing.T) {
	// Given - an agent with a customized SOUL.md (identity is a file
	// at the agent root now, not a directory; use a skill file for
	// "extra personality" content)
	tc := setup.NewTestContext(t)
	tc.InitAgent("profile-agent")

	skillsDir := filepath.Join(agent.AgentDir("profile-agent"), "skills")
	os.MkdirAll(skillsDir, 0755)
	os.WriteFile(filepath.Join(skillsDir, "custom.md"), []byte("# Custom Skill\nYou are a specialist."), 0644)

	// When - the agent is spawned in a world
	chain := tc.Spawn().
		WithAgent("profile-agent").
		Detached().
		Execute()

	// Then - the mock should see the Mind with the custom skill
	chain.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
		m.SawMind()
	})

	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasFile("skills/custom.md")
	})
}

func TestAgentLifecycle_SessionDiffersPerWorld(t *testing.T) {
	// Given - an agent spawned in world A
	tc := setup.NewTestContext(t)
	tc.InitAgent("session-diff-agent")

	chainA := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainA.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// When - world A is destroyed and the agent is spawned in world B
	chainA.Destroy()

	chainB := tc.Spawn().
		WithAgent("session-diff-agent").
		Detached().
		Execute()

	chainB.ExpectMock(func(m *setup.MockAssertion) {
		m.WasCalled()
	})

	// Then - the deterministic session ID (world ID + agent name) should differ.
	sessionA := agent.DeterministicSessionID("session-diff-agent", chainA.World().ID)
	sessionB := agent.DeterministicSessionID("session-diff-agent", chainB.World().ID)

	if sessionA == sessionB {
		t.Fatalf("Expected different session IDs for different worlds, both are %q", sessionA)
	}
}
