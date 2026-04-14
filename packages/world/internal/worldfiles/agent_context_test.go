package worldfiles

import (
	"strings"
	"testing"

	"spwn.sh/packages/world/models"
)

func TestGenerateChiefContext(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "morpheus",
		Role:      "chief",
		WorldID:   "w-acme-28373",
		Workspaces: []models.Workspace{{Name: "default", Path: "/host/project"}},
		Tools:     []string{"bash", "git", "node"},
		OtherAgents: []AgentInfo{
			{Name: "neo", Role: "worker"},
			{Name: "trinity", Role: "worker"},
		},
	})

	checks := map[string]string{
		"Chief":            "missing Chief role",
		"morpheus":         "missing agent name",
		"neo":              "missing worker neo",
		"trinity":          "missing worker trinity",
		"Messaging":        "missing messaging skill",
		"/world/inbox":     "missing inbox reference",
		"Delegation":       "missing delegation pattern",
		"w-acme-28373":     "missing world ID",
		"/host/project":    "missing workspace path",
		"bash, git, node":  "missing tools",
	}

	for want, msg := range checks {
		if !strings.Contains(ctx, want) {
			t.Errorf("%s: %q not found in output:\n%s", msg, want, ctx)
		}
	}

	// Chief should NOT reference /mind/ layers
	if strings.Contains(ctx, "/mind/identity") {
		t.Error("chief should not reference mind layers")
	}
}

func TestGenerateWorkerContext(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "neo",
		Role:      "worker",
		WorldID:   "w-acme-28373",
		Workspaces: []models.Workspace{{Name: "default", Path: "/host/project"}},
		Tools:     []string{"bash", "git"},
		Chief:     "morpheus",
		OtherAgents: []AgentInfo{
			{Name: "trinity", Role: "worker"},
		},
	})

	checks := map[string]string{
		"Worker":         "missing Worker role",
		"neo":            "missing name",
		"morpheus":       "missing chief",
		"trinity":        "missing peer",
		"/mind/core/":      "missing mind core",
		"/mind/skills/":    "missing mind skills",
		"/mind/knowledge/": "missing mind knowledge",
		"/mind/playbooks/": "missing mind playbooks",
		"/mind/journal/":   "missing mind journal",
		"/world/inbox":   "missing inbox",
		"Messaging":      "missing messaging skill",
		"w-acme-28373":   "missing world ID",
		"/host/project":  "missing workspace path",
		"bash, git":      "missing tools",
	}

	for want, msg := range checks {
		if !strings.Contains(ctx, want) {
			t.Errorf("%s: %q not found in output:\n%s", msg, want, ctx)
		}
	}
}

func TestGenerateNPCContext(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		Role:    "npc",
		WorldID: "w-acme-28373",
		NPCTask: "lint src/",
		Tools:   []string{"bash"},
	})

	checks := map[string]string{
		"NPC":          "missing NPC role",
		"lint src/":    "missing task",
		"w-acme-28373": "missing world ID",
		"bash":         "missing tools",
	}

	for want, msg := range checks {
		if !strings.Contains(ctx, want) {
			t.Errorf("%s: %q not found in output:\n%s", msg, want, ctx)
		}
	}

	// NPC should NOT have messaging or mind
	if strings.Contains(ctx, "Messaging") {
		t.Error("NPC should not have messaging")
	}
	if strings.Contains(ctx, "/mind/") {
		t.Error("NPC should not reference mind")
	}
}

func TestGenerateArchitectContext_ContainsNewCLICommands(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "architect",
		Role:      "architect",
		WorldID:   "w-test-99999",
	})

	// New CLI commands that MUST appear in architect-tier AGENT.md
	mustContain := []string{
		"spwn ls",
		"spwn down",
		"spwn agent new",
		"spwn agent ls",
		"spwn agent rm",
	}
	for _, cmd := range mustContain {
		if !strings.Contains(ctx, cmd) {
			t.Errorf("Architect-tier AGENT.md missing new command %q", cmd)
		}
	}

	// Old CLI commands that must NOT appear
	mustNotContain := []string{
		"spwn world list",
		"spwn world destroy",
		"spwn agent init",
		"spwn agent list",
		"spwn agent delete",
	}
	for _, cmd := range mustNotContain {
		if strings.Contains(ctx, cmd) {
			t.Errorf("Architect-tier AGENT.md still contains old command %q", cmd)
		}
	}
}

func TestGenerateArchitectContext_ContainsAllSections(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "architect",
		Role:      "architect",
		WorldID:   "w-test-99999",
		Tools:     []string{"bash", "git"},
	})

	sections := []string{
		"Architect",
		"World Management",
		"Agent Management",
		"Messaging",
		"Status",
		"spwn up",
		"spwn inspect",
		"spwn logs",
		"spwn agent talk",
		"spwn agent inspect",
		"spwn agent send",
		"spwn agent inbox",
		"spwn agent watch",
		"spwn status",
	}
	for _, s := range sections {
		if !strings.Contains(ctx, s) {
			t.Errorf("Architect-tier AGENT.md missing section/command %q", s)
		}
	}
}

func TestGenerateWorkerContext_DefaultRole(t *testing.T) {
	// Empty role should default to worker
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "agent1",
		Role:      "",
		WorldID:   "w-test-00001",
	})

	if !strings.Contains(ctx, "Worker") {
		t.Error("empty role should default to worker")
	}
}
