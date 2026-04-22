package tool

import "io/fs"

// Tool is the contract that any installable capability must implement.
// Built-in tools in catalog/ implement this; third-party tools can too.
type Tool interface {
	// Name returns the tool identifier (e.g., "spwn:qmd", "spwn:node").
	Name() string

	// Version returns the tool's version (semver or "latest").
	Version() string

	// Dependencies returns names of other tools this one requires.
	// The engine resolves these transitively, deduplicates, and topologically sorts.
	Dependencies() []string

	// Install returns the recipe to compile this tool into a Docker compile.
	Install() InstallSpec

	// Verify returns commands to run post-build to confirm installation.
	// Each must exit 0. Typically "command -v <binary>" or "<binary> --version".
	Verify() []string

	// Skills returns embedded skill files (Vercel SKILL.md convention).
	// Return nil if the dependency ships no skills.
	Skills() fs.FS
}

// Packages groups install-time package-manager lists. Each non-empty
// field becomes one RUN recipe in the generated Dockerfile.
//
// Today only Apt is wired end-to-end: the base image is Debian-family,
// so apt-get is the only manager the Dockerfile generator knows how
// to call. New managers (Apk for Alpine, Brew for macOS, Pacman for
// Arch, …) are added as new fields here *and* as matching cases in
// the generator. Splitting by field is intentional — each manager has
// its own flag shape, cache-cleanup pattern, and dedup semantics, so
// a single flat `[]string` couldn't be reliably translated.
//
// Unknown keys in yaml (e.g. `packages: { apy: [...] }`) are silently
// ignored by the default yaml decoder — a regrettable footgun. When
// a tool declares packages but everything lands under an unknown key,
// the image-build step's verify step still catches it (the binary
// won't exist), but a stricter parse is on the todo list.
type Packages struct {
	// Apt is Debian/Ubuntu apt-get packages. Deduplicated across
	// every tool in the image before one merged `apt-get install`
	// line is emitted.
	Apt []string `yaml:"apt,omitempty"`
}

// InstallSpec describes how to install a tool into a Docker compile.
type InstallSpec struct {
	// Packages groups package-manager installs by manager. See
	// Packages for the supported set.
	Packages Packages

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
