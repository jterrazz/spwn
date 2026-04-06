package cli

import (
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/universe"
)

// ensureDefaults creates the default world config and hierarchy if they don't
// already exist. This makes the CLI work out of the box without `spwn init`.
func ensureDefaults() error {
	universe.CreateDefaultConfig()
	universe.InitKnowledge()
	_ = agentDomain.EnsureDefaultHierarchy()
	return nil
}
