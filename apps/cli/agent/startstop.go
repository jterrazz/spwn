package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"spwn.sh/packages/project"
	"spwn.sh/packages/world"
)

// startCmd is `spwn agent start <name>` — start the world that
// contains the named agent. In the new grammar, agents don't run on
// their own: every running agent lives in a world declared by
// spwn.yaml. This command finds the first world whose inline agent
// list contains <name> and brings it up.
var startCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start the world that contains this agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		worldName, err := findWorldForAgent(agentName)
		if err != nil {
			return err
		}
		// Delegate to `spwn up <worldName>` via the shared root.
		root := cmd.Root()
		root.SetArgs([]string{"up", worldName})
		return root.Execute()
	},
}

// stopCmd is `spwn agent stop <name>` — stop the world that contains
// the named agent. The agent itself is never deleted; only its
// runtime container is torn down.
var stopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop the world that contains this agent",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		// First try to match a live running world by AgentRecord name.
		ctx := context.Background()
		arc, err := world.NewArchitectFromEnv()
		if err == nil {
			worlds, lerr := arc.List(ctx)
			if lerr == nil {
				for _, u := range worlds {
					if u.Agent == agentName {
						_, derr := arc.Destroy(ctx, u.ID)
						return derr
					}
					for _, a := range u.Agents {
						if a.Name == agentName {
							_, derr := arc.Destroy(ctx, u.ID)
							return derr
						}
					}
				}
			}
		}
		// Fall back to spwn.yaml resolution → `spwn down <worldName>`.
		worldName, err := findWorldForAgent(agentName)
		if err != nil {
			return err
		}
		root := cmd.Root()
		root.SetArgs([]string{"down", worldName})
		return root.Execute()
	},
}

func init() {
	Cmd.AddCommand(startCmd)
	Cmd.AddCommand(stopCmd)
}

// findWorldForAgent locates the first spwn.yaml world entry that
// references the named agent. Returns a descriptive error when no
// project is active or the agent is absent from every world.
func findWorldForAgent(agentName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	p, err := project.Find(cwd)
	if err != nil {
		return "", fmt.Errorf("load spwn.yaml: %w", err)
	}
	if p == nil {
		return "", fmt.Errorf("no spwn.yaml in this directory tree.\nRun \"spwn init\" first")
	}
	for name, w := range p.Manifest.Worlds {
		for _, a := range w.Agents {
			if a == agentName {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("agent %q is not in any world in spwn.yaml", agentName)
}
