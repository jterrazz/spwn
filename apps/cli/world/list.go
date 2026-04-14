package world

import (
	"context"
	"encoding/json"
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

var listAsJSON bool

func init() {
	listCmd.Flags().BoolVar(&listAsJSON, "json", false, "Emit results as structured JSON on stdout")
	Cmd.AddCommand(listCmd)
}

// worldListReport is the CLI-owned JSON schema for `spwn world ls`.
// Kept separate from world.World so the JSON contract can evolve
// independently of the internal runtime type.
type worldListReport struct {
	Mode   string          `json:"mode"`
	Worlds []worldListRow  `json:"worlds"`
}

type worldListRow struct {
	Name   string   `json:"name"`
	Status string   `json:"status"`
	Agents []string `json:"agents"`
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
			return renderRunningOnly(ctx, s, cmd)
		}

		return renderDeclared(ctx, s, proj, cmd)
	},
}

func emitWorldListJSON(cmd *cobra.Command, report worldListReport) error {
	if report.Worlds == nil {
		report.Worlds = []worldListRow{}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// renderDeclared lists worlds declared in spwn.yaml with a status
// column derived from the live Docker query.
func renderDeclared(ctx context.Context, s *ui.Stepper, proj *manifest.Project, cmd *cobra.Command) error {
	declared := proj.Manifest.Worlds
	if len(declared) == 0 {
		if listAsJSON {
			return emitWorldListJSON(cmd, worldListReport{Mode: "project"})
		}
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

	if listAsJSON {
		return emitWorldListJSON(cmd, worldListReport{
			Mode:   "project",
			Worlds: buildDeclaredJSONRows(declared, running),
		})
	}

	t := ui.NewTable("WORLD", "STATUS", "AGENTS")
	for _, row := range buildDeclaredRows(declared, running, time.Now()) {
		t.AddRow(row[0], row[1], row[2])
	}
	t.Render()
	return nil
}

// buildDeclaredJSONRows is the JSON-facing version of buildDeclaredRows.
// It emits a plain status token ("running"/"stopped") without the
// human-formatted duration so snapshots stay machine-stable.
func buildDeclaredJSONRows(
	declared map[string]manifest.World,
	running map[string]world.World,
) []worldListRow {
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)

	rows := make([]worldListRow, 0, len(names))
	for _, name := range names {
		w := declared[name]
		agents := w.Agents
		if agents == nil {
			agents = []string{}
		}
		status := "stopped"
		if _, ok := running[name]; ok {
			status = "running"
		}
		rows = append(rows, worldListRow{
			Name:   name,
			Status: status,
			Agents: agents,
		})
	}
	return rows
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
func renderRunningOnly(ctx context.Context, s *ui.Stepper, cmd *cobra.Command) error {
	arc, err := world.NewArchitectFromEnv()
	if err != nil {
		if listAsJSON {
			return emitWorldListJSON(cmd, worldListReport{Mode: "global"})
		}
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
		if listAsJSON {
			return emitWorldListJSON(cmd, worldListReport{Mode: "global"})
		}
		s.Blank()
		s.Success("No active worlds.")
		s.Blank()
		return nil
	}

	if listAsJSON {
		rows := make([]worldListRow, 0, len(worlds))
		for _, u := range worlds {
			agents := collectAgentList(u)
			rows = append(rows, worldListRow{
				Name:   u.Config,
				Status: string(u.Status),
				Agents: agents,
			})
		}
		return emitWorldListJSON(cmd, worldListReport{Mode: "global", Worlds: rows})
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

// collectAgentList returns agent names as a slice (JSON-friendly).
// Mirrors collectAgentNames which returns the comma-joined string.
func collectAgentList(u world.World) []string {
	names := make([]string, 0)
	if u.Agent != "" {
		names = append(names, u.Agent)
	}
	for _, a := range u.Agents {
		if a.Name != u.Agent {
			names = append(names, a.Name)
		}
	}
	return names
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
