package evolution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/core/agent/internal/journal"
)

// Reflect analyzes recent journal entries and promotes successful patterns to playbooks.
func Reflect(mindPath string) (*ReflexionResult, error) {
	// 1. Read journal entries
	entries, err := journal.List(mindPath, 20)
	if err != nil {
		return nil, fmt.Errorf("cannot read journal: %w", err)
	}
	if len(entries) == 0 {
		return &ReflexionResult{Skipped: true, Reason: "no journal entries"}, nil
	}

	// 2. Analyze patterns
	result := &ReflexionResult{
		EntriesAnalyzed: len(entries),
		Timestamp:       time.Now(),
	}

	// Count outcomes
	completed := 0
	failed := 0
	for _, e := range entries {
		if e.ExitCode == 0 {
			completed++
		} else {
			failed++
		}
	}
	result.CompletedTasks = completed
	result.FailedTasks = failed
	result.SuccessRate = float64(completed) / float64(len(entries))

	// 3. Write reflexion summary to playbooks
	playbooksDir := filepath.Join(mindPath, "memory", "playbooks")
	os.MkdirAll(playbooksDir, 0755)

	summary := formatReflexionSummary(result, entries)
	outPath := filepath.Join(playbooksDir, "auto-reflexion.md")
	if err := os.WriteFile(outPath, []byte(summary), 0644); err != nil {
		return nil, fmt.Errorf("cannot write reflexion: %w", err)
	}
	result.OutputPath = outPath

	return result, nil
}

// ReflexionResult holds the outcome of a reflexion analysis.
type ReflexionResult struct {
	Skipped         bool
	Reason          string
	EntriesAnalyzed int
	CompletedTasks  int
	FailedTasks     int
	SuccessRate     float64
	OutputPath      string
	Timestamp       time.Time
}

func formatReflexionSummary(r *ReflexionResult, entries []journal.Entry) string {
	var b strings.Builder
	b.WriteString("# Reflexion Summary\n\n")
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.Timestamp.Format(time.RFC3339)))
	b.WriteString("## Statistics\n\n")
	b.WriteString(fmt.Sprintf("- Entries analyzed: %d\n", r.EntriesAnalyzed))
	b.WriteString(fmt.Sprintf("- Completed: %d\n", r.CompletedTasks))
	b.WriteString(fmt.Sprintf("- Failed: %d\n", r.FailedTasks))
	b.WriteString(fmt.Sprintf("- Success rate: %.0f%%\n\n", r.SuccessRate*100))

	// List recent sessions
	b.WriteString("## Recent Sessions\n\n")
	for _, e := range entries {
		outcome := "completed"
		if e.ExitCode != 0 {
			outcome = fmt.Sprintf("failed (exit %d)", e.ExitCode)
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%s)\n", e.WorldID, outcome, e.Duration))
	}

	return b.String()
}
