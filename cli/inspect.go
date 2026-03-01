package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <id>",
	Short: "Show universe details",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}

func runInspect(cmd *cobra.Command, args []string) error {
	universeID := args[0]

	arch, err := newArchitect()
	if err != nil {
		return err
	}

	u, err := arch.Inspect(cmd.Context(), universeID)
	if err != nil {
		return fmt.Errorf("error: universe %s not found.\nRun 'universe list' to see available universes", universeID)
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(u, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println()
	fmt.Printf("  Universe %s\n", u.ID)
	fmt.Println()
	fmt.Printf("  Origin:     %s\n", u.Origin)
	fmt.Printf("  Backend:    %s\n", u.Backend)
	fmt.Printf("  Status:     %s\n", u.Status)
	fmt.Printf("  Created:    %s (%s)\n", u.CreatedAt.Format("2006-01-02 15:04:05"), timeAgo(u.CreatedAt))
	fmt.Println()

	if u.Agent != "" {
		fmt.Printf("  Agent:      %s\n", u.Agent)
		fmt.Println()
	}

	fmt.Println("  Constants:")
	fmt.Printf("    CPU: %d cores | Memory: %s | Disk: %s | Timeout: %s\n",
		u.Manifest.Physics.Constants.CPU,
		u.Manifest.Physics.Constants.Memory,
		u.Manifest.Physics.Constants.Disk,
		u.Manifest.Physics.Constants.Timeout,
	)
	fmt.Println()

	fmt.Println("  Laws:")
	fmt.Printf("    Network: %s\n", u.Manifest.Physics.Laws.Network)
	fmt.Printf("    Max processes: %d\n", u.Manifest.Physics.Laws.MaxProcesses)
	fmt.Println()

	if len(u.Manifest.Physics.Elements) > 0 {
		fmt.Println("  Elements:")
		fmt.Printf("    %s\n", strings.Join(u.Manifest.Physics.Elements, ", "))
		fmt.Println()
	}

	if len(u.Manifest.Gate) > 0 {
		fmt.Println("  Interactions:")
		for _, g := range u.Manifest.Gate {
			caps := "all"
			if len(g.Capabilities) > 0 {
				caps = strings.Join(g.Capabilities, ", ")
			}
			fmt.Printf("    %s → %s [%s]\n", g.As, g.Source, caps)
		}
		fmt.Println()
	}

	if u.Workspace != "" {
		fmt.Println("  Mounts:")
		fmt.Printf("    /workspace → %s\n", u.Workspace)
		if u.MindPath != "" {
			fmt.Printf("    /mind      → %s\n", u.MindPath)
		}
		fmt.Println()
	}

	return nil
}
