package resolve

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWalk_simpleImport(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	profile := filepath.Join(root, "core", "profile.md")
	mustWrite(t, profile, "# profile\n")
	mustWrite(t, claude, "# header\n\nread @core/profile.md for the profile\n")

	r, err := Walk(root, claude)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Visited) != 2 {
		t.Fatalf("Visited len = %d, want 2: %v", len(r.Visited), r.Visited)
	}
	if r.Visited[0] != claude || r.Visited[1] != profile {
		t.Errorf("visit order = %v, want [%s %s]", r.Visited, claude, profile)
	}
	if len(r.Missing) != 0 {
		t.Errorf("expected no missing, got: %v", r.Missing)
	}
}

func TestWalk_missingImportIsReported(t *testing.T) {
	root := t.TempDir()
	claude := filepath.Join(root, "CLAUDE.md")
	mustWrite(t, claude, "see @core/does-not-exist.md for details\n")

	r, err := Walk(root, claude)
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

	r, err := Walk(root, a)
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
	mustWrite(t, claude, "ping me at user@example.com — not an import\n")

	r, err := Walk(root, claude)
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

	r, err := Walk(root, claude)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if len(r.Missing) != 0 {
		t.Errorf("bare @foo should be ignored, got: %+v", r.Missing)
	}
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
