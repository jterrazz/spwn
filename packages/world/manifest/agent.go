package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AgentManifest declares an agent's composition — the flat package
// dependency list plus runtime configuration.
//
// Stored in ~/.spwn/agents/{name}/agent.yaml. Optional: if agent.yaml
// doesn't exist, the agent directory is used as-is with built-in
// defaults.
type AgentManifest struct {
	Name     string        `yaml:"name,omitempty"`
	Role     string        `yaml:"role,omitempty"`    // "chief", "manager", or "worker" (default: "worker")
	Team     string        `yaml:"team,omitempty"`    // team slug (references ~/.spwn/teams/{slug}.yaml)
	Runtime  RuntimeConfig `yaml:"runtime,omitempty"` // optional runtime override
	Deps []string `yaml:"deps,omitempty"` // pack refs attached to the agent
}

// RuntimeConfig allows per-agent runtime override.
type RuntimeConfig struct {
	Backend  string `yaml:"backend,omitempty"`  // "claude-code", "pi", "codex", etc.
	Provider string `yaml:"provider,omitempty"` // "anthropic", "openai", "google", etc.
	Model    string `yaml:"model,omitempty"`    // "claude-sonnet-4-6", etc.
	Auth     string `yaml:"auth,omitempty"`     // "api-key" or "subscription"
}

// DefaultRole returns the effective role, defaulting to "worker" if empty.
func DefaultRole(role string) string {
	if role == "" {
		return "worker"
	}
	return role
}

// AgentManifestPath returns the path to an agent's manifest file.
func AgentManifestPath(agentDir string) string {
	return filepath.Join(agentDir, "agent.yaml")
}

// LoadAgent reads agent.yaml from an agent directory.
// Returns (nil, nil) if agent.yaml doesn't exist - the manifest is optional.
func LoadAgent(agentDir string) (*AgentManifest, error) {
	path := AgentManifestPath(agentDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read agent manifest: %w", err)
	}

	var m AgentManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse agent manifest: %w", err)
	}

	return &m, nil
}

// SaveAgent writes an agent.yaml manifest to the given agent directory.
// Creates the directory if it doesn't exist.
func SaveAgent(agentDir string, m *AgentManifest) error {
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("create agent dir: %w", err)
	}
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshal agent manifest: %w", err)
	}
	path := AgentManifestPath(agentDir)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write agent manifest: %w", err)
	}
	return nil
}
