// Package pkgyaml is the shared parser for package.yaml — the
// declarative manifest format that describes a spwn package's
// image-build recipe. Both the catalog (catalog/packages/<name>/
// package.yaml) and project-local packages (spwn/packages/<name>/
// package.yaml in a user project) use the same schema, so a package
// can graduate from "authored in a project" to "shipped in the
// catalog" by moving its directory.
//
// A package is whatever its fields say it is: install steps + verify
// make it a tool; a plugin: section makes it inject runtime config; a
// SKILL.md sibling or content-only body makes it a skill. There is
// no explicit type field — composability determines identity.
//
// The parser produces image.Tool instances (via the adapter in
// adapter.go), so everything downstream — registry resolution,
// dockerfile generation, skill collection — is oblivious to whether a
// given package came from Go or YAML.
package pkgyaml

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Schema is the on-disk shape of package.yaml. Every field is
// optional so a minimal package ("install one thing, verify it's
// there") stays short.
type Schema struct {
	// Name is the tool identifier (e.g. "@spwn/git"). Optional: when
	// empty, the loader derives it from the directory name (plain
	// local tools) or from the parent directory (catalog tools
	// auto-prefix with "@spwn/").
	Name string `yaml:"name"`

	// Kind classifies the tool: "runtime", "sdk", "tool", or
	// "platform". Defaults to "tool".
	Kind string `yaml:"kind"`

	// Version is a semver string or "latest". Required for catalog
	// tools; defaults to "0.0.0-local" for project-local tools.
	Version string `yaml:"version"`

	// Description is a human-readable one-liner. Optional.
	Description string `yaml:"description"`

	// Dependencies is a flat list of other tool refs this one needs.
	// The registry resolves them transitively and topologically
	// sorts the final install order.
	Dependencies []string `yaml:"dependencies"`

	// Install is the build-time recipe for baking this tool into the
	// image. All sub-fields are optional — a tool that only ships
	// skills can leave Install empty entirely.
	Install InstallSection `yaml:"install"`

	// Files is a map of image-target-path → source path relative to
	// this tool's directory. Contents are read at parse time and
	// baked into the image via the Dockerfile's COPY layer.
	Files map[string]string `yaml:"files"`

	// Verify is the list of post-build sanity commands. Each must
	// exit 0. Typically "command -v <binary>" or "<binary> --version".
	Verify []string `yaml:"verify"`

	// Plugin, when present, promotes this tool to a Plugin — a tool
	// that targets one or more runtimes and injects configuration
	// into their settings files at spawn time.
	Plugin *PluginSection `yaml:"plugin,omitempty"`

	// RuntimeProvider names a host-side Go implementation that
	// handles credential sync, default config file materialisation,
	// and prelaunch shell setup at spawn time. Only runtimes
	// ("@spwn/claude-code", "@spwn/codex") need this today; a tool
	// that leaves it blank gets no spawn-time hooks. The string is
	// looked up against a Go-side registry — unknown names fail at
	// load time so typos are caught early.
	RuntimeProvider string `yaml:"runtime-provider,omitempty"`
}

// InstallSection mirrors packages/image.InstallSpec but uses wire-level
// types so the parser is self-contained.
type InstallSection struct {
	// Packages are apt-get packages. Deduplicated across every tool
	// in the image, so ordering here is irrelevant.
	Packages []string `yaml:"packages"`

	// Commands run as root, before the USER switch. Each item
	// becomes one RUN line in the Dockerfile, so order matters.
	Commands []string `yaml:"commands"`

	// UserCommands run after the USER switch. Each item becomes one
	// RUN line. The strings {{.Home}} and {{.User}} are templated
	// with the actual home directory and username by the Dockerfile
	// generator — use these instead of hard-coding /home/spwn so
	// the same tool works under any user.
	UserCommands []string `yaml:"user-commands"`

	// Env are ENV directives added to the Dockerfile.
	Env map[string]string `yaml:"env"`
}

// PluginSection is the optional `plugin:` block. A tool that declares
// this becomes a Plugin: the Runtimes list scopes which runtime
// backends the plugin targets, and Configs is a map from runtime name
// to the YAML-native snippet that gets merged into the runtime's
// settings file at spawn time.
//
// Example (mempalace targeting Claude Code):
//
//	plugin:
//	  runtimes:
//	    - "@spwn/claude-code"
//	  configs:
//	    "@spwn/claude-code":
//	      mcpServers:
//	        mempalace:
//	          command: python3
//	          args: ["-m", "mempalace.mcp_server"]
//
// The merger converts the YAML value to JSON at spawn time and
// shallow-merges into ~/.claude/settings.json.
type PluginSection struct {
	Runtimes []string                   `yaml:"runtimes"`
	Configs  map[string]yaml.Node       `yaml:"configs"`
}

// ConfigJSON marshals a plugin's config for the given runtime to JSON
// bytes, so spawn-time callers that merge into JSON settings files
// don't have to care the source was YAML. Returns nil when the plugin
// has no config for that runtime.
func (p *PluginSection) ConfigJSON(runtime string) ([]byte, error) {
	if p == nil {
		return nil, nil
	}
	node, ok := p.Configs[runtime]
	if !ok {
		return nil, nil
	}
	var raw any
	if err := node.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode plugin config for %q: %w", runtime, err)
	}
	out, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal plugin config for %q: %w", runtime, err)
	}
	return out, nil
}
