package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestEnsureDefaultOrganization_Fixture creates the default
// organization YAML on a fresh install. Fixture's before/ is empty,
// after/ contains only organizations/default.yaml. Harness proves
// the baked-in default-organization content stays byte-stable.
func TestEnsureDefaultOrganization_Fixture(t *testing.T) {
	runFixture(t, EnsureDefaultOrganization, "013_ensure_default_organization")
}

// TestEnsureDefaultOrganization_PreservesExisting verifies the
// migration does NOT overwrite a user-customised default.yaml.
func TestEnsureDefaultOrganization_PreservesExisting(t *testing.T) {
	dir := t.TempDir()
	orgsDir := filepath.Join(dir, "organizations")
	if err := os.MkdirAll(orgsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	custom := []byte("name: Custom\n")
	if err := os.WriteFile(filepath.Join(orgsDir, "default.yaml"), custom, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureDefaultOrganization.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(filepath.Join(orgsDir, "default.yaml"))
	if string(got) != string(custom) {
		t.Errorf("user-customised default.yaml was clobbered:\n got: %s\nwant: %s", got, custom)
	}
}
