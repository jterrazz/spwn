package world

import (
	"context"
	"fmt"
	"os/exec"
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

		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			return dockerHint(err)
		}

		worlds, err := arc.List(ctx)
		if err != nil {
			return fmt.Errorf("cannot list worlds: %w", err)
		}

		// Auto-cleanup: remove stale worlds whose containers no longer exist
		var liveWorlds []universe.World
		for _, w := range worlds {
			if w.ContainerID != "" && !containerExists(w.ContainerID) {
				// Container is gone — clean up state silently
				arc.Destroy(ctx, w.ID)
				continue
			}
			liveWorlds = append(liveWorlds, w)
		}
		worlds = liveWorlds

		if len(worlds) == 0 {
			s.Blank()
			s.Success("No active worlds.")
			s.Blank()
			return nil
		}

		t := ui.NewTable("ID", "CONFIG", "AGENTS", "STATUS", "CREATED")
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

// containerExists checks if a Docker container exists (running or stopped).
func containerExists(containerID string) bool {
	err := exec.Command("docker", "inspect", "--format", "{{.Id}}", containerID).Run()
	return err == nil
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
