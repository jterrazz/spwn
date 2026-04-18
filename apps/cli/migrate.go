package cli

import (
	"context"
	"os"

	"spwn.sh/packages/migration"
	"spwn.sh/packages/migration/user"
	"spwn.sh/packages/platform"
)

// runMigrations applies any pending schema migrations to ~/.spwn.
// Skipped entirely for fresh installs (directory does not exist yet).
func runMigrations() error {
	baseDir := platform.BaseDir()
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}
	runner := migration.NewRunner(baseDir, user.All())
	return runner.Run(context.Background())
}
