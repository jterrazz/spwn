//go:build e2e

package e2e

import (
	"testing"

	"github.com/jterrazz/universe/__tests__/e2e/setup"
	"github.com/jterrazz/universe/internal/mind"
)

func TestAgent_Init(t *testing.T) {
	setup.NewAgentBuilder(t).
		Init("fresh-agent").
		ExpectMind(func(m *setup.MindAssertion) {
			m.HasLayer("personas")
			m.HasLayer("skills")
			m.HasLayer("knowledge")
			m.HasLayer("playbooks")
			m.HasLayer("journal")
			m.HasLayer("sessions")
			m.HasFile("personas/default.md")
		})
}

func TestAgent_InitDuplicate(t *testing.T) {
	a := setup.NewAgentBuilder(t)
	a.Init("dup-agent")
	a.InitExpectError("dup-agent", "already exists")
}

func TestAgent_List(t *testing.T) {
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("agent-a")
	ctx.InitAgent("agent-b")

	agents, err := mind.List()
	if err != nil {
		t.Fatalf("Failed to list agents: %v", err)
	}

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
	ctx := setup.NewTestContext(t)
	ctx.InitAgent("inspect-agent")

	info, err := mind.Inspect("inspect-agent")
	if err != nil {
		t.Fatalf("Failed to inspect agent: %v", err)
	}

	if info.Name != "inspect-agent" {
		t.Fatalf("Expected name 'inspect-agent', got %q", info.Name)
	}

	// Should have all 6 layers
	for _, layer := range []string{"personas", "skills", "knowledge", "playbooks", "journal", "sessions"} {
		if _, ok := info.Layers[layer]; !ok {
			t.Fatalf("Missing layer %q", layer)
		}
	}

	// Personas should have default.md
	if files, ok := info.Layers["personas"]; ok {
		found := false
		for _, f := range files {
			if f == "default.md" {
				found = true
			}
		}
		if !found {
			t.Fatal("Expected default.md in personas layer")
		}
	}
}
