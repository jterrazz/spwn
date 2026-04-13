package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrgPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "org.yaml")

	content := `name: test-org
version: 1
defaults:
  runtime:
    backend: claude-code
    provider: anthropic
  backend: docker
  memory: filesystem
  store: json
governance:
  max-worlds: 10
  audit: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test org.yaml: %v", err)
	}

	org, err := LoadOrgPath(path)
	if err != nil {
		t.Fatalf("LoadOrgPath() error: %v", err)
	}

	if org.Name != "test-org" {
		t.Errorf("Name = %q, want %q", org.Name, "test-org")
	}
	if org.Version != 1 {
		t.Errorf("Version = %d, want 1", org.Version)
	}
	if org.Defaults.Runtime.Backend != "claude-code" {
		t.Errorf("Runtime.Backend = %q, want %q", org.Defaults.Runtime.Backend, "claude-code")
	}
	if org.Defaults.Runtime.Provider != "anthropic" {
		t.Errorf("Runtime.Provider = %q, want %q", org.Defaults.Runtime.Provider, "anthropic")
	}
	if org.Defaults.Backend != "docker" {
		t.Errorf("Defaults.Backend = %q, want %q", org.Defaults.Backend, "docker")
	}
	if org.Governance.MaxWorlds != 10 {
		t.Errorf("Governance.MaxWorlds = %d, want 10", org.Governance.MaxWorlds)
	}
	if !org.Governance.Audit {
		t.Error("Governance.Audit should be true")
	}
}

func TestLoadOrgPath_NotFound(t *testing.T) {
	_, err := LoadOrgPath("/nonexistent/org.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadOrgPath_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "org.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := LoadOrgPath(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestCreateOrg(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", dir)
	t.Setenv("UNIVERSE_HOME", "")

	if err := CreateOrg("my-org"); err != nil {
		t.Fatalf("CreateOrg() error: %v", err)
	}

	// Verify the file was created and can be loaded
	org, err := LoadOrgPath(filepath.Join(dir, "org.yaml"))
	if err != nil {
		t.Fatalf("LoadOrgPath() after CreateOrg error: %v", err)
	}

	if org.Name != "my-org" {
		t.Errorf("Name = %q, want %q", org.Name, "my-org")
	}
	if org.Version != 1 {
		t.Errorf("Version = %d, want 1", org.Version)
	}
	if org.Defaults.Runtime.Backend != "claude-code" {
		t.Errorf("Runtime.Backend = %q, want %q", org.Defaults.Runtime.Backend, "claude-code")
	}
	if org.Defaults.Backend != "docker" {
		t.Errorf("Defaults.Backend = %q, want %q", org.Defaults.Backend, "docker")
	}
}

func TestLoadOrgPath_WithClaw(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "org.yaml")

	content := `name: claw-org
version: 1
claw:
  sync:
    repo: git@github.com:test/repo.git
    branch: main
    auto-push: true
    auto-pull: true
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test org.yaml: %v", err)
	}

	org, err := LoadOrgPath(path)
	if err != nil {
		t.Fatalf("LoadOrgPath() error: %v", err)
	}

	if org.Claw.Sync.Repo != "git@github.com:test/repo.git" {
		t.Errorf("Sync.Repo = %q, want %q", org.Claw.Sync.Repo, "git@github.com:test/repo.git")
	}
	if !org.Claw.Sync.AutoPush {
		t.Error("Sync.AutoPush should be true")
	}
}
