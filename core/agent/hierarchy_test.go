package agent

import (
	"testing"
)

func setupHierarchyTest(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
}

func sampleHierarchy() Hierarchy {
	return Hierarchy{
		Slug:        "test-hierarchy",
		Name:        "Test Hierarchy",
		Description: "A hierarchy for testing",
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

func TestCreateAndGetHierarchy(t *testing.T) {
	setupHierarchyTest(t)
	h := sampleHierarchy()

	if err := CreateHierarchy(h); err != nil {
		t.Fatalf("CreateHierarchy: %v", err)
	}

	got, err := GetHierarchy(h.Slug)
	if err != nil {
		t.Fatalf("GetHierarchy: %v", err)
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

func TestCreateHierarchyDuplicate(t *testing.T) {
	setupHierarchyTest(t)
	h := sampleHierarchy()

	if err := CreateHierarchy(h); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := CreateHierarchy(h); err == nil {
		t.Fatal("expected error on duplicate create")
	}
}

func TestListHierarchies(t *testing.T) {
	setupHierarchyTest(t)

	list, err := ListHierarchies()
	if err != nil {
		t.Fatalf("ListHierarchies empty: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list, got %d", len(list))
	}

	if err := CreateHierarchy(sampleHierarchy()); err != nil {
		t.Fatalf("create: %v", err)
	}

	list, err = ListHierarchies()
	if err != nil {
		t.Fatalf("ListHierarchies: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 hierarchy, got %d", len(list))
	}
}

func TestUpdateHierarchy(t *testing.T) {
	setupHierarchyTest(t)
	h := sampleHierarchy()
	if err := CreateHierarchy(h); err != nil {
		t.Fatalf("create: %v", err)
	}

	h.Description = "Updated description"
	if err := UpdateHierarchy(h); err != nil {
		t.Fatalf("UpdateHierarchy: %v", err)
	}

	got, err := GetHierarchy(h.Slug)
	if err != nil {
		t.Fatalf("GetHierarchy: %v", err)
	}
	if got.Description != "Updated description" {
		t.Errorf("description = %q, want %q", got.Description, "Updated description")
	}
}

func TestUpdateHierarchyNotFound(t *testing.T) {
	setupHierarchyTest(t)
	h := sampleHierarchy()
	if err := UpdateHierarchy(h); err == nil {
		t.Fatal("expected error updating non-existent hierarchy")
	}
}

func TestDeleteHierarchy(t *testing.T) {
	setupHierarchyTest(t)
	h := sampleHierarchy()
	if err := CreateHierarchy(h); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := DeleteHierarchy(h.Slug); err != nil {
		t.Fatalf("DeleteHierarchy: %v", err)
	}
	if _, err := GetHierarchy(h.Slug); err == nil {
		t.Fatal("expected not found after delete")
	}
}

func TestDeleteHierarchyNotFound(t *testing.T) {
	setupHierarchyTest(t)
	if err := DeleteHierarchy("nonexistent"); err == nil {
		t.Fatal("expected error deleting non-existent hierarchy")
	}
}

func TestValidateHierarchyMissingSlug(t *testing.T) {
	h := Hierarchy{Name: "No Slug", Roles: []Role{{Name: "r", Level: 0}}}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for missing slug")
	}
}

func TestValidateHierarchyInvalidSlug(t *testing.T) {
	h := Hierarchy{Slug: "BAD SLUG!", Name: "Bad", Roles: []Role{{Name: "r", Level: 0}}}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for invalid slug")
	}
}

func TestValidateHierarchyMissingName(t *testing.T) {
	h := Hierarchy{Slug: "ok", Roles: []Role{{Name: "r", Level: 0}}}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestValidateHierarchyNoRoles(t *testing.T) {
	h := Hierarchy{Slug: "ok", Name: "OK"}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for no roles")
	}
}

func TestValidateHierarchyDuplicateRoles(t *testing.T) {
	h := Hierarchy{
		Slug: "ok", Name: "OK",
		Roles: []Role{{Name: "a", Level: 0}, {Name: "a", Level: 1}},
	}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for duplicate role names")
	}
}

func TestValidateHierarchyBadReportsTo(t *testing.T) {
	h := Hierarchy{
		Slug: "ok", Name: "OK",
		Roles: []Role{{Name: "a", Level: 0, ReportsTo: "nonexistent"}},
	}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for bad reports_to reference")
	}
}

func TestValidateHierarchyBadCanCommand(t *testing.T) {
	h := Hierarchy{
		Slug: "ok", Name: "OK",
		Roles: []Role{{Name: "a", Level: 0, CanCommand: []string{"nonexistent"}}},
	}
	if err := ValidateHierarchy(h); err == nil {
		t.Fatal("expected error for bad can_command reference")
	}
}

func TestGetRole(t *testing.T) {
	setupHierarchyTest(t)
	h := sampleHierarchy()

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
	h := sampleHierarchy()

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

func TestEnsureDefaultHierarchy(t *testing.T) {
	setupHierarchyTest(t)

	if err := EnsureDefaultHierarchy(); err != nil {
		t.Fatalf("first EnsureDefaultHierarchy: %v", err)
	}

	// Should be idempotent.
	if err := EnsureDefaultHierarchy(); err != nil {
		t.Fatalf("second EnsureDefaultHierarchy: %v", err)
	}

	h, err := GetHierarchy("default")
	if err != nil {
		t.Fatalf("GetHierarchy default: %v", err)
	}
	if len(h.Roles) != 2 {
		t.Errorf("default hierarchy roles = %d, want 2", len(h.Roles))
	}

	gov := h.GetRole("governor")
	if gov == nil {
		t.Fatal("expected governor role in default hierarchy")
	}
	if gov.MaxPerWorld != 1 {
		t.Errorf("governor max_per_world = %d, want 1", gov.MaxPerWorld)
	}
}

// TestGetHierarchyNotFound ensures a clean error for missing hierarchies.
func TestGetHierarchyNotFound(t *testing.T) {
	setupHierarchyTest(t)
	_, err := GetHierarchy("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent hierarchy")
	}
}

// TestCreateHierarchyAutoSlug verifies slug is derived from name when empty.
func TestCreateHierarchyAutoSlug(t *testing.T) {
	setupHierarchyTest(t)
	h := Hierarchy{
		Name:  "My Cool Hierarchy",
		Roles: []Role{{Name: "boss", Level: 0}},
	}
	if err := CreateHierarchy(h); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := GetHierarchy("my-cool-hierarchy")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "My Cool Hierarchy" {
		t.Errorf("name = %q, want %q", got.Name, "My Cool Hierarchy")
	}
}
