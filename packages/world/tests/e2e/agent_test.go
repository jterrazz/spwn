//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestAgent_Init(t *testing.T) {
	// Given - a fresh test context
	// When - a new agent is initialized
	chain := setup.NewAgentBuilder(t).
		Init("fresh-agent")

	// Then - the Mind should have all standard layers plus SOUL.md at root
	// (identity collapsed into a single file; knowledge moved to worlds)
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("skills")
		m.HasLayer("playbooks")
		m.HasLayer("journal")
		m.HasFile("SOUL.md")
	})
}

func TestAgent_InitDuplicate(t *testing.T) {
	// Given - an agent that already exists
	a := setup.NewAgentBuilder(t)
	a.Init("dup-agent")

	// When - initializing an agent with the same name
	// Then - it should fail with "already exists"
	a.InitExpectError("dup-agent", "already exists")
}

func TestAgent_List(t *testing.T) {
	// Given - two initialized agents
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("agent-a")
	ctx.InitAgent("agent-b")

	// When - listing agents
	agents, err := agent.ListAgents()
	if err != nil {
		t.Fatalf("Failed to list agents: %v", err)
	}

	// Then - both agents should appear in the list
	if len(agents) != 2 {
		t.Fatalf("Expected 2 agents, got %d", len(agents))
	}

	names := map[string]bool{}
	for _, a := range agents {
		names[a.Name] = true
	}
	if !names["agent-a"] || !names["agent-b"] {
		t.Fatalf("Expected agents 'agent-a' and 'agent-b', got %v", names)
	}
}

func TestAgent_Inspect(t *testing.T) {
	// Given - an initialized agent
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("inspect-agent")

	// When - inspecting the agent
	info, err := agent.InspectAgent("inspect-agent")
	if err != nil {
		t.Fatalf("Failed to inspect agent: %v", err)
	}

	// Then - the name should match
	if info.Name != "inspect-agent" {
		t.Fatalf("Expected name 'inspect-agent', got %q", info.Name)
	}

	// AND the 3 standard layers should exist (identity collapsed into
	// SOUL.md at agent root; knowledge moved to world scope).
	for _, layer := range []string{"skills", "playbooks", "journal"} {
		if _, ok := info.Layers[layer]; !ok {
			t.Fatalf("Missing layer %q", layer)
		}
	}

	// AND SOUL.md should exist at the agent root.
	soulPath := filepath.Join(info.Path, "SOUL.md")
	if _, err := os.Stat(soulPath); err != nil {
		t.Fatalf("Expected SOUL.md at agent root: %v", err)
	}
}
