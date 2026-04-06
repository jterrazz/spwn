package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/core/foundation"
)

// setupTestAgent creates a minimal agent Mind in a temp SPWN_HOME and returns the home dir.
func setupTestAgent(t *testing.T, name string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	agentDir := filepath.Join(home, foundation.AgentsSubDir, name)
	layers := []string{"identity", "skills", "memory/knowledge", "memory/playbooks", "memory/journal", "sessions"}
	for _, layer := range layers {
		if err := os.MkdirAll(filepath.Join(agentDir, layer), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// Create default persona
	os.WriteFile(filepath.Join(agentDir, "identity", "persona.md"), []byte("# Default\nYou are a test agent.\n"), 0644)
	return home
}

func TestProfile_AgentNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)
	os.MkdirAll(filepath.Join(home, "agents"), 0755)

	err := Cmd.RunE(Cmd, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %s", err)
	}
	if !strings.Contains(err.Error(), "spwn agent new") {
		t.Errorf("expected hint about creating agent, got: %s", err)
	}
}

func TestProfile_ShowCharacterSheet(t *testing.T) {
	setupTestAgent(t, "neo")

	// Should not return an error for an existing agent
	err := Cmd.RunE(Cmd, []string{"neo"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Purpose_FileNotFound(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "purpose"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// Should not crash — shows "Not set yet."
}

func TestProfile_Purpose_ShowsContent(t *testing.T) {
	home := setupTestAgent(t, "neo")

	purposePath := filepath.Join(home, "agents", "neo", "identity", "purpose.md")
	os.WriteFile(purposePath, []byte("Build the future.\n"), 0644)

	err := Cmd.RunE(Cmd, []string{"neo", "purpose"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Skills_Empty(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "skills"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Skills_ListsFiles(t *testing.T) {
	home := setupTestAgent(t, "neo")

	skillsDir := filepath.Join(home, "agents", "neo", "skills")
	os.WriteFile(filepath.Join(skillsDir, "deploy.md"), []byte("Deploy to production\n"), 0644)
	os.WriteFile(filepath.Join(skillsDir, "debug.md"), []byte("Debug issues\n"), 0644)

	err := Cmd.RunE(Cmd, []string{"neo", "skills"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Journal_Empty(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "journal"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Journal_ShowsEntries(t *testing.T) {
	home := setupTestAgent(t, "neo")

	journalDir := filepath.Join(home, "agents", "neo", "memory", "journal")
	// Write a journal entry in the expected format
	entry := "---\nworld: w-test-123\nexit_code: 0\nduration: 5m\ncreated_at: 2025-01-15T10:00:00Z\n---\nSession completed successfully.\n"
	os.WriteFile(filepath.Join(journalDir, "2025-01-15T10-00-00.md"), []byte(entry), 0644)

	err := Cmd.RunE(Cmd, []string{"neo", "journal"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Role_NoProfileYaml(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "role"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Role_ShowsCurrent(t *testing.T) {
	home := setupTestAgent(t, "neo")

	profilePath := filepath.Join(home, "agents", "neo", "profile.yaml")
	os.WriteFile(profilePath, []byte("role: chief\n"), 0644)

	err := Cmd.RunE(Cmd, []string{"neo", "role"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Edit_CreatesDefault(t *testing.T) {
	home := setupTestAgent(t, "neo")

	profilePath := filepath.Join(home, "agents", "neo", "profile.yaml")

	// Verify profile.yaml does not exist
	if _, err := os.Stat(profilePath); err == nil {
		t.Fatal("profile.yaml should not exist before edit")
	}

	// We can't actually open an editor in tests, but we can test that
	// editProfile creates the default if missing. We'll set EDITOR to "true" (no-op).
	t.Setenv("EDITOR", "true")
	err := editProfile(Cmd, "neo")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Verify profile.yaml was created
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("profile.yaml should have been created: %s", err)
	}
	if !strings.Contains(string(data), "role") {
		t.Error("profile.yaml should contain role field")
	}
}

func TestProfile_UnknownAspect(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown aspect")
	}
	if !strings.Contains(err.Error(), "unknown profile aspect") {
		t.Errorf("expected 'unknown profile aspect' error, got: %s", err)
	}
}

func TestProfile_Sessions_Empty(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "sessions"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Knowledge_Empty(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "knowledge"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Playbooks_Empty(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "playbooks"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Bonds_FileNotFound(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "bonds"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestProfile_Engine_ShowsDefault(t *testing.T) {
	setupTestAgent(t, "neo")

	err := Cmd.RunE(Cmd, []string{"neo", "engine"})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func TestLoadProfileYAML_Defaults(t *testing.T) {
	setupTestAgent(t, "neo")

	p, err := loadProfileYAML("neo")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if p.Role != "worker" {
		t.Errorf("expected default role 'worker', got %q", p.Role)
	}
	if p.Runtime.Engine != "claude-code" {
		t.Errorf("expected default engine 'claude-code', got %q", p.Runtime.Engine)
	}
}

func TestLoadProfileYAML_CustomValues(t *testing.T) {
	home := setupTestAgent(t, "neo")

	profilePath := filepath.Join(home, "agents", "neo", "profile.yaml")
	os.WriteFile(profilePath, []byte("role: chief\nruntime:\n  engine: gpt\n  provider: openai\n  model: gpt-4\n"), 0644)

	p, err := loadProfileYAML("neo")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if p.Role != "chief" {
		t.Errorf("expected role 'chief', got %q", p.Role)
	}
	if p.Runtime.Engine != "gpt" {
		t.Errorf("expected engine 'gpt', got %q", p.Runtime.Engine)
	}
}
