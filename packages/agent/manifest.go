package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest is the public agent.yaml type — composition + runtime config.
//
// Lives at ~/.spwn/agents/{name}/agent.yaml. Optional: if absent, the agent
// directory is used as-is with built-in defaults.
type Manifest struct {
	Name    string        `yaml:"name,omitempty"`
	Role    string        `yaml:"role,omitempty"`
	Team    string        `yaml:"team,omitempty"`
	Runtime RuntimeConfig `yaml:"runtime,omitempty"`
	Profile string        `yaml:"profile,omitempty"` // personality template reference
	Tools   []string      `yaml:"tools,omitempty"`   // tool packs
	Skills  []string      `yaml:"skills,omitempty"`  // skill files
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

// AddTool appends a tool pack to the agent's composition (idempotent).
func AddTool(agentName, tool string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	for _, t := range m.Tools {
		if t == tool {
			return nil // already present
		}
	}
	m.Tools = append(m.Tools, tool)
	return SaveManifest(agentName, m)
}

// RemoveTool removes a tool pack from the agent's composition.
// No-op if the tool isn't attached.
func RemoveTool(agentName, tool string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	out := make([]string, 0, len(m.Tools))
	for _, t := range m.Tools {
		if t != tool {
			out = append(out, t)
		}
	}
	m.Tools = out
	return SaveManifest(agentName, m)
}

// AddSkill appends a skill to the agent's composition (idempotent).
func AddSkill(agentName, skill string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	for _, s := range m.Skills {
		if s == skill {
			return nil
		}
	}
	m.Skills = append(m.Skills, skill)
	return SaveManifest(agentName, m)
}

// RemoveSkill removes a skill from the agent's composition.
func RemoveSkill(agentName, skill string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	out := make([]string, 0, len(m.Skills))
	for _, s := range m.Skills {
		if s != skill {
			out = append(out, s)
		}
	}
	m.Skills = out
	return SaveManifest(agentName, m)
}

// SetProfile attaches a profile template to the agent.
func SetProfile(agentName, profile string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	m.Profile = profile
	return SaveManifest(agentName, m)
}

// ClearProfile removes the profile attachment.
func ClearProfile(agentName string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	m.Profile = ""
	return SaveManifest(agentName, m)
}
