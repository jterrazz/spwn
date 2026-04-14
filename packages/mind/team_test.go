package mind

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Matrix Ops", "matrix-ops"},
		{"infra", "infra"},
		{"The  Frontend  Squad", "the-frontend-squad"},
		{"  spaces  ", "spaces"},
		{"UPPER", "upper"},
		{"a-b-c", "a-b-c"},
		{"", "team"},
		{"@#$%", "team"},
	}
	for _, tt := range tests {
		got := Slugify(tt.in)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTeamCRUD(t *testing.T) {
	// Point SPWN_HOME at a temp directory.
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Create
	team := Team{Name: "Matrix Ops", Slug: "matrix-ops", Color: "#8B5CF6", Description: "Core team"}
	if err := CreateTeam(team); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}

	// Duplicate should fail
	if err := CreateTeam(team); err == nil {
		t.Fatal("expected duplicate error")
	}

	// Get
	got, err := GetTeam("matrix-ops")
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Name != "Matrix Ops" || got.Color != "#8B5CF6" {
		t.Errorf("unexpected team: %+v", got)
	}

	// List
	teams, err := ListTeams()
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 1 || teams[0].Slug != "matrix-ops" {
		t.Errorf("ListTeams: %+v", teams)
	}

	// Update
	got.Description = "Updated desc"
	if err := UpdateTeam(*got); err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	got2, _ := GetTeam("matrix-ops")
	if got2.Description != "Updated desc" {
		t.Errorf("update didn't persist: %q", got2.Description)
	}

	// Delete
	if err := DeleteTeam("matrix-ops"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	if _, err := GetTeam("matrix-ops"); err == nil {
		t.Fatal("expected not-found after delete")
	}
}

func TestTeamMembers(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	// Create team
	_ = CreateTeam(Team{Name: "Ops", Slug: "ops"})

	// Create two agent dirs with agent.yaml referencing the team
	agentsDir := filepath.Join(tmp, "agents")
	for _, name := range []string{"neo", "trinity"} {
		dir := filepath.Join(agentsDir, name)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte("team: ops\n"), 0644)
	}
	// One agent without team
	soloDir := filepath.Join(agentsDir, "qa")
	os.MkdirAll(soloDir, 0755)
	os.WriteFile(filepath.Join(soloDir, "agent.yaml"), []byte("role: worker\n"), 0644)

	members, err := TeamMembers("ops")
	if err != nil {
		t.Fatalf("TeamMembers: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d: %v", len(members), members)
	}

	// Non-existent team
	empty, _ := TeamMembers("nonexistent")
	if len(empty) != 0 {
		t.Errorf("expected 0 for nonexistent team, got %v", empty)
	}
}

func TestGetTeam_NotFound(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	_, err := GetTeam("nope")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteTeam_NotFound(t *testing.T) {
	t.Setenv("SPWN_HOME", t.TempDir())
	if err := DeleteTeam("nope"); err == nil {
		t.Fatal("expected error")
	}
}
