package agent

import (
	"github.com/jterrazz/universe/cli/ui"
	"github.com/spf13/cobra"
)

// Cmd is the parent command for all agent subcommands.
var Cmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents — the living identities that inhabit universes",
	Long: `Manage agents — the living identities that inhabit universes.

An agent is backed by a Mind — a persistent directory of personas, skills,
knowledge, playbooks, journal entries, and session state. One agent per
universe. The agent survives after the universe is destroyed.`,
}

// newStepper creates a Stepper using the persistent root flags.
func newStepper(cmd *cobra.Command) *ui.Stepper {
	q, _ := cmd.Flags().GetBool("quiet")
	v, _ := cmd.Flags().GetBool("verbose")
	j, _ := cmd.Flags().GetBool("json")
	return ui.New(q, v, j)
}
