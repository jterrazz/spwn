package physics

import (
	"strings"
	"testing"
)

func TestAgentsBookContent(t *testing.T) {
	// Verify that AGENT.md content for a citizen has key sections
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "test-agent",
		Tier:      "citizen",
		WorldID:   "w-test-12345",
		Workspace: "/workspace",
		Elements:  []string{"bash", "git"},
		CPU:       2,
		Memory:    "4g",
		Timeout:   "30m",
	})

	// Must contain key sections that make up the "Agent Operating Manual"
	keySections := []string{
		"Your Role",
		"Your Mind",
		"Your World",
		"Messaging",
	}

	for _, section := range keySections {
		if !strings.Contains(ctx, section) {
			t.Errorf("AGENT.md (citizen) missing key section %q", section)
		}
	}

	// Must contain identity and skills references
	mindPaths := []string{
		"/mind/identity/",
		"/mind/skills/",
		"/mind/memory/knowledge/",
		"/mind/memory/playbooks/",
		"/mind/memory/journal/",
	}

	for _, path := range mindPaths {
		if !strings.Contains(ctx, path) {
			t.Errorf("AGENT.md (citizen) missing mind path %q", path)
		}
	}
}

func TestSystemSkillsExist(t *testing.T) {
	// Verify that the system generates content for all 4 skill contexts
	// (mind management, messaging, workspace, journal) in citizen context
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "neo",
		Tier:      "citizen",
		WorldID:   "w-test-99999",
		Workspace: "/workspace",
		Elements:  []string{"bash"},
		Governor:  "morpheus",
		OtherAgents: []AgentInfo{
			{Name: "trinity", Tier: "citizen"},
		},
	})

	// System skills are embedded in the AGENT.md for citizens
	// 1. Mind management skill
	if !strings.Contains(ctx, "/mind/") {
		t.Error("citizen context missing mind management skill references")
	}

	// 2. Messaging / collaboration skill
	if !strings.Contains(ctx, "/world/inbox") {
		t.Error("citizen context missing messaging/collaboration skill")
	}

	// 3. World awareness skill (elements, workspace)
	if !strings.Contains(ctx, "/workspace") {
		t.Error("citizen context missing workspace/world awareness")
	}

	// 4. Other agents awareness
	if !strings.Contains(ctx, "trinity") {
		t.Error("citizen context missing other agents awareness")
	}
}

func TestArchitectSkillsExist(t *testing.T) {
	// Verify that the architect (god tier) has all 3 key skill sections
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "architect",
		Tier:      "god",
		WorldID:   "w-god-00001",
		Elements:  []string{"bash", "git", "docker"},
		CPU:       4,
		Memory:    "8g",
		Timeout:   "60m",
	})

	// 1. World Management skill
	if !strings.Contains(ctx, "World Management") {
		t.Error("architect context missing World Management skill")
	}

	// 2. Agent Management skill
	if !strings.Contains(ctx, "Agent Management") {
		t.Error("architect context missing Agent Management skill")
	}

	// 3. Messaging skill
	if !strings.Contains(ctx, "Messaging") {
		t.Error("architect context missing Messaging skill")
	}

	// Verify key commands are present
	requiredCommands := []string{
		"spwn ls",
		"spwn up",
		"spwn down",
		"spwn agent new",
		"spwn agent ls",
		"spwn agent talk",
		"spwn agent send",
		"spwn agent inbox",
		"spwn status",
	}

	for _, cmd := range requiredCommands {
		if !strings.Contains(ctx, cmd) {
			t.Errorf("architect context missing command %q", cmd)
		}
	}
}
