package world

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active worlds",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		j, _ := cmd.Flags().GetBool("json")
		q, _ := cmd.Flags().GetBool("quiet")

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		worlds, err := arc.List(ctx)
		if err != nil {
			return fmt.Errorf("cannot list worlds: %w", err)
		}

		if j {
			data, _ := json.MarshalIndent(map[string]interface{}{"active": worlds}, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		if q {
			return nil
		}

		if len(worlds) == 0 {
			s.Blank()
			s.Success("No active worlds.")
			s.Blank()
			return nil
		}

		t := ui.NewTable(ui.ModeNormal, "ID", "CONFIG", "AGENTS", "STATUS", "CREATED")
		for _, u := range worlds {
			agents := collectAgentNames(u)
			config := u.Config
			if config == "" {
				config = "default"
			}
			t.AddRow(u.ID, config, agents, string(u.Status), timeAgo(u.CreatedAt))
		}
		t.Render()

		return nil
	},
}

// collectAgentNames returns a comma-separated list of agent names for a world.
func collectAgentNames(u universe.World) string {
	names := make([]string, 0)

	// Primary agent
	if u.Agent != "" {
		names = append(names, u.Agent)
	}

	// Multi-agent records (avoid duplicating the primary agent)
	for _, a := range u.Agents {
		if a.Name != u.Agent {
			names = append(names, a.Name)
		}
	}

	if len(names) == 0 {
		return "\u2014"
	}
	return strings.Join(names, ", ")
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "\u2014"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
