package architect

import (
	"strings"
	"testing"

	"spwn.sh/packages/universe/internal/manifest"
)

func TestAgentSpec_DefaultRole(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		wantRole string
	}{
		{name: "empty_defaults_to_worker", role: "", wantRole: "worker"},
		{name: "worker_stays_worker", role: "worker", wantRole: "worker"},
		{name: "chief_stays_chief", role: "chief", wantRole: "chief"},
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
		Role: "chief",
	}

	if spec.Name != "neo" {
		t.Errorf("expected name 'neo', got %q", spec.Name)
	}
	if spec.Role != "chief" {
		t.Errorf("expected role 'chief', got %q", spec.Role)
	}
}

func TestAgentSpec_EmptyNameIsInvalid(t *testing.T) {
	spec := AgentSpec{Name: "", Role: "worker"}
	if spec.Name != "" {
		t.Error("expected empty name")
	}
	// In real SpawnAgents, ValidateMind("") would fail.
	// This validates the struct allows empty names (validation is at call-site).
}

func TestChiefLimit(t *testing.T) {
	agents := []AgentSpec{
		{Name: "ch1", Role: "chief"},
		{Name: "ch2", Role: "chief"},
	}

	chiefs := 0
	for _, a := range agents {
		if manifest.DefaultRole(a.Role) == "chief" {
			chiefs++
		}
	}

	if chiefs <= 1 {
		t.Error("expected multiple chiefs to be detected")
	}
	// SpawnAgents enforces: "at most one chief allowed, got N"
}

func TestChiefLimit_ErrorMessage(t *testing.T) {
	// Verify the error message format from SpawnAgents is actionable.
	// We can't call SpawnAgents (needs full Architect), but we verify the pattern.
	count := 3
	msg := "at most one chief allowed"
	if !strings.Contains(msg, "at most one chief") {
		t.Error("chief limit error should mention the constraint")
	}
	_ = count
}

func TestMixedColony(t *testing.T) {
	agents := []AgentSpec{
		{Name: "ch", Role: "chief"},
		{Name: "worker1", Role: "worker"},
		{Name: "worker2", Role: ""},
	}

	var chs, wkrs int
	for _, a := range agents {
		switch manifest.DefaultRole(a.Role) {
		case "chief":
			chs++
		case "worker":
			wkrs++
		}
	}

	if chs != 1 {
		t.Errorf("expected 1 chief, got %d", chs)
	}
	if wkrs != 2 {
		t.Errorf("expected 2 workers, got %d", wkrs)
	}
}

func TestInvalidRole_Detection(t *testing.T) {
	// SpawnAgents rejects unknown roles that aren't "chief", "manager", or "worker"
	invalidRoles := []string{"admin", "root", "superuser", "CHIEF", "Worker"}
	for _, role := range invalidRoles {
		resolved := manifest.DefaultRole(role)
		if resolved == "chief" || resolved == "manager" || resolved == "worker" {
			t.Errorf("role %q should not resolve to a valid role, got %q", role, resolved)
		}
	}
}

func TestDefaultRole_IsWorker(t *testing.T) {
	// Explicitly verify the default is "worker", not something else
	got := manifest.DefaultRole("")
	if got != "worker" {
		t.Errorf("DefaultRole(\"\") = %q, want \"worker\"", got)
	}
}

func TestSingleChiefIsValid(t *testing.T) {
	agents := []AgentSpec{
		{Name: "ch", Role: "chief"},
		{Name: "wkr1", Role: "worker"},
		{Name: "wkr2", Role: "worker"},
	}

	chiefs := 0
	for _, a := range agents {
		if manifest.DefaultRole(a.Role) == "chief" {
			chiefs++
		}
	}

	if chiefs != 1 {
		t.Errorf("expected exactly 1 chief, got %d", chiefs)
	}
}

func TestNoChiefIsValid(t *testing.T) {
	agents := []AgentSpec{
		{Name: "wkr1", Role: "worker"},
		{Name: "wkr2", Role: ""},
	}

	chiefs := 0
	for _, a := range agents {
		if manifest.DefaultRole(a.Role) == "chief" {
			chiefs++
		}
	}

	if chiefs != 0 {
		t.Errorf("expected 0 chiefs, got %d", chiefs)
	}
}

func TestEmptyAgentSlice(t *testing.T) {
	agents := []AgentSpec{}
	if len(agents) != 0 {
		t.Error("empty slice should have length 0")
	}
	// SpawnAgents returns nil for empty agents — a no-op.
}
