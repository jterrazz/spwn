package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/project"
	"spwn.sh/packages/agent"
	"spwn.sh/packages/architect"
	"spwn.sh/packages/world"
)

var (
	listFilterWorld string
	listAsJSON      bool
)

func init() {
	listCmd.Flags().StringVar(&listFilterWorld, "world", "", "Filter agents by world ID")
	listCmd.Flags().BoolVar(&listAsJSON, "json", false, "Emit results as structured JSON on stdout")
	Cmd.AddCommand(listCmd)
}

// agentListReport is the CLI-owned JSON schema for `spwn agent ls`.
type agentListReport struct {
	Mode   string          `json:"mode"`
	Agents []agentListRow  `json:"agents"`
}

type agentListRow struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	World  string `json:"world,omitempty"`
}

func emitAgentListJSON(cmd *cobra.Command, report agentListReport) error {
	if report.Agents == nil {
		report.Agents = []agentListRow{}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	return nil
}

// agentWorldInfo holds cross-referenced world data for an agent.
type agentWorldInfo struct {
	WorldID string
	Status  string
}

// LsCmd is exported so the root command can reuse the agent-centric
// smart listing as the top-level `spwn ls` shortcut when a project
// is active.
var LsCmd = listCmd

var listCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List all agents on this Host",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Project-aware mode: three-column AGENT | STATUS | WORLD
		// view driven by spwn.yaml + live runtime state. Falls back
		// to the legacy global view when no project is present.
		if cwd, werr := os.Getwd(); werr == nil {
			if p, perr := project.Find(cwd); perr == nil && p != nil {
				return renderSmartAgentList(cmd, p)
			}
		}

		agents, err := agent.ListAgents()
		if err != nil {
			return fmt.Errorf("cannot list agents: %w", err)
		}

		// Build agent -> world mapping from state
		agentMap := buildAgentWorldMap()

		// Filter by world if requested
		if listFilterWorld != "" {
			filtered := make([]agent.Info, 0)
			for _, a := range agents {
				if info, ok := agentMap[a.Name]; ok && info.WorldID == listFilterWorld {
					filtered = append(filtered, a)
				}
			}
			agents = filtered
		}

		s := newStepper(cmd)

		if len(agents) == 0 {
			if listAsJSON {
				return emitAgentListJSON(cmd, agentListReport{Mode: "global"})
			}
			s.Blank()
			s.Success("No agents yet.")
			s.Log("Create one with: spwn agent new <name>")
			s.Blank()
			return nil
		}

		if listAsJSON {
			rows := make([]agentListRow, 0, len(agents))
			for _, a := range agents {
				status := "unattached"
				worldID := ""
				if info, ok := agentMap[a.Name]; ok {
					worldID = info.WorldID
					status = info.Status
				}
				rows = append(rows, agentListRow{Name: a.Name, Status: status, World: worldID})
			}
			return emitAgentListJSON(cmd, agentListReport{Mode: "global", Agents: rows})
		}

		// Unified schema: AGENT | STATUS | WORLD. Matches the
		// project-mode renderer so the header row stays stable no
		// matter where the user runs `spwn agent ls`. See finding #24.
		t := ui.NewTable("AGENT", "STATUS", "WORLD")
		for _, a := range agents {
			worldID := "\u2014"
			status := "unattached"

			if info, ok := agentMap[a.Name]; ok {
				worldID = info.WorldID
				status = info.Status
			}

			t.AddRow(a.Name, status, worldID)
		}
		t.Render()

		return nil
	},
}

