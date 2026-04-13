package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDir(t, s, d)
		} else {
			copyFile(t, s, d)
		}
	}
}

func TestTierToRoleProfiles(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, testdataPath("002_before"), dir)

	if err := TierToRoleProfiles.Apply(context.Background(), dir); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "agents", "neo", "profile.yaml"))
	want, _ := os.ReadFile(testdataPath(filepath.Join("002_after", "agents", "neo", "profile.yaml")))
	if string(got) != string(want) {
		t.Errorf("mismatch.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTierToRoleProfiles_Idempotent(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, testdataPath("002_before"), dir)

	ctx := context.Background()
	if err := TierToRoleProfiles.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if err := TierToRoleProfiles.Apply(ctx, dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "agents", "neo", "profile.yaml"))
	want, _ := os.ReadFile(testdataPath(filepath.Join("002_after", "agents", "neo", "profile.yaml")))
	if string(got) != string(want) {
		t.Errorf("second run changed output.\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTierToRoleProfiles_MissingDir(t *testing.T) {
	dir := t.TempDir()
	if err := TierToRoleProfiles.Apply(context.Background(), dir); err != nil {
		t.Fatalf("expected no error on missing agents dir, got: %v", err)
	}
}
