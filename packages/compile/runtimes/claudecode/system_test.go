package claudecode

import (
	"strings"
	"testing"

	"spwn.sh/packages/world/models"
)

func TestAgentsBookContent(t *testing.T) {
	// Verify that AGENTS.md content for a worker has key sections
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "test-agent",
		Role:      "worker",
		WorldID:   "w-test-12345",
		Workspaces: []models.Workspace{{Name: "default", Path: "/workspace"}},
		Packages:  []string{"bash", "git"},
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
			t.Errorf("AGENTS.md (worker) missing key section %q", section)
		}
	}

	// Must contain core and skills references
	mindPaths := []string{
		"/mind/identity/",
		"/mind/skills/",
		"/mind/knowledge/",
		"/mind/playbooks/",
		"/mind/journal/",
	}

	for _, path := range mindPaths {
		if !strings.Contains(ctx, path) {
			t.Errorf("AGENTS.md (worker) missing mind path %q", path)
		}
	}
}

func TestSystemSkillsExist(t *testing.T) {
	// Verify that the system generates content for all 4 skill contexts
	// (mind management, messaging, workspace, journal) in worker context
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "neo",
		Role:      "worker",
		WorldID:   "w-test-99999",
		Workspaces: []models.Workspace{{Name: "default", Path: "/workspace"}},
		Packages:[]string{"bash"},
		Chief:     "morpheus",
		OtherAgents: []AgentInfo{
			{Name: "trinity", Role: "worker"},
		},
	})

	// System skills are embedded in the AGENTS.md for workers
	// 1. Mind management skill
	if !strings.Contains(ctx, "/mind/") {
		t.Error("worker context missing mind management skill references")
	}

	// 2. Messaging / collaboration skill
	if !strings.Contains(ctx, "/world/inbox") {
		t.Error("worker context missing messaging/collaboration skill")
	}

	// 3. World awareness skill (tools, workspace)
	if !strings.Contains(ctx, "/workspace") {
		t.Error("worker context missing workspace/world awareness")
	}

	// 4. Other agents awareness
	if !strings.Contains(ctx, "trinity") {
		t.Error("worker context missing other agents awareness")
	}
}

func TestArchitectSkillsExist(t *testing.T) {
	// Verify that the architect tier has all 3 key skill sections
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "architect",
		Role:      "architect",
		WorldID:   "spwn-world-architect-00001",
		Packages:  []string{"bash", "git", "docker"},
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
