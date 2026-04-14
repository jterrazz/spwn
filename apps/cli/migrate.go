package cli

import (
	"context"
	"os"

	"spwn.sh/packages/base"
	"spwn.sh/packages/migration"
	"spwn.sh/packages/migration/migrations"
)

// runMigrations applies any pending schema migrations to ~/.spwn.
// Skipped entirely for fresh installs (directory does not exist yet).
func runMigrations() error {
	baseDir := base.BaseDir()
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}
	runner := migration.NewRunner(baseDir, migrations.All())
	return runner.Run(context.Background())
}
