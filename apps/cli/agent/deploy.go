package agent

import (
	"context"
	"fmt"

	"spwn.sh/packages/universe"
	"github.com/spf13/cobra"
)

var deployRole string

func init() {
	deployCmd.Flags().StringVar(&deployRole, "role", "worker", "Agent role in the world organization")
	Cmd.AddCommand(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy <agent-name> <world-id>",
	Short: "Deploy an agent to a running world",
	Long: `Adds an agent to an already-running world. The agent's mind is
mounted and a Claude Code session starts in the background.

The world must be running (idle or active). The agent must not already
be deployed in that world.`,
	Example: `  spwn agent deploy neo w-mars-47965
  spwn agent deploy morpheus w-mars-47965 --role chief`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		worldID := args[1]
		s := newStepper(cmd)

		ctx := context.Background()
		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return s.FailHint("Docker", err, "Start Docker Desktop or OrbStack, then try again")
		}

		s.Blank()
		s.Start(fmt.Sprintf("Deploying %s to %s...", agentName, worldID))

		if err := arc.DeployAgent(ctx, worldID, agentName, deployRole); err != nil {
			return s.FailHint("Deploy failed", err,
				fmt.Sprintf("Check that %q exists and world %q is running", agentName, worldID))
		}

		s.Done("Deployed", fmt.Sprintf("%s → %s (%s)", agentName, worldID, deployRole))
		s.Blank()
		return nil
	},
}
