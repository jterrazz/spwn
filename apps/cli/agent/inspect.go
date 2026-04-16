package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/packages/platform"
	"spwn.sh/packages/agent"
)

func init() {
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:     "inspect <agent-name>",
	Aliases: []string{"show"},
	Short:   "Inspect an agent - composition, memory, world status, history",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := ui.New()

		info, err := agent.InspectAgent(name)
		if err != nil {
			return fmt.Errorf("agent %q not found", name)
		}

		s.Blank()
		s.Info("Agent:", info.Name)
		s.Info("Path:", info.Path)

		// Show world association
		agentMap := buildAgentWorldMap()
		if wInfo, ok := agentMap[name]; ok {
			s.Info("World:", wInfo.WorldID)
			s.Info("Status:", wInfo.Status)
		} else {
			s.Info("World:", "unattached")
		}

		s.Blank()

		// Mind layers with file sizes
		for _, layer := range platform.MindLayers {
			files := info.Layers[layer]
			if len(files) == 0 {
				s.Info(layer+"/", "(empty)")
			} else {
				// Show file names and sizes
				details := make([]string, 0, len(files))
				for _, f := range files {
					fpath := filepath.Join(info.Path, layer, f)
					fi, err := os.Stat(fpath)
					if err == nil {
						details = append(details, fmt.Sprintf("%s (%s)", f, formatSize(fi.Size())))
					} else {
						details = append(details, f)
					}
				}
				if len(details) <= 3 {
					s.Info(layer+"/", strings.Join(details, ", "))
				} else {
					s.Info(layer+"/", fmt.Sprintf("%d file(s)", len(files)))
					for _, d := range details {
						s.Info("", "  "+d)
					}
				}
			}
		}

		// The legacy "Sessions:" block used to list the same files as
		// the journal layer (sessions were stored there). That made
		// `agent inspect` render the same tree twice and confused
		// users about whether two sources existed (Finding #18). The
		// journal layer above is the single source of truth; suppress
		// the duplicate block entirely.

		// Show recent journal entries
		entries, err := agent.ListJournal(info.Path, 5)
		if err == nil && len(entries) > 0 {
			s.Blank()
			s.Info("Journal:", fmt.Sprintf("last %d entries", len(entries)))
			for _, e := range entries {
				ts := e.CreatedAt.Format("2006-01-02 15:04")
				s.Info(ts, fmt.Sprintf("%-24s %-10s %s", e.WorldID, e.Outcome, ui.FormatDuration(e.Duration)))
			}
		}

		s.Blank()

		return nil
	},
}

// formatSize formats a byte count into a human-readable string.
func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%dB", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1fK", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1fM", float64(bytes)/(1024*1024))
	}
}
