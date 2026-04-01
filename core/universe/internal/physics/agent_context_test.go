package physics

import (
	"strings"
	"testing"
)

func TestGenerateGovernorContext(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "morpheus",
		Tier:      "governor",
		WorldID:   "w-acme-28373",
		Workspace: "/workspace",
		Elements:  []string{"bash", "git", "node"},
		CPU:       2,
		Memory:    "4g",
		Timeout:   "30m",
		OtherAgents: []AgentInfo{
			{Name: "neo", Tier: "citizen"},
			{Name: "trinity", Tier: "citizen"},
		},
	})

	checks := map[string]string{
		"Governor":         "missing Governor role",
		"morpheus":         "missing agent name",
		"neo":              "missing citizen neo",
		"trinity":          "missing citizen trinity",
		"Messaging":        "missing messaging skill",
		"/world/inbox":     "missing inbox reference",
		"Delegation":       "missing delegation pattern",
		"w-acme-28373":     "missing world ID",
		"/workspace":       "missing workspace",
		"bash, git, node":  "missing elements",
		"2 cpu":            "missing CPU",
		"4g":               "missing memory",
		"30m":              "missing timeout",
	}

	for want, msg := range checks {
		if !strings.Contains(ctx, want) {
			t.Errorf("%s: %q not found in output:\n%s", msg, want, ctx)
		}
	}

	// Governor should NOT reference /mind/ layers
	if strings.Contains(ctx, "/mind/identity") {
		t.Error("governor should not reference mind layers")
	}
}

func TestGenerateCitizenContext(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "neo",
		Tier:      "citizen",
		WorldID:   "w-acme-28373",
		Workspace: "/workspace",
		Elements:  []string{"bash", "git"},
		CPU:       2,
		Memory:    "4g",
		Timeout:   "30m",
		Governor:  "morpheus",
		OtherAgents: []AgentInfo{
			{Name: "trinity", Tier: "citizen"},
		},
	})

	checks := map[string]string{
		"Citizen":        "missing Citizen role",
		"neo":            "missing name",
		"morpheus":       "missing governor",
		"trinity":        "missing peer",
		"/mind/identity/":         "missing mind identity",
		"/mind/skills/":           "missing mind skills",
		"/mind/memory/knowledge/": "missing mind knowledge",
		"/mind/memory/playbooks/": "missing mind playbooks",
		"/mind/memory/journal/":   "missing mind journal",
		"/world/inbox":   "missing inbox",
		"Messaging":      "missing messaging skill",
		"w-acme-28373":   "missing world ID",
		"/workspace":     "missing workspace",
		"bash, git":      "missing elements",
	}

	for want, msg := range checks {
		if !strings.Contains(ctx, want) {
			t.Errorf("%s: %q not found in output:\n%s", msg, want, ctx)
		}
	}
}

func TestGenerateNPCContext(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		Tier:     "npc",
		WorldID:  "w-acme-28373",
		NPCTask:  "lint src/",
		Elements: []string{"bash"},
	})

	checks := map[string]string{
		"NPC":          "missing NPC role",
		"lint src/":    "missing task",
		"w-acme-28373": "missing world ID",
		"bash":         "missing elements",
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

func TestGenerateGodContext_ContainsNewCLICommands(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "architect",
		Tier:      "god",
		WorldID:   "w-test-99999",
	})

	// New CLI commands that MUST appear in god-tier AGENT.md
	mustContain := []string{
		"spwn ls",
		"spwn down",
		"spwn agent new",
		"spwn agent ls",
		"spwn agent rm",
	}
	for _, cmd := range mustContain {
		if !strings.Contains(ctx, cmd) {
			t.Errorf("God-tier AGENT.md missing new command %q", cmd)
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
			t.Errorf("God-tier AGENT.md still contains old command %q", cmd)
		}
	}
}

func TestGenerateGodContext_ContainsAllSections(t *testing.T) {
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "architect",
		Tier:      "god",
		WorldID:   "w-test-99999",
		Elements:  []string{"bash", "git"},
		CPU:       4,
		Memory:    "8g",
		Timeout:   "60m",
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
			t.Errorf("God-tier AGENT.md missing section/command %q", s)
		}
	}
}

func TestGenerateCitizenContext_DefaultTier(t *testing.T) {
	// Empty tier should default to citizen
	ctx := GenerateAgentContext(AgentContextOpts{
		AgentName: "agent1",
		Tier:      "",
		WorldID:   "w-test-00001",
	})

	if !strings.Contains(ctx, "Citizen") {
		t.Error("empty tier should default to citizen")
	}
}
