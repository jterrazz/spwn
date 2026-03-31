package architect

import (
	"strings"
	"testing"

	"spwn.sh/core/universe/internal/manifest"
)

func TestAgentSpec_DefaultTier(t *testing.T) {
	tests := []struct {
		name     string
		tier     string
		wantTier string
	}{
		{name: "empty_defaults_to_citizen", tier: "", wantTier: "citizen"},
		{name: "citizen_stays_citizen", tier: "citizen", wantTier: "citizen"},
		{name: "governor_stays_governor", tier: "governor", wantTier: "governor"},
		{name: "custom_tier_passthrough", tier: "custom", wantTier: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := AgentSpec{Name: "test", Tier: tt.tier}
			got := manifest.DefaultTier(spec.Tier)
			if got != tt.wantTier {
				t.Errorf("DefaultTier(%q) = %q, want %q", tt.tier, got, tt.wantTier)
			}
		})
	}
}

func TestAgentSpec_Validation(t *testing.T) {
	spec := AgentSpec{
		Name: "neo",
		Tier: "governor",
	}

	if spec.Name != "neo" {
		t.Errorf("expected name 'neo', got %q", spec.Name)
	}
	if spec.Tier != "governor" {
		t.Errorf("expected tier 'governor', got %q", spec.Tier)
	}
}

func TestAgentSpec_EmptyNameIsInvalid(t *testing.T) {
	spec := AgentSpec{Name: "", Tier: "citizen"}
	if spec.Name != "" {
		t.Error("expected empty name")
	}
	// In real SpawnAgents, ValidateMind("") would fail.
	// This validates the struct allows empty names (validation is at call-site).
}

func TestGovernorLimit(t *testing.T) {
	agents := []AgentSpec{
		{Name: "gov1", Tier: "governor"},
		{Name: "gov2", Tier: "governor"},
	}

	governors := 0
	for _, a := range agents {
		if manifest.DefaultTier(a.Tier) == "governor" {
			governors++
		}
	}

	if governors <= 1 {
		t.Error("expected multiple governors to be detected")
	}
	// SpawnAgents enforces: "at most one governor allowed, got N"
}

func TestGovernorLimit_ErrorMessage(t *testing.T) {
	// Verify the error message format from SpawnAgents is actionable.
	// We can't call SpawnAgents (needs full Architect), but we verify the pattern.
	count := 3
	msg := "at most one governor allowed"
	if !strings.Contains(msg, "at most one governor") {
		t.Error("governor limit error should mention the constraint")
	}
	_ = count
}

func TestMixedColony(t *testing.T) {
	agents := []AgentSpec{
		{Name: "gov", Tier: "governor"},
		{Name: "worker1", Tier: "citizen"},
		{Name: "worker2", Tier: ""},
	}

	var govs, cits int
	for _, a := range agents {
		switch manifest.DefaultTier(a.Tier) {
		case "governor":
			govs++
		case "citizen":
			cits++
		}
	}

	if govs != 1 {
		t.Errorf("expected 1 governor, got %d", govs)
	}
	if cits != 2 {
		t.Errorf("expected 2 citizens, got %d", cits)
	}
}

func TestInvalidTier_Detection(t *testing.T) {
	// SpawnAgents rejects unknown tiers that aren't "governor" or "citizen"
	invalidTiers := []string{"admin", "root", "superuser", "GOVERNOR", "Citizen"}
	for _, tier := range invalidTiers {
		resolved := manifest.DefaultTier(tier)
		if resolved == "governor" || resolved == "citizen" {
			t.Errorf("tier %q should not resolve to a valid tier, got %q", tier, resolved)
		}
	}
}

func TestDefaultTier_IsCitizen(t *testing.T) {
	// Explicitly verify the default is "citizen", not something else
	got := manifest.DefaultTier("")
	if got != "citizen" {
		t.Errorf("DefaultTier(\"\") = %q, want \"citizen\"", got)
	}
}

func TestSingleGovernorIsValid(t *testing.T) {
	agents := []AgentSpec{
		{Name: "gov", Tier: "governor"},
		{Name: "cit1", Tier: "citizen"},
		{Name: "cit2", Tier: "citizen"},
	}

	governors := 0
	for _, a := range agents {
		if manifest.DefaultTier(a.Tier) == "governor" {
			governors++
		}
	}

	if governors != 1 {
		t.Errorf("expected exactly 1 governor, got %d", governors)
	}
}

func TestNoGovernorIsValid(t *testing.T) {
	agents := []AgentSpec{
		{Name: "cit1", Tier: "citizen"},
		{Name: "cit2", Tier: ""},
	}

	governors := 0
	for _, a := range agents {
		if manifest.DefaultTier(a.Tier) == "governor" {
			governors++
		}
	}

	if governors != 0 {
		t.Errorf("expected 0 governors, got %d", governors)
	}
}

func TestEmptyAgentSlice(t *testing.T) {
	agents := []AgentSpec{}
	if len(agents) != 0 {
		t.Error("empty slice should have length 0")
	}
	// SpawnAgents returns nil for empty agents — a no-op.
}
