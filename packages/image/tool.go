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
