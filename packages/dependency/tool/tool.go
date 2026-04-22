package tool

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

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
// Unknown keys in yaml are rejected at parse time via the custom
// UnmarshalYAML below. Without that, `packages: { apy: [...] }`
// would silently install nothing (the image build succeeds because
// apt has nothing to install; tools that expected those packages
// fail their verify step much later with cryptic "command not found"
// errors). Strict parsing surfaces typos in the tool.yaml itself,
// where they can be fixed.
type Packages struct {
	// Apt is Debian/Ubuntu apt-get packages. Deduplicated across
	// every tool in the image before one merged `apt-get install`
	// line is emitted.
	Apt []string `yaml:"apt,omitempty"`
}

// UnmarshalYAML strictly decodes the keyed `install.packages` block:
// only keys matching a Packages field are accepted. Any other key
// (typo like `apy:` / unknown manager like `apk:` before we support
// it) fails the parse with a message that points at the offending
// key so users can fix the manifest directly rather than chasing a
// silent no-op.
func (p *Packages) UnmarshalYAML(node *yaml.Node) error {
	// Allow explicit null / missing blocks (zero-value Packages).
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return nil
	}
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("install.packages: want a mapping (e.g. `packages: { apt: [...] }`), got %s at line %d", nodeKind(node), node.Line)
	}

	// Known keys must stay in sync with the Packages struct fields
	// above. One entry per field, keyed by the struct tag.
	known := map[string]bool{"apt": true}

	// Keys come in pairs (key, value) under Content. Reject anything
	// not in `known` before delegating to the default decoder.
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if !known[keyNode.Value] {
			return fmt.Errorf("install.packages: unknown package manager %q at line %d (supported: apt)", keyNode.Value, keyNode.Line)
		}
	}

	// Decode into a sibling type so we don't recurse back into
	// UnmarshalYAML. Same fields, no methods.
	type packagesRaw Packages
	var raw packagesRaw
	if err := node.Decode(&raw); err != nil {
		return fmt.Errorf("install.packages: %w", err)
	}
	*p = Packages(raw)
	return nil
}

func nodeKind(node *yaml.Node) string {
	switch node.Kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.MappingNode:
		return "mapping"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	}
	return "unknown"
}

// InstallSpec describes how to install a tool into a Docker compile.
type InstallSpec struct {
	// Packages groups package-manager installs by manager. See
	// Packages for the supported set.
	Packages Packages

	// Commands are RUN lines executed as root, in the order declared.
	// Runtime-user config (dotfiles, per-agent settings) does NOT
	// belong here — it's materialised at spawn time by each runtime
	// adapter's DefaultConfigFiles method, which lands files directly
	// under the agent's real HOME (/agents/<name>/) instead of the
	// image's build-time /home/spwn.
	Commands []string

	// Env are ENV key=value directives added to the Dockerfile.
	Env map[string]string

	// Files are paths→content pairs COPYd into the image.
	Files map[string][]byte
}
