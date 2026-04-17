package migrations

import (
	"context"
	"os"
	"path/filepath"

	"spwn.sh/packages/migration"
	"spwn.sh/packages/platform"
)

// EnsureUserConfig writes a default ~/.spwn/config.yaml on first
// upgrade if the file does not exist. This is the one-time seed of
// the user-level preferences file introduced in 2026-04 —
// subsequent runs read whatever the user has customised.
//
// No content is ever overwritten: if config.yaml already exists we
// leave it alone (the user's edits win). The sibling marker files
// (.onboarding-complete, .version-check, credentials/) are untouched
// by this migration; they each have their own lifecycle and are
// consulted independently by the features that care about them.
var EnsureUserConfig = migration.Migration{
	Number:      15,
	Description: "seed ~/.spwn/config.yaml with defaults if missing",
	Apply: func(_ context.Context, baseDir string) error {
		path := filepath.Join(baseDir, platform.ConfigFileName)
		if _, err := os.Stat(path); err == nil {
			return nil // already exists — do not overwrite
		}

		// Mirror SaveConfig output deterministically but against the
		// explicit baseDir the migration runner gave us (tests point
		// this at a temp dir; prod runs it at ~/.spwn/).
		content := `# spwn user config — edit freely; spwn re-reads on every CLI invocation.
# Docs: https://spwn.sh/docs/config
apiVersion: ` + platform.CurrentConfigAPIVersion + `
runtime:
    default_backend: spwn:claude-code
telemetry:
    enabled: false
update:
    channel: stable
`
		if err := os.MkdirAll(baseDir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(content), 0o644)
	},
}
