package agent

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// initAgent scaffolds a minimal agent directory so the Manifest
// helpers have somewhere to read/write.
func initAgent(t *testing.T, name string) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)
	dir := filepath.Join(tmp, "agents", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return dir
}

// ── Load / Save roundtrip ────────────────────────────────────────────────────

func TestLoadManifest_MissingFileReturnsEmpty(t *testing.T) {
	initAgent(t, "neo")

	m, err := LoadManifest("neo")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if m == nil {
		t.Fatal("LoadManifest returned nil on missing file; expected empty Manifest")
	}
	if m.Name != "" || len(m.Packages) != 0 {
		t.Errorf("expected empty manifest, got %+v", m)
	}
}

func TestSaveManifest_WritesYAML(t *testing.T) {
	initAgent(t, "neo")

	m := &Manifest{
		Name:     "neo",
		Role:     "chief",
		Packages: []string{"@spwn/unix", "@spwn/python", "kung-fu"},
	}
	if err := SaveManifest("neo", m); err != nil {
		t.Fatalf("SaveManifest: %v", err)
	}

	path := ManifestPath("neo")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if len(data) == 0 {
		t.Error("manifest file is empty")
	}

	var got Manifest
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "neo" {
		t.Errorf("Name = %q, want \"neo\"", got.Name)
	}
	if len(got.Packages) != 3 {
		t.Errorf("Packages count = %d, want 3", len(got.Packages))
	}
}

func TestSaveManifest_AgentNotFound(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("SPWN_HOME", tmp)

	err := SaveManifest("ghost", &Manifest{Name: "ghost"})
	if err == nil {
		t.Error("expected error when agent dir doesn't exist, got nil")
	}
}

func TestLoadManifest_InvalidYAML(t *testing.T) {
	initAgent(t, "neo")
	if err := os.WriteFile(ManifestPath("neo"), []byte("{{ not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadManifest("neo")
	if err == nil {
		t.Error("expected parse error, got nil")
	}
}

func TestLoadManifest_RoundtripPreservesFields(t *testing.T) {
	initAgent(t, "curie")

	original := &Manifest{
		Name: "curie",
		Role: "worker",
		Team: "research",
		Runtime: RuntimeConfig{
			Backend:  "claude-code",
			Provider: "anthropic",
			Model:    "claude-sonnet-4-6",
		},
		Packages: []string{"@spwn/python", "@spwn/unix", "paper-reading"},
	}
	if err := SaveManifest("curie", original); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadManifest("curie")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != original.Name || loaded.Role != original.Role || loaded.Team != original.Team {
		t.Errorf("identity fields drifted: got %+v", loaded)
	}
	if loaded.Runtime.Backend != "claude-code" {
		t.Errorf("Runtime.Backend = %q", loaded.Runtime.Backend)
	}
	if len(loaded.Packages) != 3 || loaded.Packages[0] != "@spwn/python" {
		t.Errorf("Packages drifted: %v", loaded.Packages)
	}
}

// ── AddPackage / RemovePackage ──────────────────────────────────────────────

func TestAddPackage_AppendsAndIsIdempotent(t *testing.T) {
	initAgent(t, "neo")

	if err := AddPackage("neo", "@spwn/python"); err != nil {
		t.Fatal(err)
	}
	if err := AddPackage("neo", "@spwn/unix"); err != nil {
		t.Fatal(err)
	}
	if err := AddPackage("neo", "@spwn/python"); err != nil {
		t.Fatal(err)
	}

	m, _ := LoadManifest("neo")
	if len(m.Packages) != 2 {
		t.Errorf("expected 2 packages after double-add, got %d: %v", len(m.Packages), m.Packages)
	}
}

func TestRemovePackage_RemovesPresentAndIsNoOpForAbsent(t *testing.T) {
	initAgent(t, "neo")

	AddPackage("neo", "@spwn/python")
	AddPackage("neo", "@spwn/git")

	if err := RemovePackage("neo", "@spwn/git"); err != nil {
		t.Fatal(err)
	}
	m, _ := LoadManifest("neo")
	if len(m.Packages) != 1 || m.Packages[0] != "@spwn/python" {
		t.Errorf("after remove: %v", m.Packages)
	}

	if err := RemovePackage("neo", "@spwn/never-added"); err != nil {
		t.Errorf("remove absent: %v", err)
	}
	m, _ = LoadManifest("neo")
	if len(m.Packages) != 1 {
		t.Errorf("after no-op remove: %v", m.Packages)
	}
}

func TestComposition_FullRoundtrip(t *testing.T) {
	initAgent(t, "neo")

	AddPackage("neo", "@spwn/unix")
	AddPackage("neo", "@spwn/python")
	AddPackage("neo", "refactoring")
	AddPackage("neo", "paper-reading")

	RemovePackage("neo", "@spwn/unix")
	RemovePackage("neo", "paper-reading")

	m, err := LoadManifest("neo")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Packages) != 2 {
		t.Errorf("Packages count = %d, want 2", len(m.Packages))
	}
}

func TestManifestPath_UsesAgentYAML(t *testing.T) {
	initAgent(t, "neo")
	path := ManifestPath("neo")
	if filepath.Base(path) != "agent.yaml" {
		t.Errorf("ManifestPath returned %q, want agent.yaml", path)
	}
}
