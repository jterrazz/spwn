package agent

import (
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
