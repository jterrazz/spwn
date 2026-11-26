package architecture_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

// invariant: the System Settings panes the window's shell opens are the
// panes the engine's catalog names, and no others: the shell's list and
// catalog.Panes are one list, written twice because a Rust build reads no
// Go file.
func TestThePanesTheShellOpensAreTheCatalogs(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile(filepath.Join(repoRoot(t), "apps", "desktop", "src-tauri", "src", "opener.rs"))
	if err != nil {
		t.Fatal(err)
	}
	block := regexp.MustCompile(`(?s)pub const PANES: &\[&str\] = &\[(.*?)\];`).FindSubmatch(source)
	if block == nil {
		t.Fatal("opener.rs holds no PANES list")
	}
	found := regexp.MustCompile(`"([^"]+)"`).FindAllSubmatch(block[1], -1)
	shell := make([]string, 0, len(found))
	for _, m := range found {
		shell = append(shell, string(m[1]))
	}
	engine := make([]string, 0, len(catalog.Panes))
	for _, p := range catalog.Panes {
		engine = append(engine, p.URL)
	}
	slices.Sort(shell)
	slices.Sort(engine)
	if !slices.Equal(shell, engine) {
		t.Errorf("the shell opens %v; the catalog names %v", shell, engine)
	}
}
