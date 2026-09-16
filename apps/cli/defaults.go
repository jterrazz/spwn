package cli

import (
	"spwn.sh/packages/world"
)

// ensureDefaults creates the default world config if it doesn't already exist.
// Organization and schema migrations are handled by runMigrations() which runs first.
func ensureDefaults() error {
	// CreateDefaultConfig reports "already exists" on every run after the
	// first, which is the state this function wants. Any other failure
	// surfaces where the config is actually read.
	_ = world.CreateDefaultConfig()
	return nil
}
