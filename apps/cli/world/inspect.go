package world

import (
	"context"
	"encoding/json"
	"fmt"

	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <world-id>",
	Short: "Show world details, physics, and agent status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		j, _ := cmd.Flags().GetBool("json")

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return err
		}

		u, err := arc.Inspect(ctx, worldID)
		if err != nil {
			return fmt.Errorf("error: world %s not found.\nRun 'spwn world list' to see available worlds.", worldID)
		}

		if j {
			data, _ := json.MarshalIndent(u, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		s.Blank()
		s.Info("World:", u.ID)
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
