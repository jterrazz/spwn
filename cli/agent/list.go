package agent

import (
	"encoding/json"
	"fmt"

	"github.com/jterrazz/universe/internal/mind"
	"github.com/spf13/cobra"
)

var listJSON bool

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agents on this Substrate",
	RunE: func(cmd *cobra.Command, args []string) error {
		agents, err := mind.List()
		if err != nil {
			return fmt.Errorf("error: cannot list agents.\n%w", err)
		}

		if listJSON {
			data, _ := json.MarshalIndent(agents, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if len(agents) == 0 {
			fmt.Println("\n  No agents found.")
			fmt.Println("  Run 'universe agent init <name>' to create one.")
			return nil
		}

		fmt.Printf("\n  %-20s %-10s\n", "NAME", "LAYERS")
		for _, a := range agents {
			layerCount := mind.LayerCount(&a)
			fmt.Printf("  %-20s %d/6\n", a.Name, layerCount)
		}
		fmt.Println()

		return nil
	},
}
