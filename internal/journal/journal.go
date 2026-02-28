package journal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jterrazz/universe/internal/mind"
)

// Entry represents a single journal entry for a universe session.
type Entry struct {
	UniverseID string
	Image      string
	Outcome    string // "completed" or "failed"
	ExitCode   int
	Duration   time.Duration
	Timestamp  time.Time
}

// Append writes a journal entry as a markdown file in the mind's journal directory.
// Filename format: YYYY-MM-DD_HHMMSS_{shortID}.md
func Append(mindID string, entry Entry) error {
	mindPath, err := mind.ResolvePath(mindID)
	if err != nil {
		return err
	}
	journalDir := filepath.Join(mindPath, "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		return fmt.Errorf("creating journal directory: %w", err)
	}

	ts := entry.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	shortID := entry.UniverseID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	filename := fmt.Sprintf("%s_%s.md", ts.Format("2006-01-02_150405"), shortID)
	path := filepath.Join(journalDir, filename)

	content := fmt.Sprintf(`# Session %s

- **Universe:** %s
- **Image:** %s
- **Outcome:** %s
- **Exit Code:** %d
- **Duration:** %s
- **Timestamp:** %s
`, shortID, entry.UniverseID, entry.Image, entry.Outcome, entry.ExitCode, entry.Duration, ts.Format(time.RFC3339))

	return os.WriteFile(path, []byte(content), 0o644)
}

// List returns journal filenames for a given mind, sorted chronologically.
func List(mindID string) ([]string, error) {
	mindPath, err := mind.ResolvePath(mindID)
	if err != nil {
		return nil, err
	}
	journalDir := filepath.Join(mindPath, "journal")
	entries, err := os.ReadDir(journalDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading journal directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
