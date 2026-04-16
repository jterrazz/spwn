package migrations

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceToWorkspaces(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("004_before.json"), filepath.Join(dir, "state.json"))

	if err := WorkspaceToWorkspaces.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	want, _ := os.ReadFile(testdataPath("004_after.json"))

	// Compare structurally to avoid whitespace differences.
	var gotJSON, wantJSON any
	json.Unmarshal(got, &gotJSON)
	json.Unmarshal(want, &wantJSON)

	gotNorm, _ := json.MarshalIndent(gotJSON, "", "  ")
	wantNorm, _ := json.MarshalIndent(wantJSON, "", "  ")

	if string(gotNorm) != string(wantNorm) {
		t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", gotNorm, wantNorm)
	}
}

func TestWorkspaceToWorkspaces_Idempotent(t *testing.T) {
	dir := t.TempDir()
	copyFile(t, testdataPath("004_before.json"), filepath.Join(dir, "state.json"))

	ctx := context.Background()
	if err := WorkspaceToWorkspaces.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	first, _ := os.ReadFile(filepath.Join(dir, "state.json"))

	if err := WorkspaceToWorkspaces.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	second, _ := os.ReadFile(filepath.Join(dir, "state.json"))

	if string(first) != string(second) {
		t.Error("second run changed output")
	}
}

func TestWorkspaceToWorkspaces_MissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := WorkspaceToWorkspaces.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error on missing file, got: %v", err)
	}
}

func TestWorkspaceToWorkspaces_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	// Write a state that already has workspaces.
	copyFile(t, testdataPath("004_after.json"), filepath.Join(dir, "state.json"))

	if err := WorkspaceToWorkspaces.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "state.json"))
	want, _ := os.ReadFile(testdataPath("004_after.json"))

	// Verify no change.
	var gotJSON, wantJSON any
	json.Unmarshal(got, &gotJSON)
	json.Unmarshal(want, &wantJSON)
	gotNorm, _ := json.MarshalIndent(gotJSON, "", "  ")
	wantNorm, _ := json.MarshalIndent(wantJSON, "", "  ")
	if string(gotNorm) != string(wantNorm) {
		t.Errorf("already-migrated file was changed.\ngot:\n%s\nwant:\n%s", gotNorm, wantNorm)
	}
}
