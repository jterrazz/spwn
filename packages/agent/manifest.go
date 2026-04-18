package agent

import (
	"fmt"
	"os"

	intmanifest "spwn.sh/packages/agent/internal/manifest"
)

// Manifest is the parsed agent.yaml — composition + runtime config.
// Re-exported as a type alias so callers stay on the single `agent`
// import and never need to know the yaml schema lives under internal/.
type Manifest = intmanifest.Manifest

// RuntimeConfig is the per-agent runtime override.
type RuntimeConfig = intmanifest.RuntimeConfig

// ManifestPath returns the full path to an agent's manifest file.
func ManifestPath(agentName string) string {
	return intmanifest.Path(AgentDir(agentName))
}

// LoadManifest reads the agent.yaml manifest for the given agent.
// Returns an empty Manifest (not an error) if the file doesn't exist.
func LoadManifest(agentName string) (*Manifest, error) {
	m, _, err := intmanifest.Load(AgentDir(agentName))
	return m, err
}

// LoadManifestPath reads agent.yaml from an explicit directory.
// Returns (nil, nil) when agent.yaml doesn't exist. Used by callers
// that have a resolved agent dir (e.g. the spawn pipeline) rather
// than a name.
func LoadManifestPath(agentDir string) (*Manifest, error) {
	m, ok, err := intmanifest.Load(agentDir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return m, nil
}

// SaveManifest writes the manifest to the given agent's agent.yaml.
func SaveManifest(agentName string, m *Manifest) error {
	dir := AgentDir(agentName)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("agent %q not found", agentName)
	}
	return intmanifest.Save(dir, m)
}

// AddDependency appends a dependency ref to the agent's composition
// (idempotent). Replaces the old AddTool/AddPack/AddSkill trio.
func AddDependency(agentName, ref string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	for _, p := range m.Deps {
		if p == ref {
			return nil // already present
		}
	}
	m.Deps = append(m.Deps, ref)
	return SaveManifest(agentName, m)
}

// RemoveDependency removes a dependency ref from the agent's
// composition. No-op when the ref isn't present.
func RemoveDependency(agentName, ref string) error {
	m, err := LoadManifest(agentName)
	if err != nil {
		return err
	}
	out := make([]string, 0, len(m.Deps))
	for _, p := range m.Deps {
		if p != ref {
			out = append(out, p)
		}
	}
	m.Deps = out
	return SaveManifest(agentName, m)
}

// DefaultRole returns the effective role, defaulting to "worker" if empty.
func DefaultRole(role string) string {
	if role == "" {
		return "worker"
	}
	return role
}
