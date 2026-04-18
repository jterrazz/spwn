// Package manifest is the hidden parser for agent.yaml. The agent
// package re-exports Manifest + RuntimeConfig as type aliases and
// wraps Load/Save with domain operations (AddDependency,
// RemoveDependency) so callers stay on a single import and never see
// yaml tags or file-I/O plumbing.
package manifest

// Manifest is the parsed agent.yaml — composition + runtime config.
//
// The composition is a single flat dependency list. Under the old
// tool/runtime-config/skill trichotomy, each entry would land in a
// different key; under the unified dependency model they all share
// one `dependencies:` list. The parser distinguishes what's what by
// the manifest the ref resolves to (an `install:` block makes it a
// tool, a `runtime-config:` block makes it a runtime-config injector,
// a content-only body makes it a skill).
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
	Deps        []string      `yaml:"dependencies,omitempty"`
}

// RuntimeConfig is the per-agent runtime override.
type RuntimeConfig struct {
	Backend  string `yaml:"backend,omitempty"`
	Provider string `yaml:"provider,omitempty"`
	Model    string `yaml:"model,omitempty"`
	Auth     string `yaml:"auth,omitempty"`
}
