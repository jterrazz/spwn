package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// godModeAllowed lists the top-level commands available inside an Architect container.
var godModeAllowed = map[string]bool{
	"world":       true,
	"agent":       true,
	"status":      true,
	"architect":   true,
	"dash": true,
	"get":  true,
	"help":        true,
}

// isGodMode returns true when running inside an Architect container.
func isGodMode() bool {
	return os.Getenv("SPWN_GOD_MODE") == "1"
}

// validateGodCommand checks if the current command is allowed in Architect mode.
// Returns an error for admin-only commands (auth, upgrade, init, doctor).
func validateGodCommand(cmd *cobra.Command) error {
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

	if godModeAllowed[name] {
		return nil
	}

	return fmt.Errorf("%q is not available in Architect mode", name)
}
