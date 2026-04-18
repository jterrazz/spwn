package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestRenameHierarchies_Fixture verifies the directory rename via
// the shared harness. Fixture lives at
// testdata/user/012_rename_hierarchies/.
func TestRenameHierarchies_Fixture(t *testing.T) {
	runFixture(t, RenameHierarchiesToOrganizations, "012_rename_hierarchies")
}

// TestRenameHierarchies_AlreadyMigrated: when organizations/ already
// exists the migration leaves both sides alone (no clobber).
func TestRenameHierarchies_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	orgsDir := filepath.Join(dir, "organizations")
	if err := os.MkdirAll(orgsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orgsDir, "marker"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RenameHierarchiesToOrganizations.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(orgsDir, "marker"))
	if string(got) != "keep" {
		t.Errorf("organizations/marker content clobbered: %q", got)
	}
}

// TestRenameHierarchies_NoHierarchiesDir: fresh installs have no
// hierarchies/ — migration is a no-op.
func TestRenameHierarchies_NoHierarchiesDir(t *testing.T) {
	dir := t.TempDir()
	if err := RenameHierarchiesToOrganizations.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
}
