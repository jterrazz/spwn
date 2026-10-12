package cli

import (
	"context"
	"os"

	"spwn.sh/packages/migration"
	"spwn.sh/packages/migration/user"
	"spwn.sh/packages/platform"
)

func runMigrations() error {
	baseDir := platform.BaseDir()
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil
	}
	runner := migration.NewRunner(baseDir, user.All())
	return runner.Run(context.Background())
}
