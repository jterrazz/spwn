package migrations

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRestructureAgentDirs_FullMigration(t *testing.T) {
	dir := t.TempDir()

	// Create an agent with old directory structure
	agentDir := filepath.Join(dir, "agents", "neo")
	os.MkdirAll(filepath.Join(agentDir, "core"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "skills"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "memory", "knowledge"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "memory", "playbooks"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "memory", "journal"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "sessions"), 0755)

	// Write files in old locations
	os.WriteFile(filepath.Join(agentDir, "core", "persona.md"), []byte("# Persona"), 0644)
	os.WriteFile(filepath.Join(agentDir, "core", "purpose.md"), []byte("# Purpose"), 0644)
	os.WriteFile(filepath.Join(agentDir, "skills", "coding.md"), []byte("# Coding"), 0644)
	os.WriteFile(filepath.Join(agentDir, "memory", "knowledge", "facts.md"), []byte("# Facts"), 0644)
	os.WriteFile(filepath.Join(agentDir, "memory", "playbooks", "deploy.md"), []byte("# Deploy"), 0644)
	os.WriteFile(filepath.Join(agentDir, "memory", "journal", "2025-01-01.md"), []byte("# Journal"), 0644)
	os.WriteFile(filepath.Join(agentDir, "sessions", "w-test.json"), []byte(`{"id":"s1"}`), 0644)

	// Write profile.yaml with deprecated fields
	profileContent := "name: neo\nrole: chief\nidentity:\n  purpose: test\nrequires:\n  - git\ndelegation: auto\nmemory:\n  knowledge:\n    - facts\nskills:\n  - golang\n"
	os.WriteFile(filepath.Join(agentDir, "profile.yaml"), []byte(profileContent), 0644)

	if err := RestructureAgentDirs.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	// Verify core/ was renamed to identity/
	if _, err := os.Stat(filepath.Join(agentDir, "core")); !os.IsNotExist(err) {
		t.Error("core/ should have been renamed")
	}
	data, err := os.ReadFile(filepath.Join(agentDir, "identity", "persona.md"))
	if err != nil {
		t.Fatalf("identity/persona.md missing: %v", err)
	}
	if string(data) != "# Persona" {
		t.Errorf("unexpected content: %s", data)
	}

	// Verify memory/knowledge/ moved to knowledge/
	data, err = os.ReadFile(filepath.Join(agentDir, "knowledge", "facts.md"))
	if err != nil {
		t.Fatalf("knowledge/facts.md missing: %v", err)
	}
	if string(data) != "# Facts" {
		t.Errorf("unexpected content: %s", data)
	}

	// Verify memory/playbooks/ moved to playbooks/
	data, err = os.ReadFile(filepath.Join(agentDir, "playbooks", "deploy.md"))
	if err != nil {
		t.Fatalf("playbooks/deploy.md missing: %v", err)
	}
	if string(data) != "# Deploy" {
		t.Errorf("unexpected content: %s", data)
	}

	// Verify memory/journal/ merged into journal/
	data, err = os.ReadFile(filepath.Join(agentDir, "journal", "2025-01-01.md"))
	if err != nil {
		t.Fatalf("journal/2025-01-01.md missing: %v", err)
	}
	if string(data) != "# Journal" {
		t.Errorf("unexpected content: %s", data)
	}

	// Verify sessions/ merged into journal/
	data, err = os.ReadFile(filepath.Join(agentDir, "journal", "w-test.json"))
	if err != nil {
		t.Fatalf("journal/w-test.json missing: %v", err)
	}
	if string(data) != `{"id":"s1"}` {
		t.Errorf("unexpected content: %s", data)
	}

	// Verify memory/ and sessions/ removed
	if _, err := os.Stat(filepath.Join(agentDir, "memory")); !os.IsNotExist(err) {
		t.Error("memory/ should have been removed")
	}
	if _, err := os.Stat(filepath.Join(agentDir, "sessions")); !os.IsNotExist(err) {
		t.Error("sessions/ should have been removed")
	}

	// Verify profile.yaml was slimmed
	data, err = os.ReadFile(filepath.Join(agentDir, "profile.yaml"))
	if err != nil {
		t.Fatalf("profile.yaml missing: %v", err)
	}
	content := string(data)
	if contains(content, "identity:") {
		t.Error("profile.yaml should not contain identity block")
	}
	if contains(content, "requires:") {
		t.Error("profile.yaml should not contain requires block")
	}
	if contains(content, "delegation:") {
		t.Error("profile.yaml should not contain delegation field")
	}
	if contains(content, "memory:") {
		t.Error("profile.yaml should not contain memory block")
	}
	if !contains(content, "role:") {
		t.Error("profile.yaml should still contain role")
	}
	if !contains(content, "skills:") {
		t.Error("profile.yaml should still contain skills")
	}
}

func TestRestructureAgentDirs_NoAgentsDir(t *testing.T) {
	dir := t.TempDir()
	if err := RestructureAgentDirs.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
}

func TestRestructureAgentDirs_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents", "neo")
	os.MkdirAll(filepath.Join(agentDir, "identity"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "skills"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "knowledge"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "playbooks"), 0755)
	os.MkdirAll(filepath.Join(agentDir, "journal"), 0755)
	os.WriteFile(filepath.Join(agentDir, "identity", "persona.md"), []byte("# Persona"), 0644)

	// Running migration on already-migrated agent should be idempotent
	if err := RestructureAgentDirs.Apply(context.Background(), dir); err != nil {
		t.Fatal(err)
	}

	// identity/ should still exist with its file
	data, err := os.ReadFile(filepath.Join(agentDir, "identity", "persona.md"))
	if err != nil {
		t.Fatalf("identity/persona.md missing: %v", err)
	}
	if string(data) != "# Persona" {
		t.Errorf("unexpected content: %s", data)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
