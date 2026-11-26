package resolve

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"spwn.sh/packages/platform"
)

func TestWalk_simpleImport(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	soul := filepath.Join(root, "SOUL.md")
	mustWrite(t, soul, "# soul\n")
	mustWrite(t, claude, "# header\n\nread @SOUL.md for the agent's soul\n")

	r, err := walk(t, root, claude)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Visited) != 2 {
		t.Fatalf("Visited len = %d, want 2: %v", len(r.Visited), r.Visited)
	}
	if r.Visited[0] != claude || r.Visited[1] != soul {
		t.Errorf("visit order = %v, want [%s %s]", r.Visited, claude, soul)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got: %v", r.Missing)
	}
}

func TestWalk_missingImportIsReported(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	mustWrite(t, claude, "see @core/does-not-exist.md for details\n")

	r, err := walk(t, root, claude)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Missing) != 1 {
		t.Fatalf("expected 1 missing reference, got %d: %+v", len(r.Missing), r.Missing)
	}
	if r.Missing[0].Target != "core/does-not-exist.md" {
		t.Errorf("target = %q, want core/does-not-exist.md", r.Missing[0].Target)
	}
}

func TestWalk_detectsCycle(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a.md")
	b := filepath.Join(root, "b.md")
	mustWrite(t, a, "see @b.md\n")
	mustWrite(t, b, "see @a.md\n")

	r, err := walk(t, root, a)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Cycles) == 0 {
		t.Fatal("expected a cycle to be detected")
	}
}

func TestWalk_ignoresEmailAddresses(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	mustWrite(t, claude, "ping me at user@example.com - not an import\n")

	r, err := walk(t, root, claude)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Visited) != 1 {
		t.Errorf("expected just CLAUDE.md visited, got %v", r.Visited)
	}
	if len(r.Missing) != 0 {
		t.Errorf("email should not count as a broken import, got: %+v", r.Missing)
	}
}

func TestWalk_requiresMdExtension(t *testing.T) {
	// @foo is not an import; @foo.md is.
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	mustWrite(t, claude, "tag @someone and mention @other things\n")

	r, err := walk(t, root, claude)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("bare @foo should be ignored, got: %+v", r.Missing)
	}
}

// TestWalk_anImportThatLeavesTheProjectIsNeverRead: an @-import that
// climbs out of the project, or names an absolute path elsewhere, is
// Outside, and the file there is never read: neither its @-tokens nor
// its existence reach the result.
func TestWalk_anImportThatLeavesTheProjectIsNeverRead(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "project")
	agent := filepath.Join(project, "spwn", "agents", "neo")
	entry := filepath.Join(agent, "AGENTS.md")
	outside := filepath.Join(base, "outside", "notes.md")
	mustWrite(t, outside, "@secret-token.md\n")
	mustWrite(t, entry, "read @../../../../outside/notes.md and @"+outside+"\n")

	tree := platform.Within(project)
	defer tree.Close()
	r, err := Walk(tree, agent, entry)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Outside) != 2 {
		t.Fatalf("Outside = %+v, want the two imports", r.Outside)
	}
	if len(r.Visited) != 1 || len(r.Missing) != 0 {
		t.Errorf("Visited = %v, Missing = %+v: nothing outside may be read", r.Visited, r.Missing)
	}
}

// TestWalk_aFIFOAtAnImportIsNeverWaitedOn: a FIFO an import names is
// not a file: Walk returns at once and lists it missing.
func TestWalk_aFIFOAtAnImportIsNeverWaitedOn(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	mustWrite(t, claude, "read @pipe.md\n")
	if err := syscall.Mkfifo(filepath.Join(root, "pipe.md"), 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan *Result, 1)
	go func() {
		r, _ := walk(t, root, claude)
		done <- r
	}()
	select {
	case r := <-done:
		if r == nil || len(r.Missing) != 1 {
			t.Errorf("Missing = %+v, want the FIFO", r)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Walk waited on a FIFO")
	}
}

func walk(t *testing.T, root, start string) (*Result, error) {
	t.Helper()
	tree := platform.Within(root)
	defer tree.Close()
	return Walk(tree, root, start)
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
