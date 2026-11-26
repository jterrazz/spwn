package world

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/project"
)

// projectWith inits a project at <dir>/site whose spwn.yaml is manifest,
// beside an empty folder <dir>/outside, and returns the loaded project and
// both folders.
func projectWith(t *testing.T, manifest string) (p *project.Project, root, outside string) {
	t.Helper()
	dir := t.TempDir()
	root = filepath.Join(dir, "site")
	outside = filepath.Join(dir, "outside")
	for _, d := range []string{root, outside} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := project.Init(root, project.InitOpts{Name: "site"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := project.Find(root)
	if err != nil || p == nil {
		t.Fatalf("Find: %v", err)
	}
	return p, root, outside
}

func TestAWorldAgentThatClimbsIsRefusedTheSameWhetherItsTargetExists(t *testing.T) {
	const manifest = "version: 1\nname: site\nworlds:\n  neo:\n    agents: [\"../../../outside/x\"]\n"
	p, _, outside := projectWith(t, manifest)

	_, gone := resolveProjectWorld(p, "neo", "")
	if err := os.MkdirAll(filepath.Join(outside, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "x", "SOUL.md"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, there := resolveProjectWorld(p, "neo", "")

	if gone == nil || there == nil {
		t.Fatalf("a climbing agent name was taken: gone=%v there=%v", gone, there)
	}
	if gone.Error() != there.Error() {
		t.Fatalf("the refusal tells what is outside:\n gone:  %v\n there: %v", gone, there)
	}
	const want = `world "neo": agent name "../../../outside/x" is not one name: a name is not empty, "." or "..", and holds no /, :, \ or NUL`
	if gone.Error() != want {
		t.Fatalf("refusal = %q, want %q", gone, want)
	}
}

func TestAWorldNameThatClimbsIsRefused(t *testing.T) {
	p, _, _ := projectWith(t, "version: 1\nname: site\nworlds:\n  \"..\":\n    agents: []\n")
	if _, err := resolveProjectWorld(p, "..", ""); err == nil {
		t.Fatal(`world ".." was resolved`)
	}
}
