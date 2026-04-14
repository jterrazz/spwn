package migrations

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameDefaultRoles_StateJSON(t *testing.T) {
	dir := t.TempDir()

	// Copy before state.json
	before, err := os.ReadFile(testdataPath("005_before.json"))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "state.json"), before, 0644)

	if err := RenameDefaultRoles.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	want, _ := os.ReadFile(testdataPath("005_after.json"))
	if string(got) != string(want) {
		t.Errorf("state.json mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenameDefaultRoles_Profiles(t *testing.T) {
	dir := t.TempDir()

	// Set up agent directories with old roles
	for _, agent := range []string{"morpheus", "neo"} {
		agentDir := filepath.Join(dir, "agents", agent)
		os.MkdirAll(agentDir, 0755)
		src, _ := os.ReadFile(testdataPath(filepath.Join("005_before", "agents", agent, "profile.yaml")))
		os.WriteFile(filepath.Join(agentDir, "profile.yaml"), src, 0644)
	}

	if err := RenameDefaultRoles.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	for _, agent := range []string{"morpheus", "neo"} {
		got, _ := os.ReadFile(filepath.Join(dir, "agents", agent, "profile.yaml"))
		want, _ := os.ReadFile(testdataPath(filepath.Join("005_after", "agents", agent, "profile.yaml")))
		if string(got) != string(want) {
			t.Errorf("profile.yaml for %s mismatch.\ngot:\n%s\nwant:\n%s", agent, got, want)
		}
	}
}

func TestRenameDefaultRoles_Hierarchy(t *testing.T) {
	dir := t.TempDir()

	// Set up old hierarchy
	hierDir := filepath.Join(dir, "hierarchies")
	os.MkdirAll(hierDir, 0755)
	src, _ := os.ReadFile(testdataPath(filepath.Join("005_before", "hierarchies", "default.yaml")))
	os.WriteFile(filepath.Join(hierDir, "default.yaml"), src, 0644)

	if err := RenameDefaultRoles.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(hierDir, "default.yaml"))
	want, _ := os.ReadFile(testdataPath(filepath.Join("005_after", "hierarchies", "default.yaml")))
	if string(got) != string(want) {
		t.Errorf("hierarchy mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenameDefaultRoles_Idempotent(t *testing.T) {
	dir := t.TempDir()

	// Set up already-migrated state
	os.WriteFile(filepath.Join(dir, "state.json"), []byte(`[{"agents":[{"role":"chief"}]}]`), 0644)

	ctx := context.Background()
	if err := RenameDefaultRoles.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	if strings.Contains(string(got), "governor") || strings.Contains(string(got), "citizen") {
		t.Error("idempotent run should not reintroduce old roles")
	}
}

func TestRenameDefaultRoles_NoStateFile(t *testing.T) {
	dir := t.TempDir()

	// No state.json - should not error
	if err := RenameDefaultRoles.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error for missing state.json, got: %v", err)
	}
}
