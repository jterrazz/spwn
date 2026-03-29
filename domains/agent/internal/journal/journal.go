package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry represents a parsed journal entry.
type Entry struct {
	Filename   string
	UniverseID string
	Outcome    string
	ExitCode   int
	Duration   time.Duration
	CreatedAt  time.Time
}

// Append writes a new journal entry to the Mind's journal directory.
func Append(mindPath, universeID string, exitCode int, duration time.Duration) error {
	dir := filepath.Join(mindPath, "journal")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create journal dir: %w", err)
	}

	now := time.Now()
	outcome := "completed"
	if exitCode != 0 {
		outcome = "failed"
	}

	filename := fmt.Sprintf("%s_%s.md", now.Format("2006-01-02_150405"), universeID)

	content := fmt.Sprintf(`# Session Journal

- **Universe:** %s
- **Outcome:** %s
- **Exit Code:** %d
- **Duration:** %s
- **Started:** %s
- **Ended:** %s
`, universeID, outcome, exitCode, formatDuration(duration), now.Add(-duration).Format(time.RFC3339), now.Format(time.RFC3339))

	path := filepath.Join(dir, filename)
	return os.WriteFile(path, []byte(content), 0644)
}

// List returns the last n journal entries, newest first.
// If n <= 0, returns all entries.
func List(mindPath string, n int) ([]Entry, error) {
	dir := filepath.Join(mindPath, "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var parsed []Entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		entry, err := parseEntry(dir, e.Name())
		if err != nil {
			continue
		}
		parsed = append(parsed, *entry)
	}

	// Sort newest first
	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].CreatedAt.After(parsed[j].CreatedAt)
	})

	if n > 0 && len(parsed) > n {
		parsed = parsed[:n]
	}
	return parsed, nil
}

func parseEntry(dir, filename string) (*Entry, error) {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return nil, err
	}

	entry := &Entry{Filename: filename}

	// Parse filename for timestamp: 2006-01-02_150405_universe-id.md
	parts := strings.SplitN(strings.TrimSuffix(filename, ".md"), "_", 3)
	if len(parts) == 3 {
		t, err := time.Parse("2006-01-02_150405", parts[0]+"_"+parts[1])
		if err == nil {
			entry.CreatedAt = t
		}
		entry.UniverseID = parts[2]
	}

	// Parse metadata from content
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- **Outcome:**") {
			entry.Outcome = strings.TrimSpace(strings.TrimPrefix(line, "- **Outcome:**"))
		}
		if strings.HasPrefix(line, "- **Exit Code:**") {
			fmt.Sscanf(strings.TrimPrefix(line, "- **Exit Code:**"), "%d", &entry.ExitCode)
		}
		if strings.HasPrefix(line, "- **Duration:**") {
			entry.Duration, _ = time.ParseDuration(strings.TrimSpace(strings.TrimPrefix(line, "- **Duration:**")))
		}
	}

	return entry, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
