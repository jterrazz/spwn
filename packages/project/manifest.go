// Package project owns everything about a spwn project's declarative
// config: parsing spwn.yaml, scaffolding a new project, discovering the
// project from a child directory, and validating the resulting tree.
//
// A spwn project is a directory tree that contains:
//
//	my-project/
//	├── spwn.yaml           - the manifest (committed)
//	├── spwn/               - committed project assets
//	│   ├── agents/<name>/
//	│   ├── tools/<name>/
//	│   └── skills/<name>.md
//	└── .spwn/              - gitignored local state
//	    └── state.json
//
// Worlds are no longer separate yaml files — they live as inline map
// entries under spwn.yaml#worlds. Agents are the source of truth for
// the project roster: every directory under spwn/agents/ that the
// manifest's worlds reference is "deployable", everything else is
// considered an orphan agent.
package project

import (
	"spwn.sh/packages/project/internal/discovery"
	intmanifest "spwn.sh/packages/project/internal/manifest"
	"spwn.sh/packages/project/internal/scaffold"
	"spwn.sh/packages/project/internal/validate"
)

// Manifest is the parsed spwn.yaml content.
type Manifest = intmanifest.Manifest

// World is one inline world entry under spwn.yaml#worlds.
type World = intmanifest.World

// Runtime is the project-wide runtime block from spwn.yaml.
type Runtime = intmanifest.Runtime

// Automation is one trigger → agent wakeup binding inside a world.
// See packages/project/internal/manifest for full schema docs.
type Automation = intmanifest.Automation

// Trigger is the event source of an Automation. Exactly one of
// Cron / FS is set.
type Trigger = intmanifest.Trigger

// FSTrigger configures a filesystem-watch event source.
type FSTrigger = intmanifest.FSTrigger

// Duration wraps time.Duration with a YAML codec that accepts the
// natural "10s" / "1m" / "1h30m" string form.
type Duration = intmanifest.Duration

// Project is a loaded spwn project - manifest plus resolved references
// to the agents it declares.
type Project struct {
	// Root is the absolute path to the project root (the directory
	// that contains spwn.yaml).
	Root string

	// ManifestPath is the absolute path to the spwn.yaml file.
	ManifestPath string

	// Manifest is the parsed content.
	Manifest *Manifest

	// Agents is one entry per *deployable* agent (referenced by at
	// least one world in the manifest). Orphan agents — directories
	// under spwn/agents/ not listed in any world — are surfaced via
	// OrphanAgents.
	Agents []AgentRef

	// OrphanAgents are agent directories on disk that no world in the
	// manifest references. They are listed informationally; not
	// runnable until added to a world.
	OrphanAgents []AgentRef
}

// AgentRef points at one agent directory under spwn/agents/.
type AgentRef struct {
	Name   string // directory basename
	Path   string // absolute path to ./spwn/agents/<name>/
	Exists bool
}

// InitOpts configures Init. Zero-value is fine - all fields have sane
// defaults.
type InitOpts struct {
	// Name overrides the default (filepath.Base of dir). Leave empty
	// to use the directory name.
	Name string

	// Force allows Init to overwrite an existing spwn.yaml.
	Force bool

	// NoGitignore skips appending .spwn/ to .gitignore.
	NoGitignore bool

	// Backend, when non-empty, is written to the scaffolded agent's
	// `runtime.backend:` line (e.g. "spwn:claude-code", "spwn:codex").
	// Empty leaves the scaffold backend-agnostic so `spwn up` can
	// resolve the runtime at spawn time.
	Backend string
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
		Backend:     opts.Backend,
	})
}

// AddAgentToManifest atomically inserts a new world entry into
// spwn.yaml that deploys the named agent on its own. Used by
// `spwn agent create <name>` to keep the auto-world wired up.
func AddAgentToManifest(manifestPath, agentName string) error {
	return scaffold.AddAgentWorld(manifestPath, agentName)
}

// RemoveAgentFromManifest strips every reference to the named agent
// from spwn.yaml#worlds and drops any worlds left empty. Used by
// `spwn agent rm` to keep the manifest consistent with disk state.
func RemoveAgentFromManifest(manifestPath, agentName string) error {
	return scaffold.RemoveAgentFromManifest(manifestPath, agentName)
}

// AddWorldOpts configures AddWorld.
type AddWorldOpts = scaffold.AddWorldOpts

