package image

import "io/fs"

// Kind classifies what role a tool plays in an image.
type Kind string

const (
	KindRuntime  Kind = "runtime"  // Agent thinking engine (@spwn/claude-code, @spwn/aider)
	KindTool     Kind = "tool"     // Extra capability (@spwn/qmd, @jq)
	KindSDK      Kind = "sdk"      // Language/runtime SDK (@spwn/node, @spwn/python)
	KindPlatform Kind = "platform" // Spwn infrastructure (@spwn/cli, @spwn/architect)
)

// Tool is the contract that any installable capability must implement.
// Built-in tools in catalog/ implement this; third-party tools can too.
type Tool interface {
	// Name returns the tool identifier (e.g., "@spwn/qmd", "@spwn/node").
	Name() string

	// Kind returns the tool's category.
	Kind() Kind

	// Version returns the tool's version (semver or "latest").
	Version() string

	// Dependencies returns names of other tools this one requires.
	// The engine resolves these transitively, deduplicates, and topologically sorts.
	Dependencies() []string

	// Install returns the recipe to bake this tool into a Docker image.
	Install() InstallSpec

	// Verify returns commands to run post-build to confirm installation.
	// Each must exit 0. Typically "command -v <binary>" or "<binary> --version".
	Verify() []string

	// Skills returns embedded skill files (Vercel SKILL.md convention).
	// Return nil if the tool ships no skills.
	Skills() fs.FS
}

// Plugin is an optional extension of the Tool interface. A Tool that
// also implements Plugin is a "plugin": it targets one or more runtimes
// and can inject runtime-specific configuration (merged into e.g.
// ~/.claude/settings.json inside the container) at spawn time.
//
// Plugin is intentionally a separate interface (not added to Tool) so
// existing tools keep compiling unchanged. Call sites that care about
// the plugin-specific methods should type-assert:
//
//	if p, ok := t.(Plugin); ok { ... }
//
// A Tool can also embed PluginBase to satisfy the Plugin contract with
// safe defaults (runtime-agnostic, no config).
type Plugin interface {
	Tool

	// Runtimes returns the runtime backends this plugin plugs into
	// (e.g. "@spwn/claude-code"). An empty slice means runtime-agnostic.
	Runtimes() []string

	// Config returns the JSON snippet to merge into the named runtime's
	// config file inside the container, or nil if this plugin has no
	// config for that runtime. Runtimes that don't match the plugin's
	// declared Runtimes() must return nil.
	Config(runtime string) []byte
}

// PluginBase is a zero-value helper that satisfies the Plugin-specific
// methods with runtime-agnostic no-op defaults. Tools that want to
// become plugins without writing the methods by hand can embed this.
//
//	type tool struct{ ib.PluginBase }
//
// The defaults are Runtimes() → nil and Config(_) → nil, which is the
// correct behavior for a runtime-agnostic tool that ships no injected
// config.
type PluginBase struct{}

// Runtimes returns nil — runtime-agnostic by default.
func (PluginBase) Runtimes() []string { return nil }

// Config returns nil — no runtime config by default.
func (PluginBase) Config(runtime string) []byte { return nil }

// PluginRuntimes returns the runtimes a Tool targets, or nil if the
// Tool is not a Plugin. Convenience for callers that would otherwise
// type-assert inline.
func PluginRuntimes(t Tool) []string {
	if p, ok := t.(Plugin); ok {
		return p.Runtimes()
	}
	return nil
}

// PluginConfig returns the config JSON a Tool would inject for the
// given runtime, or nil if the Tool is not a Plugin or has no config
// for that runtime.
func PluginConfig(t Tool, runtime string) []byte {
	p, ok := t.(Plugin)
	if !ok {
		return nil
	}
	// A plugin that declared Runtimes() must only return config for
	// runtimes in that list. Enforce the rule at the boundary so
	// individual plugins don't have to.
	runtimes := p.Runtimes()
	if len(runtimes) > 0 {
		match := false
		for _, r := range runtimes {
			if r == runtime {
				match = true
				break
			}
		}
		if !match {
			return nil
		}
	}
	return p.Config(runtime)
}

// InstallSpec describes how to install a tool into a Docker image.
type InstallSpec struct {
	// Packages are apt-get packages to install. Deduplicated across tools.
	Packages []string

	// Commands are RUN lines executed as root, before the USER switch.
	Commands []string

	// UserCommands are RUN lines executed after the USER switch.
	// Use these for writing config files to $HOME or other user-specific setup.
	// The generator templates {{.Home}} and {{.User}} with the actual values.
	UserCommands []string

	// Env are ENV key=value directives added to the Dockerfile.
	Env map[string]string

	// Files are paths→content pairs COPYd into the image.
	Files map[string][]byte
}
