package evolution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"spwn.sh/packages/agent/internal/journal"
	"spwn.sh/packages/foundation/activity"
)

// Dream analyzes recent journal entries and promotes successful patterns to playbooks.
// This is the primary function - formerly called Reflect.
func Dream(mindPath string) (*ReflexionResult, error) {
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

	// Count outcomes: 0 = success, -1 = destroyed (neutral), >0 = failed
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

	// 3. Write reflexion summary to playbooks
	playbooksDir := filepath.Join(mindPath, "playbooks")
	os.MkdirAll(playbooksDir, 0755)

	summary := formatReflexionSummary(result, entries)
	outPath := filepath.Join(playbooksDir, "auto-reflexion.md")
	if err := os.WriteFile(outPath, []byte(summary), 0644); err != nil {
		return nil, fmt.Errorf("cannot write reflexion: %w", err)
	}
	result.OutputPath = outPath

	// Emit activity event. Agent name = last path component of mindPath.
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
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", r.Timestamp.Format(time.RFC3339)))
	b.WriteString("## Statistics\n\n")
	b.WriteString(fmt.Sprintf("- Entries analyzed: %d\n", r.EntriesAnalyzed))
	b.WriteString(fmt.Sprintf("- Completed: %d\n", r.CompletedTasks))
	b.WriteString(fmt.Sprintf("- Failed: %d\n", r.FailedTasks))
	b.WriteString(fmt.Sprintf("- Destroyed: %d\n", r.DestroyedTasks))
	b.WriteString(fmt.Sprintf("- Success rate: %.0f%%\n\n", r.SuccessRate*100))

	// List recent sessions
	b.WriteString("## Recent Sessions\n\n")
	for _, e := range entries {
		outcome := "completed"
		if e.ExitCode < 0 {
			outcome = "destroyed"
		} else if e.ExitCode > 0 {
			outcome = fmt.Sprintf("failed (exit %d)", e.ExitCode)
		}
		b.WriteString(fmt.Sprintf("- %s: %s (%s)\n", e.WorldID, outcome, e.Duration))
	}

	return b.String()
}

// Reflect is a backward-compatible alias for Dream.
func Reflect(mindPath string) (*ReflexionResult, error) {
	return Dream(mindPath)
}
