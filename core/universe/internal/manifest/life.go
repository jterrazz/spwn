package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LifeManifest declares an agent's identity structure and requirements.
// Optional: if life.yaml doesn't exist in the agent directory, the agent dir is used as-is.
type LifeManifest struct {
	Name    string        `yaml:"name"`
	Role    string        `yaml:"role"`    // "governor" or "citizen" (default: "citizen")
	Runtime RuntimeConfig `yaml:"runtime"` // optional runtime override
	Soul    SoulManifest  `yaml:"soul"`
	Mind    MindManifest  `yaml:"mind"`
	Body    BodyManifest  `yaml:"body"`
}

// RuntimeConfig allows per-agent runtime override.
type RuntimeConfig struct {
	Backend  string `yaml:"backend"`  // "claude-code", "pi", "codex", etc.
	Provider string `yaml:"provider"` // "anthropic", "openai", "google", etc.
	Model    string `yaml:"model"`    // "claude-sonnet-4-6", etc.
	Auth     string `yaml:"auth"`     // "api-key" or "subscription"
}

// SoulManifest declares the agent's persona layers.
type SoulManifest struct {
	Personas []string `yaml:"personas"`
}

// MindManifest declares the agent's cognitive assets.
type MindManifest struct {
	Skills    []string `yaml:"skills"`
	Knowledge []string `yaml:"knowledge"`
	Playbooks []string `yaml:"playbooks"`
}

// BodyManifest declares the agent's physical requirements.
type BodyManifest struct {
	Requires []string `yaml:"requires"`
}

// DefaultRole returns the effective role, defaulting to "citizen" if empty.
func DefaultRole(role string) string {
	if role == "" {
		return "citizen"
	}
	return role
}

// DefaultTier is a deprecated wrapper for DefaultRole. Use DefaultRole instead.
func DefaultTier(tier string) string {
	return DefaultRole(tier)
}

// LoadLife reads life.yaml from an agent directory.
// Returns nil (no error) if life.yaml doesn't exist — it's optional.
func LoadLife(agentDir string) (*LifeManifest, error) {
	path := filepath.Join(agentDir, "life.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read life manifest: %w", err)
	}

	var life LifeManifest
	if err := yaml.Unmarshal(data, &life); err != nil {
		return nil, fmt.Errorf("parse life manifest: %w", err)
	}

	return &life, nil
}

// ValidateBody checks that all elements in body.requires are available in the universe.
// availableElements should be the expanded element list from the universe manifest.
func ValidateBody(life *LifeManifest, availableElements []string) error {
	if life == nil || len(life.Body.Requires) == 0 {
		return nil
	}

	available := make(map[string]bool)
	for _, e := range availableElements {
		available[e] = true
	}

	var missing []string
	for _, req := range life.Body.Requires {
		// Expand @packs to check individual binaries
		expanded := ExpandElements([]string{req})
		for _, e := range expanded {
			if !available[e] {
				missing = append(missing, e)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("agent requires element(s) not provided by this world: %s.\nHint: Add them to the world config's elements, or remove them from life.yaml body.requires",
			strings.Join(missing, ", "))
	}

	return nil
}
