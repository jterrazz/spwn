package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	agentDomain "spwn.sh/core/agent"
	"spwn.sh/apps/cli/ui"
	"spwn.sh/core/foundation"
	"github.com/spf13/cobra"
)

func init() {
	// Removed: now handled by `spwn profile <name>`
}

var statsCmd = &cobra.Command{
	Use:   "stats <agent-name>",
	Short: "Show agent statistics and Mind layer summary",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		s := newStepper(cmd)

		if err := agentDomain.ValidateMind(name); err != nil {
			return fmt.Errorf("agent %q not found", name)
		}

		info, err := agentDomain.InspectAgent(name)
		if err != nil {
			return fmt.Errorf("cannot inspect agent: %w", err)
		}

		mindPath := agentDomain.AgentDir(name)

		// Agent creation time (from directory mtime)
		created := "unknown"
		if fi, err := os.Stat(mindPath); err == nil {
			created = fi.ModTime().Format("2006-01-02")
		}

		// Sessions
		sessions, _ := agentDomain.ListSessions(mindPath)
		sessionCount := len(sessions)

		// Unique worlds from sessions
		worldSet := make(map[string]struct{})
		for _, sess := range sessions {
			if sess.WorldID != "" {
				worldSet[sess.WorldID] = struct{}{}
			}
		}

		// Journal entries (all)
		journalEntries, _ := agentDomain.ListJournal(mindPath, 0)
		journalCount := len(journalEntries)

		// Total uptime from journal
		var totalUptime time.Duration
		for _, e := range journalEntries {
			totalUptime += e.Duration
		}

		// Also count unique worlds from journal
		for _, e := range journalEntries {
			if e.WorldID != "" {
				worldSet[e.WorldID] = struct{}{}
			}
		}

		// Last active from journal
		lastActive := "never"
		lastWorld := ""
		if len(journalEntries) > 0 {
			newest := journalEntries[0] // already sorted newest-first
			lastActive = newest.CreatedAt.Format("2006-01-02")
			lastWorld = newest.WorldID
		}

		// Display header
		s.Blank()
		s.Info("Agent:", name)
		s.Info("Created:", created)
		s.Blank()

		// Key stats
		s.Info("Sessions:", fmt.Sprintf("%d total", sessionCount))
		s.Info("Worlds:", fmt.Sprintf("%d unique", len(worldSet)))
		s.Info("Uptime:", ui.FormatDuration(totalUptime)+" total")
		s.Info("Journals:", fmt.Sprintf("%d entries", journalCount))
		if lastWorld != "" {
			s.Info("Last active:", fmt.Sprintf("%s (%s)", lastActive, lastWorld))
		} else {
			s.Info("Last active:", lastActive)
		}
		s.Blank()

		// Mind layers table
		t := ui.NewTable(ui.ModeNormal, "LAYER", "FILES", "SIZE")
		for _, layer := range foundation.MindLayers {
			files := info.Layers[layer]
			fileCount := len(files)

			var totalSize int64
			for _, f := range files {
				fpath := filepath.Join(info.Path, layer, f)
				if fi, err := os.Stat(fpath); err == nil {
					totalSize += fi.Size()
				}
			}

			countStr := fmt.Sprintf("%d", fileCount)
			if fileCount == 1 {
				countStr = "1 file"
			} else {
				countStr = fmt.Sprintf("%d files", fileCount)
			}

			t.AddRow(layer, countStr, formatSize(totalSize))
		}
		t.Render()

		return nil
	},
}
