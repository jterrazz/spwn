package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"spwn.sh/core/universe"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the full status of your spwn environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := ui.New(quiet, verbose, jsonOutput)
		s.Blank()

		// Organization
		org, _ := universe.LoadOrg()
		if org != nil && org.Name != "" {
			s.Info("Organization:", org.Name)
		}
		s.Info("Home:", foundation.BaseDir())
		s.Blank()

		// Worlds
		arc, err := universe.NewArchitectFromEnv()
		if err != nil {
			s.Warn("Docker", "not available")
			return nil
		}

		worlds, err := arc.List(context.Background())
		if err != nil || len(worlds) == 0 {
			s.Log("No active worlds.")
		} else {
			t := ui.NewTable(ui.ModeNormal, "ID", "CONFIG", "AGENTS", "STATUS")
			for _, w := range worlds {
				agents := w.Agent
				if len(w.Agents) > 0 {
					names := make([]string, len(w.Agents))
					for i, a := range w.Agents {
						names[i] = a.Name
					}
					agents = strings.Join(names, ", ")
				}
				t.AddRow(w.ID, w.Config, agents, string(w.Status))
			}
			t.Render()
		}
		s.Blank()

		// Agents
		agentList, _ := agentDomain.ListAgents()
		if len(agentList) == 0 {
			s.Log("No agents.")
		} else {
			// Build world map
			worldMap := make(map[string]string)
			statusMap := make(map[string]string)
			if worlds != nil {
				for _, w := range worlds {
					if w.Agent != "" {
						worldMap[w.Agent] = w.ID
						statusMap[w.Agent] = string(w.Status)
					}
					for _, a := range w.Agents {
						worldMap[a.Name] = w.ID
						statusMap[a.Name] = string(a.Status)
					}
				}
			}

			t := ui.NewTable(ui.ModeNormal, "AGENT", "LAYERS", "WORLD", "STATUS")
			for _, a := range agentList {
				wid := worldMap[a.Name]
				st := statusMap[a.Name]
				if wid == "" {
					wid = "—"
					st = "unattached"
				}
				layers := fmt.Sprintf("%d/6", agentDomain.LayerCount(&a))
				t.AddRow(a.Name, layers, wid, st)
			}
			t.Render()
		}
		s.Blank()

		// Skills
		skillsDir := foundation.SkillsDir()
		skillCount := 0
		if entries, err := os.ReadDir(skillsDir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					skillCount++
				}
			}
		}

		// Auth
		authToken := ""
		if data, err := os.ReadFile(filepath.Join(foundation.BaseDir(), ".auth-token")); err == nil {
			authToken = strings.TrimSpace(string(data))
		}
		if authToken != "" {
			s.Done("Auth", "subscription (cached token)")
		} else if os.Getenv("ANTHROPIC_API_KEY") != "" {
			s.Done("Auth", "API key")
		} else {
			s.Warn("Auth", "not configured")
		}

		if skillCount > 0 {
			s.Done("Skills", fmt.Sprintf("%d installed", skillCount))
		}

		s.Blank()
		return nil
	},
}
