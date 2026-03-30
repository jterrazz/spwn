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
	if strings.Contains(ctx, "/mind/personas") {
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
		"/mind/personas": "missing mind personas",
		"/mind/skills":   "missing mind skills",
		"/mind/knowledge":"missing mind knowledge",
		"/mind/playbooks":"missing mind playbooks",
		"/mind/journal":  "missing mind journal",
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
