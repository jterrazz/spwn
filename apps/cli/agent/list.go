package agent

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/manifest"
	"spwn.sh/packages/mind"
	"spwn.sh/packages/world"
)

var listFilterWorld string

func init() {
	listCmd.Flags().StringVar(&listFilterWorld, "world", "", "Filter agents by world ID")
	Cmd.AddCommand(listCmd)
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
			if p, perr := manifest.Find(cwd); perr == nil && p != nil {
				return renderSmartAgentList(cmd, p)
			}
		}

		agents, err := mind.ListAgents()
		if err != nil {
			return fmt.Errorf("cannot list agents: %w", err)
		}

		// Build agent -> world mapping from state
		agentMap := buildAgentWorldMap()

		// Filter by world if requested
		if listFilterWorld != "" {
			filtered := make([]mind.Info, 0)
			for _, a := range agents {
				if info, ok := agentMap[a.Name]; ok && info.WorldID == listFilterWorld {
					filtered = append(filtered, a)
				}
			}
			agents = filtered
		}

		s := newStepper(cmd)

		if len(agents) == 0 {
			s.Blank()
			s.Success("No agents yet.")
			s.Log("Create one with: spwn agent new <name>")
			s.Blank()
			return nil
		}

		t := ui.NewTable("NAME", "WORLD", "STATUS")
		for _, a := range agents {
			worldID := "\u2014"
			status := "unattached"

			if info, ok := agentMap[a.Name]; ok {
				worldID = info.WorldID
				status = info.Status
			}

			t.AddRow(a.Name, worldID, status)
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
func renderSmartAgentList(cmd *cobra.Command, p *manifest.Project) error {
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
	if arc, aerr := world.NewArchitectFromEnv(); aerr == nil {
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
		s.Blank()
		s.Success("No agents yet.")
		s.Log("Create one with: spwn agent new <name>")
		s.Blank()
		return nil
	}

	t := ui.NewTable("AGENT", "STATUS", "WORLD")
	// Stable order: deployed first (alpha by name), then orphans.
	names := make([]string, 0, len(allAgents))
	for name := range allAgents {
		names = append(names, name)
	}
	sortStrings(names)
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
	arc, err := world.NewArchitectFromEnv()
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
