package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProfile_ValidProfileYAML(t *testing.T) {
	dir := t.TempDir()
	content := `name: neo
role: chief
runtime:
  backend: claude-code
  provider: anthropic
  model: claude-sonnet-4-6
skills:
  - golang
  - python
`
	if err := os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write profile.yaml: %v", err)
	}

	profile, err := LoadProfile(dir)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile == nil {
		t.Fatal("expected non-nil profile")
	}
	if profile.Name != "neo" {
		t.Errorf("Name = %q, want \"neo\"", profile.Name)
	}
	if profile.Role != "chief" {
		t.Errorf("Role = %q, want \"chief\"", profile.Role)
	}
	if profile.Runtime.Provider != "anthropic" {
		t.Errorf("Runtime.Provider = %q, want \"anthropic\"", profile.Runtime.Provider)
	}
	if profile.Runtime.Model != "claude-sonnet-4-6" {
		t.Errorf("Runtime.Model = %q, want \"claude-sonnet-4-6\"", profile.Runtime.Model)
	}
	if len(profile.Skills) != 2 {
		t.Errorf("Skills count = %d, want 2", len(profile.Skills))
	}
}

func TestLoadProfile_FallsBackToLifeYAML(t *testing.T) {
	dir := t.TempDir()
	content := `name: legacy-agent
role: worker
soul:
  personas:
    - helper
mind:
  skills:
    - debugging
  knowledge:
    - go-patterns
  playbooks:
    - deploy
body:
  requires:
    - git
`
	if err := os.WriteFile(filepath.Join(dir, "life.yaml"), []byte(content), 0644); err != nil {
		t.Fatalf("write life.yaml: %v", err)
	}

	profile, err := LoadProfile(dir)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile == nil {
		t.Fatal("expected non-nil profile from life.yaml fallback")
	}
	if profile.Name != "legacy-agent" {
		t.Errorf("Name = %q, want \"legacy-agent\"", profile.Name)
	}
	if profile.Role != "worker" {
		t.Errorf("Role = %q, want \"worker\"", profile.Role)
	}
	if len(profile.Skills) != 1 || profile.Skills[0] != "debugging" {
		t.Errorf("Skills = %v, want [debugging]", profile.Skills)
	}
}

func TestLoadProfile_NeitherFileReturnsNil(t *testing.T) {
	dir := t.TempDir()

	profile, err := LoadProfile(dir)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile when no files exist, got %+v", profile)
	}
}

func TestLoadProfile_ProfileYAMLTakesPrecedence(t *testing.T) {
	dir := t.TempDir()

	// Both files exist — profile.yaml should win
	profileContent := `name: profile-agent
role: chief
`
	lifeContent := `name: life-agent
role: worker
`
	os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte(profileContent), 0644)
	os.WriteFile(filepath.Join(dir, "life.yaml"), []byte(lifeContent), 0644)

	profile, err := LoadProfile(dir)
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if profile.Name != "profile-agent" {
		t.Errorf("Name = %q, want \"profile-agent\" (profile.yaml should take precedence)", profile.Name)
	}
}

func TestLoadProfile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "profile.yaml"), []byte("{{invalid yaml"), 0644)

	_, err := LoadProfile(dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parse profile manifest") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

func TestLoadProfile_InvalidLifeYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "life.yaml"), []byte("{{invalid yaml"), 0644)

	_, err := LoadProfile(dir)
	if err == nil {
		t.Error("expected error for invalid life.yaml")
	}
	if !strings.Contains(err.Error(), "parse life manifest") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

func TestValidateRequires_NilProfile(t *testing.T) {
	err := ValidateRequires(nil, []string{"git", "node"})
	if err != nil {
		t.Errorf("expected nil error for nil profile, got: %v", err)
	}
}

func TestValidateRequires_AlwaysReturnsNil(t *testing.T) {
	// ValidateRequires is deprecated (requires removed from ProfileManifest)
	profile := &ProfileManifest{}
	err := ValidateRequires(profile, []string{"git"})
	if err != nil {
		t.Errorf("expected nil error (deprecated), got: %v", err)
	}
}

func TestProfileManifest_Defaults(t *testing.T) {
	// Zero value ProfileManifest should have reasonable defaults
	profile := ProfileManifest{}

	// Role defaults via DefaultRole
	if DefaultRole(profile.Role) != "worker" {
		t.Errorf("default role should be worker, got %q", DefaultRole(profile.Role))
	}

	// Empty runtime
	if profile.Runtime.Backend != "" {
		t.Errorf("default backend should be empty, got %q", profile.Runtime.Backend)
	}
	if profile.Runtime.Provider != "" {
		t.Errorf("default provider should be empty, got %q", profile.Runtime.Provider)
	}
	if profile.Runtime.Model != "" {
		t.Errorf("default model should be empty, got %q", profile.Runtime.Model)
	}
}
