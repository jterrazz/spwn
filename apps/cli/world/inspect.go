package world

import (
	"context"
	"fmt"

	"spwn.sh/packages/architect"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <world-id>",
	Short: "Inspect a running world - agents, workspaces, status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		worldID := args[0]
		s := newStepper(cmd)

		arc, err := architect.NewFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		u, err := arc.Inspect(ctx, worldID)
		if err != nil {
			return fmt.Errorf("world %s not found\n\n  List worlds with: spwn world list", worldID)
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
		s.Info("Laws:", "Network: bridge")

		if len(u.Workspaces) > 0 || u.Agent != "" || len(u.Agents) > 0 {
			s.Blank()
			if len(u.Workspaces) > 0 {
				s.Info("Workspaces:", fmt.Sprintf("%d mounted under /workspaces/", len(u.Workspaces)))
				for _, ws := range u.Workspaces {
					ro := ""
					if ws.ReadOnly {
						ro = " (ro)"
					}
					s.Info("  "+ws.Name+":", ws.Path+" → /workspaces/"+ws.Name+ro)
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
