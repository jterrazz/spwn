package cli

import (
	"github.com/jterrazz/spwn/internal/manifest"
	"github.com/jterrazz/spwn/internal/mind"
)

// ensureDefaults creates the default universe config and default agent
// if they don't already exist. This makes the CLI work out of the box
// without requiring `spwn init` or `spwn agent init` first.
func ensureDefaults() error {
	// Create default.yaml if it doesn't exist.
	// CreateDefault returns an error when the file already exists — ignore it.
	manifest.CreateDefault()

	// Create default agent with personas/default.md if it doesn't exist.
	// Init returns an error when the agent already exists — ignore it.
	mind.Init("default")

	return nil
}