// renderSmartAgentList renders the project-aware AGENT | STATUS |
// WORLD view. Status values:
//
//	"● running (<dur>)"      — the agent's world is currently up
//	"○ deployed, stopped"    — declared in spwn.yaml but not running
//	"─ orphan"               — on-disk agent not referenced by any world
func renderSmartAgentList(cmd *cobra.Command, p *project.Project) error {
	// Build declared-agent → world map from the manifest.
	declared := map[string]string{}
	if p.Manifest != nil {
		for wname, wdef := range p.Manifest.Worlds {
			for _, a := range wdef.Agents {
				declared[a] = wname
			}
		}
	}

	// Gather on-disk agents (deployable + orphan) into a single set,
	// so the view lines up with `spwn/agents/` regardless of whether
	// spwn.yaml references them.
	allAgents := map[string]bool{}
	for _, a := range p.Agents {
		allAgents[a.Name] = true
	}
	for _, a := range p.OrphanAgents {
		allAgents[a.Name] = true
	}
	for a := range declared {
		// Agents declared in spwn.yaml but missing on disk still
		// surface in the list (as orphan/stopped depending on the
		// rest of the rules below).
		allAgents[a] = true
	}

	// Live world → duration lookup. We match a live world to a
	// declared world by config name (set by world spawn).
	running := map[string]time.Duration{}
	if arc, aerr := architect.NewFromEnv(); aerr == nil {
		ctx := context.Background()
		if worlds, lerr := arc.List(ctx); lerr == nil {
			for _, u := range worlds {
				if u.Status == world.StatusDestroyed {
					continue
				}
				if u.Config != "" {
					running[u.Config] = time.Since(u.CreatedAt)
				}
			}
		}
	}

	s := newStepper(cmd)

	if len(allAgents) == 0 {
		if listAsJSON {
			return emitAgentListJSON(cmd, agentListReport{Mode: "project"})
		}
		s.Blank()
		s.Success("No agents yet.")
		s.Log("Create one with: spwn agent new <name>")
		s.Blank()
		return nil
	}

	// Stable order: alpha by name. Matches the table order.
	names := make([]string, 0, len(allAgents))
	for name := range allAgents {
		names = append(names, name)
	}
	sortStrings(names)

	if listAsJSON {
		rows := make([]agentListRow, 0, len(names))
		for _, name := range names {
			wname, isDeployed := declared[name]
			var status, worldCol string
			switch {
			case isDeployed && running[wname] > 0:
				status = "running"
				worldCol = wname
			case isDeployed:
				status = "stopped"
				worldCol = wname
			default:
				status = "orphan"
			}
			rows = append(rows, agentListRow{Name: name, Status: status, World: worldCol})
		}
		return emitAgentListJSON(cmd, agentListReport{Mode: "project", Agents: rows})
	}

	t := ui.NewTable("AGENT", "STATUS", "WORLD")
	for _, name := range names {
		wname, isDeployed := declared[name]
		var status, worldCol string
		switch {
		case isDeployed && running[wname] > 0:
			status = fmt.Sprintf("\u25cf running (%s)", shortDur(running[wname]))
			worldCol = wname
		case isDeployed:
			status = "\u25cb deployed, stopped"
			worldCol = wname
		default:
			status = "\u2500 orphan"
			worldCol = "\u2014"
		}
		t.AddRow(name, status, worldCol)
	}
	t.Render()
	return nil
}

// shortDur formats a duration like "3m", "2h17m", "1d" - good enough
// for a status column without pulling in a humanization library.
func shortDur(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// sortStrings is a tiny in-place sort - avoids pulling in "sort" at
// the top of this file's imports which are already getting crowded.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// buildAgentWorldMap reads state.json and returns a map of agent name
// to the world it is currently attached to.
func buildAgentWorldMap() map[string]agentWorldInfo {
	result := make(map[string]agentWorldInfo)

	ctx := context.Background()
	arc, err := architect.NewFromEnv()
	if err != nil {
		return result
	}

	worlds, err := arc.List(ctx)
	if err != nil {
		return result
	}

	for _, u := range worlds {
		if u.Status == world.StatusDestroyed {
			continue
		}

		// Check primary agent
		if u.Agent != "" {
			result[u.Agent] = agentWorldInfo{
				WorldID: u.ID,
				Status:  string(u.Status),
			}
		}

		// Check multi-agent records
		for _, a := range u.Agents {
			result[a.Name] = agentWorldInfo{
				WorldID: u.ID,
				Status:  string(a.Status),
			}
		}
	}

	return result
}
