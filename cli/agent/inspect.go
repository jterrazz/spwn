package agent

import (
	"encoding/json"
	"fmt"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/mind"
	"github.com/spf13/cobra"
)

var inspectJSON bool

func init() {
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "Output as JSON")
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <agent-name>",
	Short: "Show agent details, Mind layers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		info, err := mind.Inspect(name)
		if err != nil {
			return fmt.Errorf("error: agent %q not found.\nRun 'universe agent list' to see available agents.", name)
		}

		if inspectJSON {
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("\n  Agent: %s\n", info.Name)
		fmt.Printf("  Path:  %s\n\n", info.Path)
		fmt.Printf("  Layers:\n")
		for _, layer := range config.MindLayers {
			files := info.Layers[layer]
			if len(files) == 0 {
				fmt.Printf("    %-14s (empty)\n", layer+"/")
			} else {
				detail := fmt.Sprintf("%d file(s)", len(files))
				if len(files) <= 3 {
					detail = fmt.Sprintf("%d file(s)  (%s)", len(files), joinFiles(files))
				}
				fmt.Printf("    %-14s %s\n", layer+"/", detail)
			}
		}
		fmt.Println()

		return nil
	},
}

func joinFiles(files []string) string {
	s := ""
	for i, f := range files {
		if i > 0 {
			s += ", "
		}
		s += f
	}
	return s
}
