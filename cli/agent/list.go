package agent

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/mind"
)

var (
	listJSON bool
)

var listAgentCmd = &cobra.Command{
	Use:   "list",
	Short: "List agents on this Substrate",
	RunE:  runListAgent,
}

func init() {
	listAgentCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
}

func runListAgent(cmd *cobra.Command, args []string) error {
	agents, err := mind.List()
	if err != nil {
		return err
	}

	if listJSON {
		data, _ := json.MarshalIndent(agents, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(agents) == 0 {
		fmt.Println()
		fmt.Println("  No agents found.")
		fmt.Println()
		fmt.Println("  Create one: universe agent init <name>")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %-16s %-10s\n", "NAME", "LAYERS")
	for _, a := range agents {
		layerCount := 0
		for _, files := range a.Layers {
			if len(files) > 0 {
				layerCount++
			}
		}
		fmt.Printf("  %-16s %d/6\n", a.Name, layerCount)
	}
	fmt.Println()

	return nil
}
