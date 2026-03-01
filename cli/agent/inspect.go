package agent

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/jterrazz/universe/internal/config"
	"github.com/jterrazz/universe/internal/journal"
	"github.com/jterrazz/universe/internal/mind"
)

var (
	inspectJSON bool
)

var inspectAgentCmd = &cobra.Command{
	Use:   "inspect <name>",
	Short: "Show Mind details",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspectAgent,
}

func init() {
	inspectAgentCmd.Flags().BoolVar(&inspectJSON, "json", false, "Output as JSON")
}

func runInspectAgent(cmd *cobra.Command, args []string) error {
	name := args[0]

	info, err := mind.Inspect(name)
	if err != nil {
		return fmt.Errorf("error: agent %q not found.\nRun 'universe agent list' to see available agents, or 'universe agent init %s' to create one", name, name)
	}

	if inspectJSON {
		data, _ := json.MarshalIndent(info, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println()
	fmt.Printf("  Agent: %s\n", info.Name)
	fmt.Printf("  Path:  %s\n", info.Path)
	fmt.Println()
	fmt.Println("  Layers:")
	for _, layer := range config.MindLayers {
		files := info.Layers[layer]
		if len(files) == 0 {
			fmt.Printf("    %-14s (empty)\n", layer+"/")
		} else {
			detail := fmt.Sprintf("%d files", len(files))
			if len(files) <= 3 {
				detail = fmt.Sprintf("%d files  (%s)", len(files), joinFiles(files))
			}
			fmt.Printf("    %-14s %s\n", layer+"/", detail)
		}
	}
	fmt.Println()

	// Show recent journal entries.
	entries, err := journal.List(info.Path, 5)
	if err == nil && len(entries) > 0 {
		fmt.Println("  Recent journal:")
		for _, e := range entries {
			fmt.Printf("    %s  %-24s  %-10s  exit=%d  %s\n",
				e.EndedAt.Format("2006-01-02 15:04"),
				e.UniverseID,
				e.Outcome,
				e.ExitCode,
				e.Duration.Truncate(time.Second),
			)
		}
		fmt.Println()
	}

	return nil
}

func joinFiles(files []string) string {
	result := ""
	for i, f := range files {
		if i > 0 {
			result += ", "
		}
		result += f
	}
	return result
}
