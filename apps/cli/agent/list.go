package agent

import (
	"context"
	"fmt"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/packages/agent"
	"spwn.sh/packages/world"
	"github.com/spf13/cobra"
)

var listFilterWorld string

func init() {
	listCmd.Flags().StringVar(&listFilterWorld, "world", "", "Filter agents by world ID")
	Cmd.AddCommand(listCmd)
}

// agentWorldInfo holds cross-referenced world data for an agent.
type agentWorldInfo struct {
	WorldID string
	Status  string
}

var listCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all agents on this Host",
	RunE: func(cmd *cobra.Command, args []string) error {
		agents, err := agentDomain.ListAgents()
		if err != nil {
			return fmt.Errorf("cannot list agents: %w", err)
		}

		// Build agent -> world mapping from state
		agentMap := buildAgentWorldMap()

		// Filter by world if requested
		if listFilterWorld != "" {
			filtered := make([]agentDomain.Info, 0)
			for _, a := range agents {
				if info, ok := agentMap[a.Name]; ok && info.WorldID == listFilterWorld {
					filtered = append(filtered, a)
				}
			}
			agents = filtered
		}

		s := newStepper(cmd)

		if len(agents) == 0 {
			s.Blank()
			s.Success("No agents yet.")
			s.Log("Create one with: spwn agent new <name>")
			s.Blank()
			return nil
		}

		t := ui.NewTable("NAME", "WORLD", "STATUS")
		for _, a := range agents {
			worldID := "\u2014"
			status := "unattached"

			if info, ok := agentMap[a.Name]; ok {
				worldID = info.WorldID
				status = info.Status
			}

			t.AddRow(a.Name, worldID, status)
		}
		t.Render()

		return nil
	},
}

// buildAgentWorldMap reads state.json and returns a map of agent name
// to the world it is currently attached to.
func buildAgentWorldMap() map[string]agentWorldInfo {
	result := make(map[string]agentWorldInfo)

	ctx := context.Background()
	arc, err := world.NewArchitectFromEnv()
	if err != nil {
		return result
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return result
	}

	for _, u := range worlds {
		if u.Status == world.StatusDestroyed {
			continue
		}

		// Check primary agent
		if u.Agent != "" {
			result[u.Agent] = agentWorldInfo{
				WorldID: u.ID,
				Status:  string(u.Status),
			}
		}

		// Check multi-agent records
		for _, a := range u.Agents {
			result[a.Name] = agentWorldInfo{
				WorldID: u.ID,
				Status:  string(a.Status),
			}
		}
	}

	return result
}
