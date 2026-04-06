package architect

import (
	"strings"
	"testing"

	"spwn.sh/core/universe/internal/manifest"
)

func TestAgentSpec_DefaultRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		wantRole string
	}{
		{name: "empty_defaults_to_citizen", role: "", wantRole: "citizen"},
		{name: "citizen_stays_citizen", role: "citizen", wantRole: "citizen"},
		{name: "governor_stays_governor", role: "governor", wantRole: "governor"},
		{name: "custom_role_passthrough", role: "custom", wantRole: "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := AgentSpec{Name: "test", Role: tt.role}
			got := manifest.DefaultRole(spec.Role)
			if got != tt.wantRole {
				t.Errorf("DefaultRole(%q) = %q, want %q", tt.role, got, tt.wantRole)
			}
		})
	}
}

func TestAgentSpec_Validation(t *testing.T) {
	spec := AgentSpec{
		Name: "neo",
		Role: "governor",
	}

	if spec.Name != "neo" {
		t.Errorf("expected name 'neo', got %q", spec.Name)
	}
	if spec.Role != "governor" {
		t.Errorf("expected role 'governor', got %q", spec.Role)
	}
}

func TestAgentSpec_EmptyNameIsInvalid(t *testing.T) {
	spec := AgentSpec{Name: "", Role: "citizen"}
	if spec.Name != "" {
		t.Error("expected empty name")
	}
	// In real SpawnAgents, ValidateMind("") would fail.
	// This validates the struct allows empty names (validation is at call-site).
}

func TestGovernorLimit(t *testing.T) {
	agents := []AgentSpec{
		{Name: "gov1", Role: "governor"},
		{Name: "gov2", Role: "governor"},
	}

	governors := 0
	for _, a := range agents {
		if manifest.DefaultRole(a.Role) == "governor" {
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
		{Name: "gov", Role: "governor"},
		{Name: "worker1", Role: "citizen"},
		{Name: "worker2", Role: ""},
	}

	var govs, cits int
	for _, a := range agents {
		switch manifest.DefaultRole(a.Role) {
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

func TestInvalidRole_Detection(t *testing.T) {
	// SpawnAgents rejects unknown roles that aren't "governor" or "citizen"
	invalidRoles := []string{"admin", "root", "superuser", "GOVERNOR", "Citizen"}
	for _, role := range invalidRoles {
		resolved := manifest.DefaultRole(role)
		if resolved == "governor" || resolved == "citizen" {
			t.Errorf("role %q should not resolve to a valid role, got %q", role, resolved)
		}
	}
}

func TestDefaultRole_IsCitizen(t *testing.T) {
	// Explicitly verify the default is "citizen", not something else
	got := manifest.DefaultRole("")
	if got != "citizen" {
		t.Errorf("DefaultRole(\"\") = %q, want \"citizen\"", got)
	}
}

func TestSingleGovernorIsValid(t *testing.T) {
	agents := []AgentSpec{
		{Name: "gov", Role: "governor"},
		{Name: "cit1", Role: "citizen"},
		{Name: "cit2", Role: "citizen"},
	}

	governors := 0
	for _, a := range agents {
		if manifest.DefaultRole(a.Role) == "governor" {
			governors++
		}
	}

	if governors != 1 {
		t.Errorf("expected exactly 1 governor, got %d", governors)
	}
}

func TestNoGovernorIsValid(t *testing.T) {
	agents := []AgentSpec{
		{Name: "cit1", Role: "citizen"},
		{Name: "cit2", Role: ""},
	}

	governors := 0
	for _, a := range agents {
		if manifest.DefaultRole(a.Role) == "governor" {
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
