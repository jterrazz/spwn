package evolution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/packages/activity"
	"spwn.sh/packages/agent/internal/journal"
)

// Dream analyzes recent journal entries and promotes successful patterns to playbooks.
// This is the primary function - formerly called Reflect.
func Dream(mindPath string) (*ReflexionResult, error) {
	entries, err := journal.List(mindPath, 20)
	if err != nil {
		return nil, fmt.Errorf("cannot read journal: %w", err)
	}
	if len(entries) == 0 {
		return &ReflexionResult{Skipped: true, Reason: "no journal entries"}, nil
	}

	result := &ReflexionResult{
		EntriesAnalyzed: len(entries),
		Timestamp:       time.Now(),
	}

	completed := 0
	failed := 0
	destroyed := 0
	for _, e := range entries {
		switch {
		case e.ExitCode == 0:
			completed++
		case e.ExitCode < 0:
			destroyed++
		default:
			failed++
		}
	}
	result.CompletedTasks = completed
	result.FailedTasks = failed
	result.DestroyedTasks = destroyed
	scorable := completed + failed
	if scorable > 0 {
		result.SuccessRate = float64(completed) / float64(scorable)
	}

	playbooksDir := filepath.Join(mindPath, "playbooks")
	if err := os.MkdirAll(playbooksDir, 0755); err != nil {
		return nil, fmt.Errorf("creating the playbooks layer: %w", err)
	}

	summary := formatReflexionSummary(result, entries)
	outPath := filepath.Join(playbooksDir, "auto-reflexion.md")
	if err := os.WriteFile(outPath, []byte(summary), 0644); err != nil {
		return nil, fmt.Errorf("cannot write reflexion: %w", err)
	}
	result.OutputPath = outPath

	agentName := filepath.Base(mindPath)
	activity.Log(activity.Event{
		Type:    activity.TypeAgentDreamed,
		Actor:   agentName,
		Verb:    "dreamed",
		Target:  agentName,
		Phrase:  activity.PhraseAgentDreamed(agentName, completed),
		AgentID: agentName,
		Metadata: map[string]any{
			"entries_analyzed": result.EntriesAnalyzed,
			"success_rate":     result.SuccessRate,
		},
	})

	return result, nil
}

// ReflexionResult holds the outcome of a reflexion analysis.
type ReflexionResult struct {
	Skipped         bool
	Reason          string
	EntriesAnalyzed int
	CompletedTasks  int
	FailedTasks     int
	DestroyedTasks  int
	SuccessRate     float64
	OutputPath      string
	Timestamp       time.Time
}

func formatReflexionSummary(r *ReflexionResult, entries []journal.Entry) string {
	var b strings.Builder
	b.WriteString("# Reflexion Summary\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", r.Timestamp.Format(time.RFC3339))
	b.WriteString("## Statistics\n\n")
	fmt.Fprintf(&b, "- Entries analyzed: %d\n", r.EntriesAnalyzed)
	fmt.Fprintf(&b, "- Completed: %d\n", r.CompletedTasks)
	fmt.Fprintf(&b, "- Failed: %d\n", r.FailedTasks)
	fmt.Fprintf(&b, "- Destroyed: %d\n", r.DestroyedTasks)
	fmt.Fprintf(&b, "- Success rate: %.0f%%\n\n", r.SuccessRate*100)

	b.WriteString("## Recent Sessions\n\n")
	for _, e := range entries {
		if e.WorldID == "" {
			continue
		}
		outcome := "completed"
		if e.ExitCode < 0 {
			outcome = "destroyed"
		} else if e.ExitCode > 0 {
			outcome = fmt.Sprintf("failed (exit %d)", e.ExitCode)
		}
		fmt.Fprintf(&b, "- %s: %s (%s)\n", e.WorldID, outcome, e.Duration)
	}

	return b.String()
}

// Reflect is a backward-compatible alias for Dream.
func Reflect(mindPath string) (*ReflexionResult, error) {
	return Dream(mindPath)
}
