//go:build e2e

package e2e

import (
	"testing"

	"spwn.sh/packages/agent"
	"spwn.sh/packages/world/tests/e2e/setup"
)

func TestAgent_Init(t *testing.T) {
	// Given - a fresh test context
	// When - a new agent is initialized
	chain := setup.NewAgentBuilder(t).
		Init("fresh-agent")

	// Then - the Mind should have all standard layers and a default profile
	chain.ExpectMind(func(m *setup.MindAssertion) {
		m.HasLayer("core")
		m.HasLayer("skills")
		m.HasLayer("knowledge")
		m.HasLayer("playbooks")
		m.HasLayer("journal")
		m.HasFile("core/profile.md")
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

	// AND all 5 standard layers should exist
	for _, layer := range []string{"core", "skills", "knowledge", "playbooks", "journal"} {
		if _, ok := info.Layers[layer]; !ok {
			t.Fatalf("Missing layer %q", layer)
		}
	}

	// AND core should contain profile.md
	if files, ok := info.Layers["core"]; ok {
		found := false
		for _, f := range files {
			if f == "profile.md" {
				found = true
			}
		}
		if !found {
			t.Fatal("Expected profile.md in core layer")
		}
	}
}
