package agent

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// ── Old "plugins:" key should NOT populate Deps ─────────────────────────────

func TestLoadManifest_OldPluginsKeyIgnored(t *testing.T) {
	dir := initAgent(t, "legacy")

	// Write an agent.yaml that uses the old "plugins:" key instead of "dependencies:".
	content := `name: legacy
role: worker
plugins:
  - "@spwn/unix"
  - "@spwn/git"
`
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest("legacy")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Deps) != 0 {
		t.Errorf("old plugins: key should NOT populate Deps, got %v", m.Deps)
	}
	if m.Name != "legacy" {
		t.Errorf("Name = %q, want %q", m.Name, "legacy")
	}
}

// ── AddDependency to agent with no agent.yaml yet ─────────────────────────────────

func TestAddDependency_CreatesManifestIfMissing(t *testing.T) {
	initAgent(t, "fresh")

	// No agent.yaml exists yet — AddDependency should create it.
	if err := AddDependency("fresh", "@spwn/python"); err != nil {
		t.Fatalf("AddDependency: %v", err)
	}

	m, err := LoadManifest("fresh")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Deps) != 1 || m.Deps[0] != "@spwn/python" {
		t.Errorf("Deps = %v, want [@spwn/python]", m.Deps)
	}

	// Verify file actually exists on disk.
	path := ManifestPath("fresh")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("agent.yaml should exist after AddDependency: %v", err)
	}
}

// ── RemoveDependency for ref not in list → no error ────────────────────────────────

func TestRemoveDependency_AbsentRefNoError(t *testing.T) {
	initAgent(t, "sparse")

	// Start with one dep.
	if err := AddDependency("sparse", "@spwn/unix"); err != nil {
		t.Fatal(err)
	}

	// Remove a ref that was never added — should succeed silently.
	if err := RemoveDependency("sparse", "@spwn/never-existed"); err != nil {
		t.Errorf("RemoveDependency of absent ref should not error, got: %v", err)
	}

	// Original dep should still be there.
	m, _ := LoadManifest("sparse")
	if len(m.Deps) != 1 || m.Deps[0] != "@spwn/unix" {
		t.Errorf("Deps after no-op remove = %v, want [@spwn/unix]", m.Deps)
	}
}

func TestRemoveDependency_EmptyManifestNoError(t *testing.T) {
	initAgent(t, "empty")

	// No deps at all — remove should be a no-op.
	if err := RemoveDependency("empty", "@spwn/anything"); err != nil {
		t.Errorf("RemoveDependency on empty manifest should not error, got: %v", err)
	}
}

// ── AddDependency twice with same ref → idempotent ────────────────────────────────

func TestAddDependency_Idempotent(t *testing.T) {
	initAgent(t, "idem")

	ref := "@spwn/git"
	for i := 0; i < 3; i++ {
		if err := AddDependency("idem", ref); err != nil {
			t.Fatalf("AddDependency iteration %d: %v", i, err)
		}
	}

	m, _ := LoadManifest("idem")
	if len(m.Deps) != 1 {
		t.Errorf("expected exactly 1 dep after triple add, got %d: %v", len(m.Deps), m.Deps)
	}
	if m.Deps[0] != ref {
		t.Errorf("Deps[0] = %q, want %q", m.Deps[0], ref)
	}
}

// ── LoadManifest with empty deps list ────────────────────────────────────────

func TestLoadManifest_EmptyDepsList(t *testing.T) {
	dir := initAgent(t, "nodeps")

	content := `name: nodeps
role: worker
dependencies: []
`
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest("nodeps")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m.Deps == nil {
		// YAML `deps: []` should unmarshal as non-nil empty slice or
		// nil — either is acceptable but the length must be zero.
	}
	if len(m.Deps) != 0 {
		t.Errorf("expected 0 deps, got %d: %v", len(m.Deps), m.Deps)
	}
	if m.Name != "nodeps" {
		t.Errorf("Name = %q, want %q", m.Name, "nodeps")
	}
}

func TestLoadManifest_OmittedDeps(t *testing.T) {
	dir := initAgent(t, "omit")

	// deps key entirely absent — should still parse fine.
	content := `name: omit
role: worker
`
	if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := LoadManifest("omit")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Deps) != 0 {
		t.Errorf("expected 0 deps when key omitted, got %d", len(m.Deps))
	}
}

// ── SaveManifest writes deps: not plugins: ───────────────────────────────────

func TestSaveManifest_UsesDependenciesKey(t *testing.T) {
	initAgent(t, "schema")

	m := &Manifest{
		Name: "schema",
		Deps: []string{"@spwn/unix"},
	}
	if err := SaveManifest("schema", m); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(ManifestPath("schema"))
	if err != nil {
		t.Fatal(err)
	}

	// The serialized YAML must use "dependencies:", never "plugins:".
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["plugins"]; ok {
		t.Error("SaveManifest wrote a 'plugins:' key — expected 'dependencies:' only")
	}
	if _, ok := parsed["dependencies"]; !ok {
		t.Error("SaveManifest did not write a 'dependencies:' key")
	}
}
