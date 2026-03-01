package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Entry represents a single journal record from an agent session.
type Entry struct {
	UniverseID string
	Origin     string
	Outcome    string // "completed" or "failed"
	ExitCode   int
	Duration   time.Duration
	StartedAt  time.Time
	EndedAt    time.Time
}

// Append writes a journal entry as a markdown file.
// Returns the filename of the created entry.
func Append(mindPath string, e Entry) (string, error) {
	dir := filepath.Join(mindPath, "journal")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating journal directory: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.md",
		e.EndedAt.Format("2006-01-02_150405"),
		e.UniverseID,
	)

	content := fmt.Sprintf(`# Session %s

- **Universe:** %s
- **Origin:** %s
- **Outcome:** %s
- **Exit Code:** %d
- **Duration:** %s
- **Started:** %s
- **Ended:** %s
`,
		e.UniverseID,
		e.UniverseID,
		e.Origin,
		e.Outcome,
		e.ExitCode,
		e.Duration.Truncate(time.Second).String(),
		e.StartedAt.Format(time.RFC3339),
		e.EndedAt.Format(time.RFC3339),
	)

	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("writing journal entry: %w", err)
	}

	return filename, nil
}

// List returns the most recent N journal entries, newest first.
func List(mindPath string, n int) ([]Entry, error) {
	dir := filepath.Join(mindPath, "journal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading journal directory: %w", err)
	}

	// Filter to .md files and sort reverse-chronologically (filenames are timestamp-prefixed).
	var mdFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdFiles = append(mdFiles, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(mdFiles)))

	if n > 0 && len(mdFiles) > n {
		mdFiles = mdFiles[:n]
	}

	var result []Entry
	for _, name := range mdFiles {
		entry, err := parseEntry(filepath.Join(dir, name))
		if err != nil {
			continue // skip malformed entries
		}
		result = append(result, entry)
	}

	return result, nil
}

// parseEntry reads a journal markdown file and extracts structured metadata.
func parseEntry(path string) (Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Entry{}, err
	}

	var e Entry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if val, ok := parseField(line, "Universe:"); ok {
			e.UniverseID = val
		} else if val, ok := parseField(line, "Origin:"); ok {
			e.Origin = val
		} else if val, ok := parseField(line, "Outcome:"); ok {
			e.Outcome = val
		} else if val, ok := parseField(line, "Exit Code:"); ok {
			fmt.Sscanf(val, "%d", &e.ExitCode)
		} else if val, ok := parseField(line, "Duration:"); ok {
			e.Duration, _ = time.ParseDuration(val)
		} else if val, ok := parseField(line, "Started:"); ok {
			e.StartedAt, _ = time.Parse(time.RFC3339, val)
		} else if val, ok := parseField(line, "Ended:"); ok {
			e.EndedAt, _ = time.Parse(time.RFC3339, val)
		}
	}

	return e, nil
}

// parseField extracts the value from a markdown field line like "- **Key:** value".
func parseField(line, key string) (string, bool) {
	prefix := fmt.Sprintf("- **%s**", key)
	if strings.HasPrefix(line, prefix) {
		val := strings.TrimPrefix(line, prefix)
		val = strings.TrimSpace(val)
		return val, true
	}
	return "", false
}