// AddWorld declares a new world entry in spwn.yaml. Idempotent: a
// no-op if an entry with that name already exists. Used by
// `spwn world create <name>`.
func AddWorld(manifestPath, name string, opts AddWorldOpts) error {
	return scaffold.AddWorld(manifestPath, name, opts)
}

// RemoveWorld drops the named entry from spwn.yaml#worlds. Returns
// ErrWorldNotFound if no such entry exists. Used by
// `spwn world rm <name>`.
func RemoveWorld(manifestPath, name string) error {
	return scaffold.RemoveWorld(manifestPath, name)
}

// AppendGitignore adds a `.spwn/` entry to <root>/.gitignore, creating
// the file when missing and idempotent when already present. Used by
// Init and by the catalog example installer so `spwn init spwn:<slug>`
// drops a consistent gitignore too.
func AppendGitignore(root string) error {
	return scaffold.AppendGitignore(root)
}

// ErrWorldNotFound is returned by RemoveWorld when the named world
// is not declared in spwn.yaml.
var ErrWorldNotFound = scaffold.ErrWorldNotFound

// IsReservedAgentName reports whether the given name collides with
// a `spwn agent <subcommand>` reserved verb. CLI callers should
// reject such names at creation time — before a directory is ever
// written — so the user never lands in a half-broken state.
func IsReservedAgentName(name string) bool {
	return validate.IsReservedAgentName(name)
}

// ReservedAgentNames returns the sorted list of reserved agent
// names for display in error messages.
func ReservedAgentNames() []string {
	return validate.ReservedAgentNames()
}

// IsValidAgentName reports whether the given string is a syntactically
// valid agent name (slug regex + length cap, same rules the manifest
// enforces for world names). CLI callers should reject invalid names
// at creation time before writing anything to disk.
func IsValidAgentName(name string) bool {
	return validate.IsValidAgentName(name)
}

// MaxAgentNameLen is the upper bound callers enforce on agent-name
// length (in bytes). Mirrors validate.MaxAgentNameLen so CLI code
// doesn't reach into the internal validate package.
const MaxAgentNameLen = validate.MaxAgentNameLen

// IsValidProjectName reports whether the given string matches the
// manifest's project-name regex.
func IsValidProjectName(name string) bool {
	return validate.IsValidProjectName(name)
}

// ValidateOpts configures Validate. Zero value is valid and skips
// catalog-backed rules (tool existence, runtime support). Callers
// should populate this from catalog/tools + catalog/runtimes for the
// richest error messages, including "did you mean X?" hints.
type ValidateOpts struct {
	// BuiltinTools is the authoritative list of @scope/name package
	// identifiers the host knows how to build. When empty, package
	// existence falls back to a simple spwn:* prefix heuristic.
	BuiltinTools []string

	// SupportedRuntimes is the list of runtime identifiers the host
	// can actually spawn (e.g. "spwn:claude-code"). When empty,
	// runtime validity is not checked.
	SupportedRuntimes []string

	// HookEventsByRuntime maps each runtime adapter's short name
	// (e.g. "claude-code") to the hook events it actually fires.
	// Used to flag hooks whose `event:` would silently no-op for
	// the agent's runtime. When empty, the event-support check is
	// skipped entirely (golden tests + scaffold paths).
	HookEventsByRuntime map[string][]string
}

// Validate runs every validation rule against the project and returns
// the collected issues. It never returns an error - all problems
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
	in := validate.Input{
		Root:                p.Root,
		Manifest:            p.Manifest,
		BuiltinTools:        o.BuiltinTools,
		SupportedRuntimes:   o.SupportedRuntimes,
		HookEventsByRuntime: o.HookEventsByRuntime,
	}
	for _, a := range p.Agents {
		in.AgentRefs = append(in.AgentRefs, validate.AgentRef{
			Name: a.Name, Path: a.Path, Exists: a.Exists,
		})
	}
	for _, a := range p.OrphanAgents {
		in.OrphanRefs = append(in.OrphanRefs, validate.AgentRef{
			Name: a.Name, Path: a.Path, Exists: a.Exists,
		})
	}
	return validate.Run(in)
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
	agents, orphans := resolveRefs(root, m)
	return &Project{
		Root:         root,
		ManifestPath: manifestPath,
		Manifest:     m,
		Agents:       agents,
		OrphanAgents: orphans,
	}, nil
}
