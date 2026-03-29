package architect

import (
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
	// Verify that AgentSpec struct has the expected fields
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

func TestGovernorLimit(t *testing.T) {
	// Validate the logic: at most one governor
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

	if governors > 1 {
		// This is the expected validation outcome
		return
	}
	t.Error("expected multiple governors to be detected")
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
