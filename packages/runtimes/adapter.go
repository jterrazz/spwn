package runtimes

import (
	"spwn.sh/packages/dependency/tool"
	"spwn.sh/packages/transpile"
)

// Adapter is the umbrella contract for a runtime. A runtime has up to
// three orthogonal facets: an install recipe (Tool), a source→Tree
// renderer (Render), and a host-side spawn-time adapter (Spawn). Each
// is optional — a YAML-first runtime may ship Tool only; a pure
// renderer may ship Render only; a stub runtime may ship none and
// serve purely as a declaration.
//
// Concrete runtimes live under packages/runtimes/<name>/ and register
// themselves at init() time via Register. The top-level package never
// imports subpackages directly, so there is no import cycle — callers
// that want the built-in set blank-import each subpackage (or the
// convenience umbrella) they need.
type Adapter struct {
	// Name is the short runtime identifier used everywhere in
	// config, CLI flags, and registry lookups. Example:
	// "claude-code", "codex".
	Name string

	// CatalogRef is how this runtime appears as a tool ref in the
	// catalog and in agent dependencies. Example: "spwn:claude-code".
	CatalogRef string

	// DefaultProvider names the auth provider this runtime uses by
	// default ("anthropic", "openai", …). Empty means the runtime
	// does not require an auth provider or supports multiple without
	// a clear default. Purely informational today; consumed by UI
	// surfaces and future auth-routing logic.
	DefaultProvider string

	// Tool is the image-build recipe for this runtime (apt packages,
	// install commands, user commands, env). nil when the runtime
	// does not ship an install step — e.g. a renderer-only adapter
	// or a stub declaration.
	Tool tool.Tool

	// Render transforms a source project into a per-agent/world Tree
	// of rendered files. nil when the runtime has no renderer (codex
	// today), in which case transpile.Compile with this runtime's
	// name will fail.
	Render transpile.Runtime

	// Spawn is the host-side spawn-time adapter (BuildCommand,
	// credential sync, prelaunch shell, default configs, container
	// config path). nil when the runtime is purely declarative.
	Spawn Spawner
}

// all is the package-global list of registered adapters, populated
// via Register at each subpackage's init time.
var all []Adapter

// Register appends an Adapter to the global registry. Also registers
// the Adapter's facets in their individual registries (Spawn in the
// spawner registry; Render in the transpile registry). Typically
// called once per runtime subpackage from init().
func Register(a Adapter) {
	all = append(all, a)
	if a.Spawn != nil {
		RegisterSpawner(a.Spawn)
	}
	if a.Render != nil {
		transpile.Register(a.Render)
	}
}

// All returns every registered adapter. Callers must not mutate the
// returned slice.
func All() []Adapter { return all }

// Get returns the adapter with the given name, and ok=false when no
// such runtime is registered.
func Get(name string) (Adapter, bool) {
	for _, a := range all {
		if a.Name == name {
			return a, true
		}
	}
	return Adapter{}, false
}

// Names returns the sorted-insertion-order list of registered runtime
// names. Used by the CLI to render human error messages ("unknown
// runtime; available: …").
func Names() []string {
	out := make([]string, 0, len(all))
	for _, a := range all {
		out = append(out, a.Name)
	}
	return out
}
