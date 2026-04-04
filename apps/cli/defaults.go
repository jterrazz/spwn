package cli

import (
	"spwn.sh/core/universe"
)

// ensureDefaults creates the default world config if it doesn't already exist.
// This makes the CLI work out of the box without requiring `spwn init` first.
// Note: The default agent is created on-demand when `spwn up` is called
// without --agent, not on every CLI invocation.
func ensureDefaults() error {
	// Create default.yaml if it doesn't exist.
	// CreateDefaultConfig returns an error when the file already exists — ignore it.
	universe.CreateDefaultConfig()

	// Create default knowledge if it doesn't exist.
	universe.InitKnowledge()

	return nil
}
