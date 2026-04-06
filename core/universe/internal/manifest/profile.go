package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProfileManifest declares an agent's operational configuration.
// Optional: if profile.yaml doesn't exist in the agent directory, the agent dir is used as-is.
// Backward compatible: falls back to life.yaml if profile.yaml is not found.
// Identity fields (purpose, traits, persona, bonds) now live in core/*.md files, not profile.yaml.
type ProfileManifest struct {
	Name    string        `yaml:"name,omitempty"`
	Role    string        `yaml:"role,omitempty"`    // "chief", "manager", or "worker" (default: "worker")
	Team    string        `yaml:"team,omitempty"`    // team slug (references ~/.spwn/teams/{slug}.yaml)
	Runtime RuntimeConfig `yaml:"runtime,omitempty"` // optional runtime override
	Skills  []string      `yaml:"skills,omitempty"`  // formerly under "mind"
}

// IdentityManifest declares the agent's identity/persona layers.
type IdentityManifest struct {
	Purpose  string   `yaml:"purpose"`
	Traits   []string `yaml:"traits"`
	Persona  string   `yaml:"persona"`
	Bonds    []string `yaml:"bonds"`
	Personas []string `yaml:"personas"` // backward compat
}

// MemoryManifest declares the agent's memory assets.
type MemoryManifest struct {
	Knowledge []string `yaml:"knowledge"`
	Playbooks []string `yaml:"playbooks"`
}

// LoadProfile reads profile.yaml from an agent directory.
// Falls back to life.yaml for backward compatibility.
// Returns nil (no error) if neither file exists — it's optional.
func LoadProfile(agentDir string) (*ProfileManifest, error) {
	// Try profile.yaml first
	path := filepath.Join(agentDir, "profile.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fallback to life.yaml
			return loadProfileFromLife(agentDir)
		}
		return nil, fmt.Errorf("read profile manifest: %w", err)
	}

	var profile ProfileManifest
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("parse profile manifest: %w", err)
	}

	return &profile, nil
}

// loadProfileFromLife reads a legacy life.yaml and converts it to ProfileManifest.
func loadProfileFromLife(agentDir string) (*ProfileManifest, error) {
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

	// Convert LifeManifest to ProfileManifest (slimmed — identity/memory now in .md files)
	profile := &ProfileManifest{
		Name:    life.Name,
		Role:    life.Role,
		Runtime: life.Runtime,
		Skills:  life.Mind.Skills,
	}

	return profile, nil
}

// ValidateRequires checks that all tools in requires are available in the universe.
// availableTools should be the expanded tool list from the universe manifest.
// Deprecated: requires was removed from ProfileManifest. Kept for backward compatibility
// with code that may still call it — always returns nil now.
func ValidateRequires(profile *ProfileManifest, availableTools []string) error {
	return nil
}
