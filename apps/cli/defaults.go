package cli

import (
	"spwn.sh/packages/universe"
)

// ensureDefaults creates the default world config if it doesn't already exist.
// Organization and schema migrations are handled by runMigrations() which runs first.
func ensureDefaults() error {
	universe.CreateDefaultConfig()
	return nil
}
