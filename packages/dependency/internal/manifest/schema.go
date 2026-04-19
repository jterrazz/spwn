// Package manifest is the shared parser for tool.yaml — the
// declarative manifest format that describes a spwn dependency's
// image-build recipe. Catalog entries keep their manifests under
// catalog/<slug>/tools/<name>/tool.yaml (one dir per tool, so a
// single catalog entry can ship multiple tools) and lift skills to
// catalog/<slug>/skills/ so they're shared across the bundle.
// Project-local tools live at spwn/tools/<name>/tool.yaml. All
// paths share this schema.
//
// A dependency is whatever its fields say it is: install steps +
// verify make it a tool; a SKILL.md sibling or content-only body
// makes it a skill. There is no explicit type field — composability
// determines identity.
//
// The parser produces tool.Tool instances (via the adapter in
// packages/dependency/internal/adapters/), so everything downstream —
// registry resolution, dockerfile generation, skill collection — is
// oblivious to whether a given dependency came from Go or YAML.
package manifest

import (
	"gopkg.in/yaml.v3"
)

// Schema is the on-disk shape of tool.yaml. Every field is
// optional so a minimal dependency ("install one thing, verify it's
// there") stays short.
type Schema struct {
	// Name is the dependency identifier (e.g. "spwn:git"). Optional:
	// when empty, the loader derives it from the caller-supplied
	// DefaultName (catalog loader auto-prefixes with "spwn:"; local
	// loader uses the directory basename).
	Name string `yaml:"name"`

	// Version is a semver string or "latest". Required for catalog
	// dependencies; defaults to "0.0.0-local" for project-local ones.
	Version string `yaml:"version"`

	// Title is a human display name for the catalog gallery
	// (e.g. "The Matrix"). Optional — when empty, callers fall back
	// to the slug. Distinct from Name (which is the spwn:<slug>
	// identifier).
	Title string `yaml:"title,omitempty"`

	// Tagline is a short one-line pitch used by the catalog gallery
	// (e.g. "A sandbox with Neo - interactive exploration"). Optional.
	Tagline string `yaml:"tagline,omitempty"`

	// Description is a longer human-readable blurb. Optional.
	Description string `yaml:"description"`

	// Dependencies is a flat list of other refs this one needs. The
	// registry resolves them transitively and topologically sorts
	// the final install order.
	Dependencies []string `yaml:"dependencies"`

	// Worlds is the project-world map for entries that double as
	// installable templates (matrix, startup, …). Opaque here — the
	// init path reads it by copying spwn.yaml verbatim; the install
	// path ignores it because worlds don't make sense as an
	// installable dep (they are a compose-level concept).
	Worlds yaml.Node `yaml:"worlds,omitempty"`

	// Install is the build-time recipe for baking this dependency
	// into the image. All sub-fields are optional — a dependency
	// that only ships skills can leave Install empty entirely.
	Install InstallSection `yaml:"install"`

	// Files is a map of image-target-path → source path relative to
	// this dependency's directory. Contents are read at parse time
	// and compiled into the image via the Dockerfile's COPY layer.
	Files map[string]string `yaml:"files"`

	// Verify is the list of post-build sanity commands. Each must
	// exit 0. Typically "command -v <binary>" or "<binary> --version".
	Verify []string `yaml:"verify"`

	// RuntimeProvider names a host-side Go implementation that
	// handles credential sync, default config file materialisation,
	// and prelaunch shell setup at spawn time. Only runtimes
	// ("spwn:claude-code", "spwn:codex") need this today; a tool
	// that leaves it blank gets no spawn-time hooks. The string is
	// looked up against a Go-side registry — unknown names fail at
	// load time so typos are caught early.
	RuntimeProvider string `yaml:"runtime-provider,omitempty"`
}

// InstallSection mirrors packages/tool.InstallSpec but uses wire-level
// types so the parser is self-contained.
type InstallSection struct {
	// AptPackages are apt-get packages. Deduplicated across every
	// dependency in the image, so ordering here is irrelevant. YAML
	// key is still `packages:` because inside an `install:` block
	// the Debian-family meaning is unambiguous — it's the spwn
	// domain concept that got renamed to dependencies, not this.
	AptPackages []string `yaml:"packages"`

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
