package user

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/platform"
)

// TestEnsureUserConfig_Fixture asserts that the baked-in default
// config.yaml is byte-exact. Regression guard: if the
// platform.CurrentConfigAPIVersion constant or the seed template
// drifts, the fixture diff surfaces it immediately.
// Fixture at testdata/user/015_ensure_user_config/.
func TestEnsureUserConfig_Fixture(t *testing.T) {
	runFixture(t, EnsureUserConfig, "015_ensure_user_config")
}

// TestEnsureUserConfig_keepsExistingFile covers the
// don't-overwrite-user-edits contract, which a fixture can't
// express (the fixture shows one transformation, not the "skip if
// present" branch).
func TestEnsureUserConfig_keepsExistingFile(t *testing.T) {
	base := t.TempDir()
	existing := "# user customisations preserved\napiVersion: spwn/v2\nonboarded: true\n"
	path := filepath.Join(base, platform.ConfigFileName)
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Errorf("existing config overwritten. got:\n%s\nwant:\n%s", got, existing)
	}
}

// TestEnsureUserConfig_idempotent: running twice is a no-op after
// the first run — the second Apply does not touch the file.
func TestEnsureUserConfig_idempotent(t *testing.T) {
	base := t.TempDir()
	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	path := filepath.Join(base, platform.ConfigFileName)
	first, _ := os.ReadFile(path)

	if err := EnsureUserConfig.Apply(context.Background(), base); err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	second, _ := os.ReadFile(path)
	if string(first) != string(second) {
		t.Errorf("second Apply changed file")
	}
}
