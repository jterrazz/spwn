package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spwn.sh/apps/cli/ui"
	agentDomain "spwn.sh/core/agent"
	"spwn.sh/core/foundation"
	"github.com/spf13/cobra"
)

var inspectJSON bool

func init() {
	inspectCmd.Flags().BoolVar(&inspectJSON, "json", false, "Output as JSON")
	Cmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect <agent-name>",
	Short: "Show agent details, Mind layers, universe status, and history",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		info, err := agentDomain.InspectAgent(name)
		if err != nil {
			return fmt.Errorf("error: agent %q not found.\nRun 'spwn agent list' to see available agents.", name)
		}

		if inspectJSON {
			data, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		s.Blank()
		s.Info("Agent:", info.Name)
		s.Info("Path:", info.Path)

		// Show universe association
		agentMap := buildAgentUniverseMap()
		if uInfo, ok := agentMap[name]; ok {
			s.Info("Universe:", uInfo.UniverseID)
			s.Info("Status:", uInfo.Status)
		} else {
			s.Info("Universe:", "unattached")
		}

		s.Blank()

		// Mind layers with file sizes
		for _, layer := range foundation.MindLayers {
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

		// Show session files
		sessDir := filepath.Join(info.Path, "sessions")
		if entries, err := os.ReadDir(sessDir); err == nil && len(entries) > 0 {
			s.Blank()
			s.Info("Sessions:", fmt.Sprintf("%d file(s)", len(entries)))
			for _, e := range entries {
				fi, _ := e.Info()
				if fi != nil {
					s.Info("", fmt.Sprintf("  %s (%s)", e.Name(), formatSize(fi.Size())))
				}
			}
		}

		// Show recent journal entries
		entries, err := agentDomain.ListJournal(info.Path, 5)
		if err == nil && len(entries) > 0 {
			s.Blank()
			s.Info("Journal:", fmt.Sprintf("last %d entries", len(entries)))
			for _, e := range entries {
				ts := e.CreatedAt.Format("2006-01-02 15:04")
				s.Info(ts, fmt.Sprintf("%-24s %-10s %s", e.UniverseID, e.Outcome, ui.FormatDuration(e.Duration)))
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
