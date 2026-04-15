package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest is the public agent.yaml type — composition + runtime
// config.
//
// The composition is a single flat dependency list. Under the old
// tool/plugin/skill trichotomy, each entry would land in a different
// key; under the unified package model they all share one `packages:`
// list. The parser distinguishes what's what by the manifest the ref
// resolves to (an `install:` block makes it a tool, a `plugin:` block
// makes it a plugin, a content-only body makes it a skill).
type Manifest struct {
	Name     string        `yaml:"name,omitempty"`
	Role     string        `yaml:"role,omitempty"`
	Team     string        `yaml:"team,omitempty"`
	Runtime  RuntimeConfig `yaml:"runtime,omitempty"`
	Plugins []string      `yaml:"plugins,omitempty"`
}

// RuntimeConfig allows per-agent runtime override.
type RuntimeConfig struct {
	Backend  string `yaml:"backend,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
	Auth     string `yaml:"auth,omitempty"`
}

// ManifestPath returns the full path to an agent's manifest file.
func ManifestPath(agentName string) string {
	return filepath.Join(AgentDir(agentName), "agent.yaml")
}

// LoadManifest reads the agent.yaml manifest for the given agent.
// Returns an empty Manifest (not an error) if the file doesn't exist.
func LoadManifest(agentName string) (*Manifest, error) {
	path := ManifestPath(agentName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &m, nil
}

// SaveManifest writes the manifest to the given agent's agent.yaml.
func SaveManifest(agentName string, m *Manifest) error {
	dir := AgentDir(agentName)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("agent %q not found", agentName)
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	path := ManifestPath(agentName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// AddPlugin appends a package ref to the agent's composition
// (idempotent). Replaces the old AddTool/AddPlugin/AddSkill trio.
func AddPlugin(agentName, ref string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	for _, p := range m.Plugins {
		if p == ref {
			return nil // already present
		}
	}
	m.Plugins = append(m.Plugins, ref)
	return SaveManifest(agentName, m)
}

// RemovePlugin removes a package ref from the agent's composition.
// No-op when the ref isn't present.
func RemovePlugin(agentName, ref string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	out := make([]string, 0, len(m.Plugins))
	for _, p := range m.Plugins {
		if p != ref {
			out = append(out, p)
		}
	}
	m.Plugins = out
	return SaveManifest(agentName, m)
}
