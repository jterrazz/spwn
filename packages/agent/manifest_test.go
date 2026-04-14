package agent

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// initAgent scaffolds a minimal agent directory so the Manifest helpers have
// somewhere to read/write. It does NOT use the full mind.Init flow to keep
// tests focused on the manifest layer.
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
	if m.Name != "" || len(m.Tools) != 0 || len(m.Skills) != 0 || m.Profile != "" {
		t.Errorf("expected empty manifest, got %+v", m)
	}
}

func TestSaveManifest_WritesYAML(t *testing.T) {
	initAgent(t, "neo")

	m := &Manifest{
		Name:    "neo",
		Role:    "chief",
		Profile: "the-one",
		Tools:   []string{"@spwn/unix", "@spwn/python"},
		Skills:  []string{"kung-fu"},
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

	// Parse back and verify round-trip.
	var got Manifest
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Name != "neo" {
		t.Errorf("Name = %q, want \"neo\"", got.Name)
	}
	if got.Profile != "the-one" {
		t.Errorf("Profile = %q, want \"the-one\"", got.Profile)
	}
	if len(got.Tools) != 2 {
		t.Errorf("Tools count = %d, want 2", len(got.Tools))
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
	// Write garbage to the manifest file.
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
		Name:    "curie",
		Role:    "worker",
		Team:    "research",
		Profile: "researcher",
		Runtime: RuntimeConfig{
			Backend:  "claude-code",
			Provider: "anthropic",
			Model:    "claude-sonnet-4-6",
		},
		Tools:  []string{"@spwn/python", "@spwn/unix"},
		Skills: []string{"paper-reading", "hypothesis-testing"},
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
	if loaded.Profile != original.Profile {
		t.Errorf("Profile drifted: %q → %q", original.Profile, loaded.Profile)
	}
	if loaded.Runtime.Backend != "claude-code" {
		t.Errorf("Runtime.Backend = %q", loaded.Runtime.Backend)
	}
	if len(loaded.Tools) != 2 || loaded.Tools[0] != "@spwn/python" {
		t.Errorf("Tools drifted: %v", loaded.Tools)
	}
	if len(loaded.Skills) != 2 || loaded.Skills[0] != "paper-reading" {
		t.Errorf("Skills drifted: %v", loaded.Skills)
	}
}

// ── AddTool / RemoveTool ────────────────────────────────────────────────────

func TestAddTool_AppendsAndIsIdempotent(t *testing.T) {
	initAgent(t, "neo")

	if err := AddTool("neo", "@spwn/python"); err != nil {
		t.Fatal(err)
	}
	if err := AddTool("neo", "@spwn/unix"); err != nil {
		t.Fatal(err)
	}
	// Adding the same tool twice should be a no-op.
	if err := AddTool("neo", "@spwn/python"); err != nil {
		t.Fatal(err)
	}

	m, _ := LoadManifest("neo")
	if len(m.Tools) != 2 {
		t.Errorf("expected 2 tools after double-add, got %d: %v", len(m.Tools), m.Tools)
	}
}

func TestRemoveTool_RemovesPresentToolAndIsNoOpForAbsent(t *testing.T) {
	initAgent(t, "neo")

	AddTool("neo", "@spwn/python")
	AddTool("neo", "@spwn/git")

	// Remove a present tool.
	if err := RemoveTool("neo", "@spwn/git"); err != nil {
		t.Fatal(err)
	}
	m, _ := LoadManifest("neo")
	if len(m.Tools) != 1 || m.Tools[0] != "@spwn/python" {
		t.Errorf("after remove: %v", m.Tools)
	}

	// Removing an absent tool should be a no-op, not an error.
	if err := RemoveTool("neo", "@spwn/never-added"); err != nil {
		t.Errorf("remove absent: %v", err)
	}
	m, _ = LoadManifest("neo")
	if len(m.Tools) != 1 {
		t.Errorf("after no-op remove: %v", m.Tools)
	}
}

// ── AddSkill / RemoveSkill ───────────────────────────────────────────────────

func TestAddSkill_AppendsAndIsIdempotent(t *testing.T) {
	initAgent(t, "neo")

	AddSkill("neo", "refactoring")
	AddSkill("neo", "paper-reading")
	AddSkill("neo", "refactoring") // duplicate

	m, _ := LoadManifest("neo")
	if len(m.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d: %v", len(m.Skills), m.Skills)
	}
}

func TestRemoveSkill_RemovesPresentAndIsNoOpForAbsent(t *testing.T) {
	initAgent(t, "neo")

	AddSkill("neo", "refactoring")
	AddSkill("neo", "paper-reading")

	if err := RemoveSkill("neo", "paper-reading"); err != nil {
		t.Fatal(err)
	}
	m, _ := LoadManifest("neo")
	if len(m.Skills) != 1 || m.Skills[0] != "refactoring" {
		t.Errorf("after remove: %v", m.Skills)
	}

	// No-op on absent.
	if err := RemoveSkill("neo", "no-such-skill"); err != nil {
		t.Errorf("remove absent: %v", err)
	}
}

// ── SetProfile / ClearProfile ────────────────────────────────────────────────

func TestSetProfile_OverwritesPrevious(t *testing.T) {
	initAgent(t, "neo")

	if err := SetProfile("neo", "researcher"); err != nil {
		t.Fatal(err)
	}
	m, _ := LoadManifest("neo")
	if m.Profile != "researcher" {
		t.Errorf("Profile = %q, want \"researcher\"", m.Profile)
	}

	// Setting again overwrites.
	if err := SetProfile("neo", "the-one"); err != nil {
		t.Fatal(err)
	}
	m, _ = LoadManifest("neo")
	if m.Profile != "the-one" {
		t.Errorf("Profile = %q, want \"the-one\"", m.Profile)
	}
}

func TestClearProfile_RemovesAttachment(t *testing.T) {
	initAgent(t, "neo")

	SetProfile("neo", "researcher")
	if err := ClearProfile("neo"); err != nil {
		t.Fatal(err)
	}
	m, _ := LoadManifest("neo")
	if m.Profile != "" {
		t.Errorf("Profile = %q, want empty after clear", m.Profile)
	}
}

func TestClearProfile_EmptyManifestIsNoOp(t *testing.T) {
	initAgent(t, "neo")

	if err := ClearProfile("neo"); err != nil {
		t.Errorf("clear on empty: %v", err)
	}
}

// ── Composition across multiple calls ────────────────────────────────────────

func TestComposition_FullRoundtrip(t *testing.T) {
	initAgent(t, "neo")

	// Build up a composition incrementally (simulating CLI add commands).
	AddTool("neo", "@spwn/unix")
	AddTool("neo", "@spwn/python")
	AddSkill("neo", "refactoring")
	AddSkill("neo", "paper-reading")
	SetProfile("neo", "researcher")

	// Remove one tool and one skill.
	RemoveTool("neo", "@spwn/unix")
	RemoveSkill("neo", "paper-reading")

	// Load final state and verify.
	m, err := LoadManifest("neo")
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if len(m.Tools) != 1 || m.Tools[0] != "@spwn/python" {
		t.Errorf("Tools = %v, want [@spwn/python]", m.Tools)
	}
	if len(m.Skills) != 1 || m.Skills[0] != "refactoring" {
		t.Errorf("Skills = %v, want [refactoring]", m.Skills)
	}
	if m.Profile != "researcher" {
		t.Errorf("Profile = %q, want \"researcher\"", m.Profile)
	}
}

func TestManifestPath_UsesAgentYAML(t *testing.T) {
	initAgent(t, "neo")
	path := ManifestPath("neo")
	if filepath.Base(path) != "agent.yaml" {
		t.Errorf("ManifestPath returned %q, want agent.yaml", path)
	}
}
