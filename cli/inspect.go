package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <universe-id>",
	Short: "Show universe details, physics, and agent status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		universeID := args[0]

		arc, err := newArchitect()
		if err != nil {
			return err
		}

		u, err := arc.Inspect(ctx, universeID)
		if err != nil {
			return fmt.Errorf("error: universe %s not found.\nRun 'universe list' to see available universes.", universeID)
		}

		if jsonOutput {
			data, _ := json.MarshalIndent(u, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Printf("\n  Universe %s\n\n", u.ID)
		fmt.Printf("  Config:     %s\n", u.Config)
		fmt.Printf("  Backend:    %s\n", u.Backend)
		fmt.Printf("  Status:     %s\n", u.Status)
		fmt.Printf("  Created:    %s (%s)\n", u.CreatedAt.Format("2006-01-02 15:04:05"), timeAgo(u.CreatedAt))

		if u.AgentID != "" {
			fmt.Printf("\n  Agent:      %s\n", u.AgentID)
		}

		fmt.Printf("\n  Constants:\n")
		fmt.Printf("    CPU: %d core(s) | Memory: %s | Disk: %s | Timeout: %s\n",
			u.Manifest.Physics.Constants.CPU,
			u.Manifest.Physics.Constants.Memory,
			u.Manifest.Physics.Constants.Disk,
			u.Manifest.Physics.Constants.Timeout,
		)

		fmt.Printf("\n  Laws:\n")
		fmt.Printf("    Network: %s\n", u.Manifest.Physics.Laws.Network)
		fmt.Printf("    Max processes: %d\n", u.Manifest.Physics.Laws.MaxProcesses)

		if u.Workspace != "" {
			fmt.Printf("\n  Mounts:\n")
			fmt.Printf("    /workspace → %s (read-write)\n", u.Workspace)
		}
		if u.MindPath != "" {
			if u.Workspace == "" {
				fmt.Printf("\n  Mounts:\n")
			}
			fmt.Printf("    /mind      → %s (read-write)\n", u.MindPath)
		}

		fmt.Println()
		return nil
	},
}
