package user

import (
	"context"
	"testing"
)

// TestDropLegacyStateJSON_Fixture covers the pre-labels install
// shape end-to-end via the shared harness. Before: state.json +
// state.json.bak + runtime/ with a per-world file. After: all
// three gone. Fixture at testdata/user/016_drop_legacy_state_json/.
func TestDropLegacyStateJSON_Fixture(t *testing.T) {
	runFixture(t, DropLegacyStateJSON, "016_drop_legacy_state_json")
}

// TestDropLegacyStateJSON_NoopOnFreshInstall: a fresh install has
// none of the three legacy paths. Migration must not error.
func TestDropLegacyStateJSON_NoopOnFreshInstall(t *testing.T) {
	base := t.TempDir()
	if err := DropLegacyStateJSON.Apply(context.Background(), base); err != nil {
		t.Errorf("Apply on empty baseDir should be no-op; got %v", err)
	}
}
