package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProfileManifest declares an agent's identity structure and requirements.
// Optional: if profile.yaml doesn't exist in the agent directory, the agent dir is used as-is.
// Backward compatible: falls back to life.yaml if profile.yaml is not found.
type ProfileManifest struct {
	Name       string        `yaml:"name"`
	Tier       string        `yaml:"tier"`       // "governor" or "citizen" (default: "citizen")
	Team       string        `yaml:"team"`       // team slug (references ~/.spwn/teams/{slug}.yaml)
	Runtime    RuntimeConfig `yaml:"runtime"`     // optional runtime override
	Identity   IdentityManifest `yaml:"identity"` // formerly "soul"
	Skills     []string      `yaml:"skills"`      // formerly under "mind"
	Requires   []string      `yaml:"requires"`    // formerly under "body"
	Delegation string        `yaml:"delegation"`  // formerly "body.orchestration"
	Memory     MemoryManifest `yaml:"memory"`
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

	// Convert LifeManifest to ProfileManifest
	profile := &ProfileManifest{
		Name:    life.Name,
		Tier:    life.Tier,
		Runtime: life.Runtime,
		Identity: IdentityManifest{
			Personas: life.Soul.Personas,
		},
		Skills:   life.Mind.Skills,
		Requires: life.Body.Requires,
		Memory: MemoryManifest{
			Knowledge: life.Mind.Knowledge,
			Playbooks: life.Mind.Playbooks,
		},
	}

	return profile, nil
}

// ValidateRequires checks that all elements in requires are available in the universe.
// availableElements should be the expanded element list from the universe manifest.
func ValidateRequires(profile *ProfileManifest, availableElements []string) error {
	if profile == nil || len(profile.Requires) == 0 {
		return nil
	}

	available := make(map[string]bool)
	for _, e := range availableElements {
		available[e] = true
	}

	var missing []string
	for _, req := range profile.Requires {
		// Expand @packs to check individual binaries
		expanded := ExpandElements([]string{req})
		for _, e := range expanded {
			if !available[e] {
				missing = append(missing, e)
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("agent requires element(s) not provided by this world: %s.\nHint: Add them to the world config's elements, or remove them from profile.yaml requires",
			strings.Join(missing, ", "))
	}

	return nil
}
