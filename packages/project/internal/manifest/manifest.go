// Package manifest is the internal parser and schema for spwn.yaml.
// The public surface is re-exported from packages/manifest.
//
// Schema model (v1):
//
//   - Agents are the primary runtime unit. Their on-disk presence at
//     spwn/agents/<name>/ is the source of truth for the project's
//     roster.
//   - Worlds are inline entries in spwn.yaml under `worlds:`. Each one
//     declares which agents it deploys, where the workspace mounts
//     come from, and optional tool overrides.
package manifest

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// CurrentVersion is the schema version Load emits for new manifests
// and the only version LoadPath accepts without upgrade.
const CurrentVersion = 1

// Manifest is the parsed content of spwn.yaml.
type Manifest struct {
	// Version is the schema version. Must be CurrentVersion.
	Version int `yaml:"version"`

	// Name is the project name. Used in world IDs, UI, and logs.
	Name string `yaml:"name"`

	// Worlds is the deployable world map keyed by world name. Each
	// entry declares which agents it spawns and what workspaces are
	// mounted into the resulting container.
	Worlds map[string]World `yaml:"worlds"`

	// Deps is the project-wide dependency pool. Every agent in
	// every world inherits these. Agent-level agent.yaml can add
	// more but cannot remove project-level dependency.
	Deps []string `yaml:"dependencies,omitempty"`
}

// World is one inline world entry in spwn.yaml.
type World struct {
	// Agents is the ordered list of agent names this world deploys.
	// Each name must match a directory under spwn/agents/.
	Agents []string `yaml:"agents"`

	// Workspaces is the list of host paths to mount inside the
	// container under /workspace. The first entry may be a bare host
	// path; subsequent entries must use explicit `host:/workspace/...`
	// form.
	Workspaces []string `yaml:"workspaces"`

	// Knowledge, when set, is a project-relative (or absolute) path to
	// a directory that will be bind-mounted into the container at
	// /world/knowledge/. When empty, no bind mount is performed and
	// the agent's system prompt omits every reference to the knowledge
	// base — the agent is never told a knowledge base exists.
	Knowledge string `yaml:"knowledge,omitempty"`
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
	if m.Worlds == nil {
		m.Worlds = map[string]World{}
	}
}

// AllAgentNames returns the deduplicated set of agent names referenced
// by any world entry in the manifest, in stable sorted order.
func (m *Manifest) AllAgentNames() []string {
	if m == nil {
		return nil
	}
	seen := map[string]struct{}{}
	for _, w := range m.Worlds {
		for _, a := range w.Agents {
			seen[a] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	// stable order without importing sort here would be ugly; use
	// sort to keep callers predictable.
	sortStrings(out)
	return out
}

// sortStrings is a tiny insertion sort kept local so this file doesn't
// pull in "sort" just for AllAgentNames.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
