// Package manifest is the hidden parser for agent.yaml. The agent
// package re-exports Manifest + RuntimeConfig as type aliases and
// wraps Load/Save with domain operations (AddDependency,
// RemoveDependency) so callers stay on a single import and never see
// yaml tags or file-I/O plumbing.
package manifest

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Manifest is the parsed agent.yaml — composition + runtime config.
//
// The composition is a single flat dependency list. Every entry is a
// `scheme:target` ref resolved via one of the known schemes
// (spwn:/github:/skill:/tool:/hook:); the parser distinguishes what's
// what by the manifest the ref resolves to (an `install:` block makes
// it a tool, a content-only body makes it a skill).
type Manifest struct {
	Name string `yaml:"name,omitempty"`

	// Description is a mandatory one-line pitch of what this agent is
	// for — the equivalent of the `description:` field in the skill
	// frontmatter convention. `spwn check` flags an empty description
	// as LevelError so every agent in a project has a human-readable
	// purpose line that the inspector, web UI, and external tooling
	// can render without opening AGENTS.md.
	Description string        `yaml:"description,omitempty"`
	Role        string        `yaml:"role,omitempty"`
	Team        string        `yaml:"team,omitempty"`
	Runtime     RuntimeConfig `yaml:"runtime,omitempty"`

	// Deps is the flat list of dependency refs (`spwn:x`,
	// `skill:refine`, …). Populated by UnmarshalYAML from either
	// scalar entries (legacy / no policy) or mapping entries with
	// allow/deny — see DepPolicies for the policy side.
	Deps []string `yaml:"-"`

	// DepPolicies indexes per-dep allow/deny lists, keyed by the
	// dependency ref string (matches an entry in Deps). Populated
	// when the agent.yaml entry is written as a mapping:
	//
	//   dependencies:
	//     - spwn:unix
	//     - name: spwn:x
	//       deny: [post-tweet, reply-tweet]
	//     - name: spwn:notion
	//       allow: [search, fetch_page]
	//
	// Tools without a policy entry are unfiltered.
	DepPolicies map[string]DepPolicy `yaml:"-"`
}

// DepPolicy filters which methods of a dependency the agent may
// call. allow is a positive list (everything else denied); deny is
// a negative list (everything else allowed). Setting both is valid
// — deny takes precedence. Empty policy = unfiltered.
type DepPolicy struct {
	Allow []string `yaml:"allow,omitempty"`
	Deny  []string `yaml:"deny,omitempty"`
}

// rawManifest mirrors Manifest but exposes Deps as a yaml.Node so
// we can split string vs mapping entries before populating Deps +
// DepPolicies. Kept private; the public Manifest.UnmarshalYAML
// converts between the two.
type rawManifest struct {
	Name        string        `yaml:"name,omitempty"`
	Description string        `yaml:"description,omitempty"`
	Role        string        `yaml:"role,omitempty"`
	Team        string        `yaml:"team,omitempty"`
	Runtime     RuntimeConfig `yaml:"runtime,omitempty"`
	Deps        yaml.Node     `yaml:"dependencies,omitempty"`
}

// UnmarshalYAML accepts each `dependencies:` entry as either a
// scalar string ("spwn:x") or a mapping with name + optional
// allow/deny.
func (m *Manifest) UnmarshalYAML(node *yaml.Node) error {
	var raw rawManifest
	if err := node.Decode(&raw); err != nil {
		return err
	}
	m.Name = raw.Name
	m.Description = raw.Description
	m.Role = raw.Role
	m.Team = raw.Team
	m.Runtime = raw.Runtime
	m.Deps = nil
	m.DepPolicies = nil
	if raw.Deps.Kind == 0 {
		return nil
	}
	if raw.Deps.Kind != yaml.SequenceNode {
		return fmt.Errorf("dependencies: must be a sequence")
	}
	for _, item := range raw.Deps.Content {
		switch item.Kind {
		case yaml.ScalarNode:
			m.Deps = append(m.Deps, item.Value)
		case yaml.MappingNode:
			var entry struct {
				Name  string   `yaml:"name"`
				Allow []string `yaml:"allow"`
				Deny  []string `yaml:"deny"`
			}
			if err := item.Decode(&entry); err != nil {
				return fmt.Errorf("dependencies entry: %w", err)
			}
			if entry.Name == "" {
				return fmt.Errorf("dependencies entry missing `name`")
			}
			m.Deps = append(m.Deps, entry.Name)
			if len(entry.Allow) > 0 || len(entry.Deny) > 0 {
				if m.DepPolicies == nil {
					m.DepPolicies = make(map[string]DepPolicy)
				}
				m.DepPolicies[entry.Name] = DepPolicy{Allow: entry.Allow, Deny: entry.Deny}
			}
		default:
			return fmt.Errorf("dependencies entry must be a string or mapping")
		}
	}
	return nil
}

// MarshalYAML round-trips the parsed form back to the wire format —
// mapping entries for policied deps, scalars for everything else.
// Keeps `spwn agent inspect` and Save() output stable.
func (m Manifest) MarshalYAML() (any, error) {
	out := struct {
		Name        string        `yaml:"name,omitempty"`
		Description string        `yaml:"description,omitempty"`
		Role        string        `yaml:"role,omitempty"`
		Team        string        `yaml:"team,omitempty"`
		Runtime     RuntimeConfig `yaml:"runtime,omitempty"`
		Deps        []any         `yaml:"dependencies,omitempty"`
	}{
		Name:        m.Name,
		Description: m.Description,
		Role:        m.Role,
		Team:        m.Team,
		Runtime:     m.Runtime,
	}
	for _, ref := range m.Deps {
		if pol, ok := m.DepPolicies[ref]; ok && (len(pol.Allow) > 0 || len(pol.Deny) > 0) {
			entry := map[string]any{"name": ref}
			if len(pol.Allow) > 0 {
				entry["allow"] = pol.Allow
			}
			if len(pol.Deny) > 0 {
				entry["deny"] = pol.Deny
			}
			out.Deps = append(out.Deps, entry)
		} else {
			out.Deps = append(out.Deps, ref)
		}
	}
	return out, nil
}

// RuntimeConfig is the per-agent runtime override.
type RuntimeConfig struct {
	Backend  string `yaml:"backend,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
	Auth     string `yaml:"auth,omitempty"`
}
