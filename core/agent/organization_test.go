package agent

import (
	"testing"
)

func setupOrganizationTest(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
}

func sampleOrganization() Organization {
	return Organization{
		Slug:        "test-organization",
		Name:        "Test Organization",
		Description: "An organization for testing",
		Roles: []Role{
			{
				Name:        "leader",
				Level:       0,
				CanCommand:  []string{"worker"},
				MaxPerWorld: 1,
				Permissions: []string{"delegate"},
			},
			{
				Name:        "worker",
				Level:       1,
				ReportsTo:   "leader",
				Permissions: []string{"execute"},
			},
		},
	}
}

func TestCreateAndGetOrganization(t *testing.T) {
	setupOrganizationTest(t)
	h := sampleOrganization()

	if err := CreateOrganization(h); err != nil {
		t.Fatalf("CreateOrganization: %v", err)
	}

	got, err := GetOrganization(h.Slug)
	if err != nil {
		t.Fatalf("GetOrganization: %v", err)
	}
	if got.Name != h.Name {
		t.Errorf("name = %q, want %q", got.Name, h.Name)
	}
	if len(got.Roles) != 2 {
		t.Errorf("roles count = %d, want 2", len(got.Roles))
	}
	if got.Slug != h.Slug {
		t.Errorf("slug = %q, want %q", got.Slug, h.Slug)
	}
}

func TestCreateOrganizationDuplicate(t *testing.T) {
	setupOrganizationTest(t)
	h := sampleOrganization()

	if err := CreateOrganization(h); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := CreateOrganization(h); err == nil {
		t.Fatal("expected error on duplicate create")
	}
}

func TestListOrganizations(t *testing.T) {
	setupOrganizationTest(t)

	list, err := ListOrganizations()
	if err != nil {
		t.Fatalf("ListOrganizations empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	if err := CreateOrganization(sampleOrganization()); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err = ListOrganizations()
	if err != nil {
		t.Fatalf("ListOrganizations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(list))
	}
}

func TestUpdateOrganization(t *testing.T) {
	setupOrganizationTest(t)
	h := sampleOrganization()
	if err := CreateOrganization(h); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.Description = "Updated description"
	if err := UpdateOrganization(h); err != nil {
		t.Fatalf("UpdateOrganization: %v", err)
	}

	got, err := GetOrganization(h.Slug)
	if err != nil {
		t.Fatalf("GetOrganization: %v", err)
	}
	if got.Description != "Updated description" {
		t.Errorf("description = %q, want %q", got.Description, "Updated description")
	}
}

func TestUpdateOrganizationNotFound(t *testing.T) {
	setupOrganizationTest(t)
	h := sampleOrganization()
	if err := UpdateOrganization(h); err == nil {
		t.Fatal("expected error updating non-existent organization")
	}
}

func TestDeleteOrganization(t *testing.T) {
	setupOrganizationTest(t)
	h := sampleOrganization()
	if err := CreateOrganization(h); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := DeleteOrganization(h.Slug); err != nil {
		t.Fatalf("DeleteOrganization: %v", err)
	}
	if _, err := GetOrganization(h.Slug); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestDeleteOrganizationNotFound(t *testing.T) {
	setupOrganizationTest(t)
	if err := DeleteOrganization("nonexistent"); err == nil {
		t.Fatal("expected error deleting non-existent organization")
	}
}

func TestValidateOrganizationMissingSlug(t *testing.T) {
	h := Organization{Name: "No Slug", Roles: []Role{{Name: "r", Level: 0}}}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for missing slug")
	}
}

func TestValidateOrganizationInvalidSlug(t *testing.T) {
	h := Organization{Slug: "BAD SLUG!", Name: "Bad", Roles: []Role{{Name: "r", Level: 0}}}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for invalid slug")
	}
}

func TestValidateOrganizationMissingName(t *testing.T) {
	h := Organization{Slug: "ok", Roles: []Role{{Name: "r", Level: 0}}}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestValidateOrganizationNoRoles(t *testing.T) {
	h := Organization{Slug: "ok", Name: "OK"}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for no roles")
	}
}

func TestValidateOrganizationDuplicateRoles(t *testing.T) {
	h := Organization{
		Slug: "ok", Name: "OK",
		Roles: []Role{{Name: "a", Level: 0}, {Name: "a", Level: 1}},
	}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for duplicate role names")
	}
}

func TestValidateOrganizationBadReportsTo(t *testing.T) {
	h := Organization{
		Slug: "ok", Name: "OK",
		Roles: []Role{{Name: "a", Level: 0, ReportsTo: "nonexistent"}},
	}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for bad reports_to reference")
	}
}

func TestValidateOrganizationBadCanCommand(t *testing.T) {
	h := Organization{
		Slug: "ok", Name: "OK",
		Roles: []Role{{Name: "a", Level: 0, CanCommand: []string{"nonexistent"}}},
	}
	if err := ValidateOrganization(h); err == nil {
		t.Fatal("expected error for bad can_command reference")
	}
}

func TestGetRole(t *testing.T) {
	setupOrganizationTest(t)
	h := sampleOrganization()

	leader := h.GetRole("leader")
	if leader == nil {
		t.Fatal("expected to find leader role")
	}
	if leader.Level != 0 {
		t.Errorf("leader level = %d, want 0", leader.Level)
	}

	missing := h.GetRole("nonexistent")
	if missing != nil {
		t.Fatal("expected nil for nonexistent role")
	}
}

func TestRoleCanCommand(t *testing.T) {
	h := sampleOrganization()

	if !h.RoleCanCommand("leader", "worker") {
		t.Error("leader should be able to command worker")
	}
	if h.RoleCanCommand("worker", "leader") {
		t.Error("worker should not be able to command leader")
	}
	if h.RoleCanCommand("nonexistent", "worker") {
		t.Error("nonexistent role should not be able to command anyone")
	}
}

func TestEnsureDefaultOrganization(t *testing.T) {
	setupOrganizationTest(t)

	if err := EnsureDefaultOrganization(); err != nil {
		t.Fatalf("first EnsureDefaultOrganization: %v", err)
	}

	// Should be idempotent.
	if err := EnsureDefaultOrganization(); err != nil {
		t.Fatalf("second EnsureDefaultOrganization: %v", err)
	}

	h, err := GetOrganization("default")
	if err != nil {
		t.Fatalf("GetOrganization default: %v", err)
	}
	if len(h.Roles) != 3 {
		t.Errorf("default organization roles = %d, want 3", len(h.Roles))
	}

	chief := h.GetRole("chief")
	if chief == nil {
		t.Fatal("expected chief role in default organization")
	}
	if chief.MaxPerWorld != 1 {
		t.Errorf("chief max_per_world = %d, want 1", chief.MaxPerWorld)
	}
}

// TestGetOrganizationNotFound ensures a clean error for missing organizations.
func TestGetOrganizationNotFound(t *testing.T) {
	setupOrganizationTest(t)
	_, err := GetOrganization("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent organization")
	}
}

// TestCreateOrganizationAutoSlug verifies slug is derived from name when empty.
func TestCreateOrganizationAutoSlug(t *testing.T) {
	setupOrganizationTest(t)
	h := Organization{
		Name:  "My Cool Organization",
		Roles: []Role{{Name: "boss", Level: 0}},
	}
	if err := CreateOrganization(h); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := GetOrganization("my-cool-organization")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "My Cool Organization" {
		t.Errorf("name = %q, want %q", got.Name, "My Cool Organization")
	}
}
