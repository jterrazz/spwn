package world

import (
	"context"
	"fmt"

	"spwn.sh/packages/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <world-id>",
	Short: "Inspect a running world — physics, agents, status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		u, err := arc.Inspect(ctx, worldID)
		if err != nil {
			return fmt.Errorf("world %s not found\n\n  List worlds with: spwn ls", worldID)
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

		s.Info("Laws:", "Network: bridge")

		if len(u.Workspaces) > 0 || u.Agent != "" || len(u.Agents) > 0 {
			s.Blank()
			if len(u.Workspaces) > 0 {
				s.Info("Workspaces:", fmt.Sprintf("%d mounted at /work/*", len(u.Workspaces)))
				for _, ws := range u.Workspaces {
					ro := ""
					if ws.ReadOnly {
						ro = " (ro)"
					}
					s.Info("  "+ws.Name+":", ws.Path+" → /work/"+ws.Name+ro)
				}
			}
			// Agent homes are visible at /agents/<name>; per-world data
			// (inbox, notes) at /agents/<name>/worlds/<world-id>/.
			if u.Agent != "" {
				s.Info("Agent home:", "~/.spwn/agents/"+u.Agent+" → /agents/"+u.Agent)
			}
			for _, rec := range u.Agents {
				s.Info("  "+rec.Name+":", "~/.spwn/agents/"+rec.Name+" → /agents/"+rec.Name)
			}
		}

		s.Blank()
		return nil
	},
}
