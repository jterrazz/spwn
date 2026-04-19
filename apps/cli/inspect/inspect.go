package inspect

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Cmd is the `spwn inspect` root command. Takes an optional agent
// positional arg; no args prints every deployable agent in the
// current project.
var Cmd = &cobra.Command{
	Use:   "inspect [agent]",
	Short: "Show per-agent composition: deps, skills, hooks",
	Long: `Inspect a spwn project: one block per agent, showing the
resolved dependency tree (with transitive (*)-dedup and composition
badges), the skills contributed by tool deps, and the hooks bound
to the agent.

Mirrors the kubectl describe / cargo tree convention: key-value
header, section titles with counts, whitespace-separated sections.

Examples:
  spwn inspect            # every agent
  spwn inspect neo        # one agent
  spwn inspect --offline  # skip live world-status lookup`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve cwd: %w", err)
		}

		offline, _ := cmd.Flags().GetBool("offline")

		opts := Opts{LiveStatus: !offline}
		if len(args) == 1 {
			opts.Agent = args[0]
		}

		m, err := Build(cwd, opts)
		if err != nil {
			return err
		}
		Render(cmd.OutOrStdout(), *m)
		return nil
	},
}

func init() {
	Cmd.Flags().Bool("offline", false, "Skip live world-status lookup (faster, no Docker calls)")
}
