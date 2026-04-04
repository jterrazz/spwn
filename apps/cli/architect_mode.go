package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// architectModeAllowed lists the top-level commands available inside an Architect container.
var architectModeAllowed = map[string]bool{
	"world":       true,
	"agent":       true,
	"status":      true,
	"architect":   true,
	"dash": true,
	"get":  true,
	"help":        true,
}

// isArchitectMode returns true when running inside an Architect container.
func isArchitectMode() bool {
	return os.Getenv("SPWN_ARCHITECT_MODE") == "1"
}

// validateArchitectCommand checks if the current command is allowed in Architect mode.
// Returns an error for admin-only commands (auth, upgrade, init, doctor).
func validateArchitectCommand(cmd *cobra.Command) error {
	// Walk up to find the top-level subcommand
	top := cmd
	for top.Parent() != nil && top.Parent().Parent() != nil {
		top = top.Parent()
	}

	name := top.Name()

	// Root command (help) is always allowed
	if name == "spwn" {
		return nil
	}

	if architectModeAllowed[name] {
		return nil
	}

	return fmt.Errorf("%q is not available in Architect mode", name)
}
