package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jterrazz/universe/cli/ui"
	"github.com/jterrazz/universe/internal/architect"
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
		s := ui.New(quiet, verbose, jsonOutput)

		arc, err := architect.NewFromEnv()
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

		s.Blank()
		s.Info("Universe:", u.ID)
		s.Blank()
		s.Info("Config:", u.Config)
		s.Info("Backend:", u.Backend)
		s.Info("Status:", string(u.Status))
		s.Info("Created:", fmt.Sprintf("%s (%s)", u.CreatedAt.Format("2006-01-02 15:04:05"), timeAgo(u.CreatedAt)))

		if u.AgentID != "" {
			s.Blank()
			s.Info("Agent:", u.AgentID)
		}

		s.Blank()
		s.Info("Constants:", fmt.Sprintf("CPU: %d core(s) | Memory: %s | Disk: %s | Timeout: %s",
			u.Manifest.Physics.Constants.CPU,
			u.Manifest.Physics.Constants.Memory,
			u.Manifest.Physics.Constants.Disk,
			u.Manifest.Physics.Constants.Timeout,
		))

		s.Info("Laws:", fmt.Sprintf("Network: %s | Max processes: %d",
			u.Manifest.Physics.Laws.Network,
			u.Manifest.Physics.Laws.MaxProcesses,
		))

		if u.Workspace != "" || u.MindPath != "" {
			s.Blank()
			if u.Workspace != "" {
				s.Info("Workspace:", u.Workspace+" → /workspace")
			}
			if u.MindPath != "" {
				s.Info("Mind:", u.MindPath+" → /mind")
			}
		}

		s.Blank()
		return nil
	},
}
