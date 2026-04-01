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
tier: governor
runtime:
  backend: claude-code
  provider: anthropic
  model: claude-sonnet-4-6
identity:
  purpose: "coding assistant"
  traits:
    - precise
    - helpful
skills:
  - golang
  - python
requires:
  - git
  - node
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
	if profile.Tier != "governor" {
		t.Errorf("Tier = %q, want \"governor\"", profile.Tier)
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
	if len(profile.Requires) != 2 {
		t.Errorf("Requires count = %d, want 2", len(profile.Requires))
	}
	if profile.Identity.Purpose != "coding assistant" {
		t.Errorf("Identity.Purpose = %q, want \"coding assistant\"", profile.Identity.Purpose)
	}
}

func TestLoadProfile_FallsBackToLifeYAML(t *testing.T) {
	dir := t.TempDir()
	content := `name: legacy-agent
tier: citizen
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
	if profile.Tier != "citizen" {
		t.Errorf("Tier = %q, want \"citizen\"", profile.Tier)
	}
	if len(profile.Skills) != 1 || profile.Skills[0] != "debugging" {
		t.Errorf("Skills = %v, want [debugging]", profile.Skills)
	}
	if len(profile.Requires) != 1 || profile.Requires[0] != "git" {
		t.Errorf("Requires = %v, want [git]", profile.Requires)
	}
	if len(profile.Memory.Knowledge) != 1 || profile.Memory.Knowledge[0] != "go-patterns" {
		t.Errorf("Memory.Knowledge = %v, want [go-patterns]", profile.Memory.Knowledge)
	}
	if len(profile.Identity.Personas) != 1 || profile.Identity.Personas[0] != "helper" {
		t.Errorf("Identity.Personas = %v, want [helper]", profile.Identity.Personas)
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
tier: governor
`
	lifeContent := `name: life-agent
tier: citizen
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

func TestValidateRequires_EmptyRequires(t *testing.T) {
	profile := &ProfileManifest{Requires: []string{}}
	err := ValidateRequires(profile, []string{"git"})
	if err != nil {
		t.Errorf("expected nil error for empty requires, got: %v", err)
	}
}

func TestValidateRequires_AllPresent(t *testing.T) {
	profile := &ProfileManifest{
		Requires: []string{"git", "node"},
	}
	available := ExpandElements([]string{"@git", "@node"})
	err := ValidateRequires(profile, available)
	if err != nil {
		t.Errorf("expected nil error when all requires satisfied, got: %v", err)
	}
}

func TestValidateRequires_MissingElement(t *testing.T) {
	profile := &ProfileManifest{
		Requires: []string{"git", "docker"},
	}
	available := ExpandElements([]string{"@git"})
	err := ValidateRequires(profile, available)
	if err == nil {
		t.Error("expected error when element is missing")
	}
	if !strings.Contains(err.Error(), "docker") {
		t.Errorf("error should mention missing element 'docker', got: %v", err)
	}
}

func TestValidateRequires_PackExpansion(t *testing.T) {
	profile := &ProfileManifest{
		Requires: []string{"@node"},
	}
	// World only provides @unix, not @node
	available := ExpandElements([]string{"@unix"})
	err := ValidateRequires(profile, available)
	if err == nil {
		t.Error("expected error: @node binaries not in @unix")
	}
	// Should mention specific missing binary (node, npm, or npx)
	if !strings.Contains(err.Error(), "node") {
		t.Errorf("error should mention missing 'node' binary, got: %v", err)
	}
}

func TestValidateRequires_HintInError(t *testing.T) {
	profile := &ProfileManifest{
		Requires: []string{"docker"},
	}
	err := ValidateRequires(profile, []string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Hint") {
		t.Errorf("error should contain a hint, got: %v", err)
	}
}

func TestProfileManifest_Defaults(t *testing.T) {
	// Zero value ProfileManifest should have reasonable defaults
	profile := ProfileManifest{}

	// Tier defaults via DefaultTier
	if DefaultTier(profile.Tier) != "citizen" {
		t.Errorf("default tier should be citizen, got %q", DefaultTier(profile.Tier))
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
