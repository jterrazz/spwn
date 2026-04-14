// Package manifest owns everything about a spwn project's declarative
// config: parsing spwn.yaml, scaffolding a new project, discovering the
// project from a child directory, and validating the resulting tree.
//
// A spwn project is a directory tree that contains:
//
//	my-project/
//	├── spwn.yaml           — the manifest (committed)
//	├── spwn/               — committed project assets
//	│   ├── agents/<name>/
//	│   ├── worlds/<name>.yaml
//	│   ├── tools/<name>/
//	│   └── skills/<name>.md
//	└── .spwn/              — gitignored local state
//	    └── state.json
//
// The manifest declares which world this project spawns and which
// agents it contains. Everything referenced by name is resolved
// relative to the project root — there is no cross-project lookup.
package manifest

import (
	"spwn.sh/packages/manifest/internal/build"
	"spwn.sh/packages/manifest/internal/discovery"
	intmanifest "spwn.sh/packages/manifest/internal/manifest"
	"spwn.sh/packages/manifest/internal/scaffold"
	"spwn.sh/packages/manifest/internal/validate"
)

// Manifest is the parsed spwn.yaml content.
type Manifest = intmanifest.Manifest

// Project is a loaded spwn project — manifest plus resolved references
// to the agents and world it declares. Existence of each referenced
// resource is recorded in the *Ref fields so callers can warn without
// re-walking the tree.
type Project struct {
	// Root is the absolute path to the project root (the directory
	// that contains spwn.yaml).
	Root string

	// ManifestPath is the absolute path to the spwn.yaml file.
	ManifestPath string

	// Manifest is the parsed content.
	Manifest *Manifest

	// Agents is one entry per agent declared in the manifest, with
	// its resolved filesystem path and whether the path exists.
	Agents []AgentRef

	// World is the resolved reference to the world config.
	World WorldRef
}

// AgentRef points at one agent declared in the manifest.
type AgentRef struct {
	Name   string // as declared in spwn.yaml
	Path   string // absolute path to ./spwn/agents/<name>/
	Exists bool
}

// WorldRef points at the world config declared in the manifest.
type WorldRef struct {
	Name   string // as declared in spwn.yaml
	Path   string // absolute path to ./spwn/worlds/<name>.yaml
	Exists bool
}

// InitOpts configures Init. Zero-value is fine — all fields have sane
// defaults.
type InitOpts struct {
	// Name overrides the default (filepath.Base of dir). Leave empty
	// to use the directory name.
	Name string

	// Force allows Init to overwrite an existing spwn.yaml.
	Force bool

	// NoGitignore skips appending .spwn/ to .gitignore.
	NoGitignore bool
}

// Issue is one finding produced by Validate.
type Issue = validate.Issue

// Level is the severity of an Issue.
type Level = validate.Level

const (
	LevelError   = validate.LevelError
	LevelWarning = validate.LevelWarning
	LevelInfo    = validate.LevelInfo
)

// Find walks up from startPath looking for a directory that contains
// spwn.yaml. Returns the loaded project if found, or (nil, nil) if no
// manifest exists anywhere between startPath and the filesystem root.
// A non-nil error means the manifest was found but failed to parse.
func Find(startPath string) (*Project, error) {
	manifestPath, root, found, err := discovery.Find(startPath)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return loadAt(manifestPath, root)
}

// Load parses an explicit spwn.yaml path. Use this when the caller
// already knows where the manifest lives (e.g. tests).
func Load(manifestPath string) (*Project, error) {
	root, err := discovery.RootFor(manifestPath)
	if err != nil {
		return nil, err
	}
	return loadAt(manifestPath, root)
}

// Init scaffolds a new spwn project at dir. Errors if dir already
// contains a spwn.yaml unless opts.Force is set.
func Init(dir string, opts InitOpts) error {
	return scaffold.Init(dir, scaffold.Opts{
		Name:        opts.Name,
		Force:       opts.Force,
		NoGitignore: opts.NoGitignore,
	})
}

// ValidateOpts configures Validate. Zero value is valid and skips
// catalog-backed rules (tool existence, runtime support). Callers
// should populate this from the imagebuilder catalog for the richest
// error messages, including "did you mean X?" hints.
type ValidateOpts struct {
	// BuiltinTools is the authoritative list of @scope/name tool
	// identifiers the host knows how to build. When empty, tool
	// existence falls back to a simple @spwn/* prefix heuristic.
	BuiltinTools []string

	// SupportedRuntimes is the list of runtime identifiers the host
	// can actually spawn (e.g. "claude-code"). When empty, runtime
	// validity is not checked.
	SupportedRuntimes []string
}

// Validate runs every validation rule against the project and returns
// the collected issues. It never returns an error — all problems
// surface as Issues with a Level. Callers decide what to do with
// warnings vs errors.
func Validate(p *Project, opts ...ValidateOpts) []Issue {
	if p == nil {
		return nil
	}
	var o ValidateOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return validate.Run(validate.Input{
		Root:              p.Root,
		Manifest:          p.Manifest,
		AgentPaths:        agentPaths(p.Agents),
		WorldPath:         p.World.Path,
		WorldExists:       p.World.Exists,
		AgentExists:       agentExistence(p.Agents),
		BuiltinTools:      o.BuiltinTools,
		SupportedRuntimes: o.SupportedRuntimes,
	})
}

// BuildResult is the outcome of a successful Build.
type BuildResult = build.Result

// BuildMetadata is the shape of .spwn/build/build.json.
type BuildMetadata = build.Metadata

// BuildOpts configures Build.
type BuildOpts struct {
	// ImageDigest pins the Docker image produced for this build.
	// Empty means "no image was built" — the artifact is still
	// valid, it just records no image reference.
	ImageDigest string
}

// Build flattens the project into a reproducible artifact at
// <projectRoot>/.spwn/build/. Every file the runtime will read is
// copied in. The resulting directory is self-contained and safe to
// tar and ship.
//
// Build does NOT validate. Callers should run Validate first and
// abort on errors — Build will happily flatten a broken project,
// which is useful when the user wants to inspect the artifact for
// debugging but not when they want a spawnable result.
func Build(p *Project, opts ...BuildOpts) (*BuildResult, error) {
	if p == nil {
		return nil, nil
	}
	var o BuildOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	return build.Build(build.Opts{
		Root:        p.Root,
		Manifest:    p.Manifest,
		AgentPaths:  agentPaths(p.Agents),
		WorldPath:   p.World.Path,
		ImageDigest: o.ImageDigest,
	})
}

// LoadBuildMetadata reads an existing build.json from a project's
// .spwn/build/ directory. Returns (nil, nil) when no build is present.
func LoadBuildMetadata(p *Project) (*BuildMetadata, error) {
	if p == nil {
		return nil, nil
	}
	return build.LoadMetadata(p.Root + "/.spwn/build")
}

// HasErrors returns true if any issue is LevelError.
func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Level == LevelError {
			return true
		}
	}
	return false
}

func loadAt(manifestPath, root string) (*Project, error) {
	m, err := intmanifest.LoadPath(manifestPath)
	if err != nil {
		return nil, err
	}
	agents, world := resolveRefs(root, m)
	return &Project{
		Root:         root,
		ManifestPath: manifestPath,
		Manifest:     m,
		Agents:       agents,
		World:        world,
	}, nil
}

func agentPaths(refs []AgentRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Path
	}
	return out
}

func agentExistence(refs []AgentRef) []bool {
	out := make([]bool, len(refs))
	for i, r := range refs {
		out[i] = r.Exists
	}
	return out
}
