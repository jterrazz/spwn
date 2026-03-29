package agent

import (
	"encoding/json"
	"fmt"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"github.com/spf13/cobra"
)

var listJSON bool

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agents on this Host",
	RunE: func(cmd *cobra.Command, args []string) error {
		agents, err := agentDomain.ListAgents()
		if err != nil {
			return fmt.Errorf("error: cannot list agents.\n%w", err)
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

		t := ui.NewTable(ui.ModeNormal, "NAME", "LAYERS")
		for _, a := range agents {
			layerCount := agentDomain.LayerCount(&a)
			t.AddRow(a.Name, fmt.Sprintf("%d/6", layerCount))
		}
		t.Render()

		return nil
	},
}
