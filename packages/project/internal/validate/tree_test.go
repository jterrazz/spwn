package validate

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func openFiles(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// TestARuleRunAloneClosesTheTreeItOpens: a rule run with no Tree opens
// one at the project's folder and closes it when it returns, so a
// thousand runs leave no root open, with the collector off.
func TestARuleRunAloneClosesTheTreeItOpens(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", "focus.md"), []byte("---\nname: focus\ndescription: stay on it\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer debug.SetGCPercent(debug.SetGCPercent(-1))
	before := openFiles(t)
	for range 1000 {
		ruleSkillFrontmatter(Input{Root: root})
	}
	if after := openFiles(t); after > before+5 {
		t.Errorf("open files %d -> %d after 1000 runs of a rule alone; want each run's root closed", before, after)
	}
}
