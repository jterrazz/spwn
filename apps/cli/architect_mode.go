package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var architectModeAllowed = map[string]bool{
	"world":     true,
	"agent":     true,
	"status":    true,
	"architect": true,
	"web":       true,
	"install":   true,
	"skill":     true,
	"help":      true,
}

func isArchitectMode() bool {
	return os.Getenv("SPWN_ARCHITECT_MODE") == "1"
}

func validateArchitectCommand(cmd *cobra.Command) error {
	top := cmd
	for top.Parent() != nil && top.Parent().Parent() != nil {
		top = top.Parent()
	}

	name := top.Name()

	if name == "spwn" {
		return nil
	}

	if architectModeAllowed[name] {
		return nil
	}

	return fmt.Errorf("%q is not available in Architect mode", name)
}
