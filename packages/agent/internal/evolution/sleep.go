package evolution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/packages/activity"
)

// Sleep consolidates experience into durable knowledge.
// It archives stale files and prunes sessions for destroyed worlds.
//
// Knowledge is no longer agent-owned — it lives at /world/knowledge/
// and is committed to the project tree — so sleep only touches
// agent-scoped layers (playbooks, journal). The world knowledge base
// is maintained by humans via git, not pruned automatically.
func Sleep(mindPath string) (*SleepResult, error) {
	result := &SleepResult{Timestamp: time.Now()}

	// 1. Archive stale playbooks (older than 30 days, not recently referenced)
	archived, err := archiveStaleFiles(mindPath, "playbooks", 30*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("archiving playbooks: %w", err)
	}
	result.ArchivedPlaybooks = archived

	// 2. Prune old journal entries (sessions merged into journal)
	pruned, err := pruneOldSessions(mindPath, 90*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("pruning journal: %w", err)
	}
	result.PrunedSessions = pruned

	// 3. Write sleep log
	journalDir := filepath.Join(mindPath, "journal")
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return nil, fmt.Errorf("creating the journal layer: %w", err)
	}
	logPath := filepath.Join(journalDir, fmt.Sprintf("sleep-%s.md", time.Now().Format("2006-01-02")))
	summary := formatSleepSummary(result)
	if err := os.WriteFile(logPath, []byte(summary), 0644); err != nil {
		return nil, fmt.Errorf("writing the sleep log: %w", err)
	}

	// Emit activity event
	agentName := filepath.Base(mindPath)
	activity.Log(activity.Event{
		Type:    activity.TypeAgentSlept,
		Actor:   agentName,
		Verb:    "slept",
		Target:  agentName,
		Phrase:  activity.PhraseAgentSlept(agentName, result.ArchivedPlaybooks),
		AgentID: agentName,
		Metadata: map[string]any{
			"archived_playbooks": result.ArchivedPlaybooks,
			"pruned_sessions":    result.PrunedSessions,
		},
	})

	return result, nil
}

// SleepResult holds the outcome of a sleep cycle.
type SleepResult struct {
	ArchivedPlaybooks int
	PrunedSessions    int
	Timestamp         time.Time
}

func archiveStaleFiles(mindPath, layer string, maxAge time.Duration) (int, error) {
	layerDir := filepath.Join(mindPath, layer)
	archiveDir := filepath.Join(mindPath, "archive", layer)

	entries, err := os.ReadDir(layerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	cutoff := time.Now().Add(-maxAge)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.MkdirAll(archiveDir, 0755); err != nil {
				return count, fmt.Errorf("creating the archive layer: %w", err)
			}
			src := filepath.Join(layerDir, e.Name())
			dst := filepath.Join(archiveDir, e.Name())
			if err := os.Rename(src, dst); err != nil {
				continue
			}
			count++
		}
	}
	return count, nil
}

func pruneOldSessions(mindPath string, maxAge time.Duration) (int, error) {
	sessionsDir := filepath.Join(mindPath, "journal")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	cutoff := time.Now().Add(-maxAge)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(sessionsDir, e.Name()))
			count++
		}
	}
	return count, nil
}

func formatSleepSummary(r *SleepResult) string {
	var b strings.Builder
	b.WriteString("# Sleep Cycle\n\n")
	fmt.Fprintf(&b, "Date: %s\n\n", r.Timestamp.Format(time.RFC3339))
	fmt.Fprintf(&b, "- Archived playbooks: %d\n", r.ArchivedPlaybooks)
	fmt.Fprintf(&b, "- Pruned sessions: %d\n", r.PrunedSessions)
	return b.String()
}
