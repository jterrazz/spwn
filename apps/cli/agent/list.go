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
	listJSON          bool
	listFilterUniverse string
)

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	listCmd.Flags().StringVar(&listFilterUniverse, "universe", "", "Filter agents by universe ID")
	Cmd.AddCommand(listCmd)
}

// agentUniverseInfo holds cross-referenced universe data for an agent.
type agentUniverseInfo struct {
	UniverseID string
	Status     string
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agents on this Host",
	RunE: func(cmd *cobra.Command, args []string) error {
		agents, err := agentDomain.ListAgents()
		if err != nil {
			return fmt.Errorf("error: cannot list agents.\n%w", err)
		}

		// Build agent -> universe mapping from state
		agentMap := buildAgentUniverseMap()

		// Filter by universe if requested
		if listFilterUniverse != "" {
			filtered := make([]agentDomain.Info, 0)
			for _, a := range agents {
				if info, ok := agentMap[a.Name]; ok && info.UniverseID == listFilterUniverse {
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
			s.Success("No agents found.")
			s.Log("Run 'spwn agent init [name]' to create one.")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "NAME", "LAYERS", "UNIVERSE", "STATUS")
		for _, a := range agents {
			layerCount := agentDomain.LayerCount(&a)
			universeID := "\u2014"
			status := "unattached"

			if info, ok := agentMap[a.Name]; ok {
				universeID = info.UniverseID
				status = info.Status
			}

			t.AddRow(a.Name, fmt.Sprintf("%d/6", layerCount), universeID, status)
		}
		t.Render()

		return nil
	},
}

// buildAgentUniverseMap reads state.json and returns a map of agent name
// to the universe it is currently attached to.
func buildAgentUniverseMap() map[string]agentUniverseInfo {
	result := make(map[string]agentUniverseInfo)

	ctx := context.Background()
	arc, err := universe.NewArchitectFromEnv()
	if err != nil {
		return result
	}

	universes, err := arc.List(ctx)
	if err != nil {
		return result
	}

	for _, u := range universes {
		if u.Status == universe.StatusDestroyed {
			continue
		}

		// Check primary agent
		if u.Agent != "" {
			result[u.Agent] = agentUniverseInfo{
				UniverseID: u.ID,
				Status:     string(u.Status),
			}
		}

		// Check multi-agent records
		for _, a := range u.Agents {
			result[a.Name] = agentUniverseInfo{
				UniverseID: u.ID,
				Status:     string(a.Status),
			}
		}
	}

	return result
}
