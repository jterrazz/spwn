package world

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/manifest"
	"spwn.sh/packages/world"
)

func init() {
	Cmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List declared worlds and their running status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		s := newStepper(cmd)

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		proj, err := manifest.Find(cwd)
		if err != nil {
			return err
		}

		// Legacy global mode: no project discovered → keep the old
		// Docker-only listing as a fallback.
		if proj == nil || proj.Manifest == nil {
			return renderRunningOnly(ctx, s)
		}

		return renderDeclared(ctx, s, proj)
	},
}

// renderDeclared lists worlds declared in spwn.yaml with a status
// column derived from the live Docker query.
func renderDeclared(ctx context.Context, s *ui.Stepper, proj *manifest.Project) error {
	declared := proj.Manifest.Worlds
	if len(declared) == 0 {
		s.Blank()
		s.Success("No worlds declared in spwn.yaml.")
		s.Blank()
		return nil
	}

	// Live world state, keyed by world name (Config). A world may have
	// multiple running instances historically; we pick the newest.
	running := map[string]world.World{}
	if arc, err := world.NewArchitectFromEnv(); err == nil {
		if ws, err := arc.List(ctx); err == nil {
			for _, w := range ws {
				// Skip stale entries whose containers no longer exist.
				if w.ContainerID != "" && !containerExists(w.ContainerID) {
					arc.Destroy(ctx, w.ID)
					continue
				}
				if w.Status != world.StatusRunning && w.Status != world.StatusIdle {
					continue
				}
				key := w.Config
				if prev, ok := running[key]; !ok || w.CreatedAt.After(prev.CreatedAt) {
					running[key] = w
				}
			}
		}
	}

	t := ui.NewTable("WORLD", "STATUS", "AGENTS")
	for _, row := range buildDeclaredRows(declared, running, time.Now()) {
		t.AddRow(row[0], row[1], row[2])
	}
	t.Render()
	return nil
}

// buildDeclaredRows is the pure table-row construction used by
// renderDeclared. It takes the declared-world map, a map of live
// worlds keyed by name, and a "now" timestamp (for deterministic
// running-duration formatting in tests). Rows are returned sorted by
// world name.
func buildDeclaredRows(
	declared map[string]manifest.World,
	running map[string]world.World,
	now time.Time,
) [][3]string {
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([][3]string, 0, len(names))
	for _, name := range names {
		w := declared[name]
		agents := "\u2014"
		if len(w.Agents) > 0 {
			agents = strings.Join(w.Agents, ", ")
		}
		status := "\u25cb stopped"
		if live, ok := running[name]; ok {
			status = fmt.Sprintf("\u25cf running (%s)", durationSince(now, live.CreatedAt))
		}
		rows = append(rows, [3]string{name, status, agents})
	}
	return rows
}

// renderRunningOnly keeps the legacy "list live containers" rendering
// used when no spwn project is discovered.
func renderRunningOnly(ctx context.Context, s *ui.Stepper) error {
	arc, err := world.NewArchitectFromEnv()
	if err != nil {
		return dockerHint(err)
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return fmt.Errorf("cannot list worlds: %w", err)
	}

	var liveWorlds []world.World
	for _, w := range worlds {
		if w.ContainerID != "" && !containerExists(w.ContainerID) {
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
}

// collectAgentNames returns a comma-separated list of agent names for a world.
func collectAgentNames(u world.World) string {
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

// durationSince formats the "how long it has been up" string for the
// STATUS column, e.g. "3m", "2h", "5d". Takes an explicit "now" so
// tests can be deterministic.
func durationSince(now, t time.Time) string {
	if t.IsZero() {
		return "?"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
