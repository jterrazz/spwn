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
	// Return nil if the package ships no skills.
	Skills() fs.FS

	// Runtimes returns the runtime backends this package injects
	// config into (e.g. "@spwn/claude-code"). Return nil for the
	// common case of "not a plugin" — packages that aren't targeting
	// a runtime. Non-nil means the spawn-time merge pass should call
	// Config(runtime) for each matching runtime and shallow-merge the
	// result into the runtime's settings file.
	Runtimes() []string

	// Config returns the JSON snippet to merge into the named
	// runtime's config file inside the container, or nil if this
	// package has no config for that runtime. Called only when the
	// runtime is in Runtimes().
	Config(runtime string) []byte
}

// PluginConfig returns the config JSON a package would inject for the
// given runtime, enforcing the Runtimes() allowlist so individual
// packages don't have to repeat the check. Returns nil when the
// package doesn't target the runtime or has no config for it.
func PluginConfig(t Tool, runtime string) []byte {
	runtimes := t.Runtimes()
	if len(runtimes) == 0 {
		return nil
	}
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
	return t.Config(runtime)
}

// PluginRuntimes is retained as a convenience for callers that want
// the runtime list without caring whether the Tool implements the
// method themselves. Equivalent to t.Runtimes() today.
func PluginRuntimes(t Tool) []string { return t.Runtimes() }

// InstallSpec describes how to install a tool into a Docker image.
type InstallSpec struct {
	// AptPackages are apt-get packages to install. Deduplicated across tools.
	AptPackages []string

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
