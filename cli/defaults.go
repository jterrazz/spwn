package cli

import (
	"github.com/jterrazz/spwn/domains/agent"
	"github.com/jterrazz/spwn/domains/universe"
)

// ensureDefaults creates the default universe config and default agent
// if they don't already exist. This makes the CLI work out of the box
// without requiring `spwn init` or `spwn agent init` first.
func ensureDefaults() error {
	// Create default.yaml if it doesn't exist.
	// CreateDefaultConfig returns an error when the file already exists — ignore it.
	universe.CreateDefaultConfig()

	// Create default agent with personas/default.md if it doesn't exist.
	// InitMind returns an error when the agent already exists — ignore it.
	agent.InitMind("default")

	return nil
}
