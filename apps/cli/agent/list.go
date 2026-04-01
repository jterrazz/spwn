package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

var (
	listJSON       bool
	listFilterWorld string
)

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
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

		if listJSON {
			data, _ := json.MarshalIndent(agents, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		s := newStepper(cmd)

		if len(agents) == 0 {
			s.Blank()
			s.Success("No agents yet.")
			s.Log("Create one with: spwn agent new <name>")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "NAME", "LAYERS", "WORLD", "STATUS")
		for _, a := range agents {
			layerCount := agentDomain.LayerCount(&a)
			worldID := "\u2014"
			status := "unattached"

			if info, ok := agentMap[a.Name]; ok {
				worldID = info.WorldID
				status = info.Status
			}

			t.AddRow(a.Name, fmt.Sprintf("%d/6", layerCount), worldID, status)
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
	arc, err := universe.NewArchitectFromEnv()
	if err != nil {
		return result
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return result
	}

	for _, u := range worlds {
		if u.Status == universe.StatusDestroyed {
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
