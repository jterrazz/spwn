package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Top-level deps: field is parsed correctly ───────────────────────────────

func TestLoadPath_TopLevelDeps(t *testing.T) {
	dir := t.TempDir()
	content := `version: 2
name: deptest
dependencies:
  - "@spwn/unix"
  - "@spwn/git"
  - custom-tool
worlds:
  home:
    agents: [neo]
    workspaces: [.]
`
	path := filepath.Join(dir, "spwn.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	if len(m.Deps) != 3 {
		t.Fatalf("expected 3 deps, got %d: %v", len(m.Deps), m.Deps)
	}
	want := []string{"@spwn/unix", "@spwn/git", "custom-tool"}
	for i, w := range want {
		if m.Deps[i] != w {
			t.Errorf("Deps[%d] = %q, want %q", i, m.Deps[i], w)
		}
	}
}

func TestLoadPath_NoDeps(t *testing.T) {
	dir := t.TempDir()
	content := `version: 2
name: nodep
worlds:
  home:
    agents: [neo]
    workspaces: [.]
`
	path := filepath.Join(dir, "spwn.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	if len(m.Deps) != 0 {
		t.Errorf("expected 0 deps when key omitted, got %d: %v", len(m.Deps), m.Deps)
	}
}

func TestLoadPath_EmptyDepsList(t *testing.T) {
	dir := t.TempDir()
	content := `version: 2
name: emptydep
dependencies: []
worlds:
  home:
    agents: [neo]
    workspaces: [.]
`
	path := filepath.Join(dir, "spwn.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	if len(m.Deps) != 0 {
		t.Errorf("expected 0 deps for empty list, got %d", len(m.Deps))
	}
}

// ── World entries no longer have a deps field ────────────────────────────────

func TestLoadPath_WorldDepsKeyIgnored(t *testing.T) {
	dir := t.TempDir()
	// If someone adds a deps: key inside a world entry, it should be
	// silently ignored (not parsed into the World struct).
	content := `version: 2
name: worlddep
worlds:
  home:
    agents: [neo]
    workspaces: [.]
    dependencies:
      - "@spwn/unix"
`
	path := filepath.Join(dir, "spwn.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	w, ok := m.Worlds["home"]
	if !ok {
		t.Fatal("missing world 'home'")
	}
	// The World struct should not have a Deps field. We verify by
	// checking that the struct only has Agents and Workspaces populated.
	if len(w.Agents) != 1 || w.Agents[0] != "neo" {
		t.Errorf("Agents = %v, want [neo]", w.Agents)
	}
	if len(w.Workspaces) != 1 || w.Workspaces[0] != "." {
		t.Errorf("Workspaces = %v, want [.]", w.Workspaces)
	}

	// Project-level deps should NOT contain the world-level value.
	if len(m.Deps) != 0 {
		t.Errorf("project-level Deps should be empty, got %v", m.Deps)
	}
}

// ── Defaults are applied ─────────────────────────────────────────────────────

func TestLoadPath_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	content := `name: defaultstest
dependencies:
  - foo
`
	path := filepath.Join(dir, "spwn.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	if m.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d (default)", m.Version, CurrentVersion)
	}
	if m.Worlds == nil {
		t.Error("Worlds should be non-nil after defaults")
	}
	if len(m.Deps) != 1 || m.Deps[0] != "foo" {
		t.Errorf("Deps = %v, want [foo]", m.Deps)
	}
}
