// Package manifest is the internal parser and schema for spwn.yaml.
// The public surface is re-exported from packages/manifest.
package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CurrentVersion is the schema version Load emits for new manifests
// and the only version LoadPath accepts without migration.
const CurrentVersion = 1

// Manifest is the parsed content of spwn.yaml.
type Manifest struct {
	// Version is the schema version. Must be CurrentVersion.
	Version int `yaml:"version"`

	// Name is the project name. Used in world IDs, UI, and logs.
	Name string `yaml:"name"`

	// Workspace is the host directory mounted into every world this
	// project spawns. Relative paths resolve from the manifest root.
	Workspace string `yaml:"workspace,omitempty"`

	// World is the name of the world config to spawn. Resolved to
	// ./spwn/worlds/<World>.yaml at load time.
	World string `yaml:"world"`

	// Agents lists the agents this project spawns. Each name
	// resolves to ./spwn/agents/<name>/ at load time.
	Agents []string `yaml:"agents"`
}

// LoadPath reads and parses spwn.yaml from an explicit file path.
// Applies defaults but does NOT run validation rules.
func LoadPath(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	ApplyDefaults(&m)
	return &m, nil
}

// ApplyDefaults fills in optional fields that were left blank.
func ApplyDefaults(m *Manifest) {
	if m.Version == 0 {
		m.Version = CurrentVersion
	}
	if m.Workspace == "" {
		m.Workspace = "."
	}
}
