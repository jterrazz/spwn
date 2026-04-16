package cli

import (
	"context"
	"os"

	"spwn.sh/packages/upgrade"
	"spwn.sh/packages/upgrade/migrations"
	"spwn.sh/packages/paths"
)

// runMigrations applies any pending schema migrations to ~/.spwn.
// Skipped entirely for fresh installs (directory does not exist yet).
func runMigrations() error {
	baseDir := paths.BaseDir()
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}
	runner := upgrade.NewRunner(baseDir, migrations.All())
	return runner.Run(context.Background())
}
