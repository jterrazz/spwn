// Package runtimes collects the agent runtimes spwn ships with.
// A runtime is the adapter that actually executes an agent's thoughts
// (claude-code, codex, …) — as distinct from tools, which are the
// things an agent reaches for while thinking.
//
// A runtime has up to three orthogonal facets, each bundled into a
// single Adapter:
//
//   - Tool:   image-build recipe (apt, curl, npm install, user-side config).
//   - Render: source → Tree renderer (CLAUDE.md, settings.json, …).
//   - Spawn:  host-side spawn-time behavior (BuildCommand, credential
//     sync, prelaunch shell, default configs, container config path).
//
// Each runtime lives as a subpackage under packages/runtimes/<name>/
// and registers its Adapter at init time via Register. The top-level
// package does NOT import subpackages — callers blank-import each
// runtime they want to enable (apps/cli, architect, tests). This
// mirrors the database/sql driver pattern and keeps the dependency
// graph acyclic.
//
// Runtimes stay in Go (unlike tools, which live as spwn.yaml) because
// they carry non-trivial host-side behavior — credential sync, default
// config materialisation, prelaunch shell, auth flows — that YAML
// can't express cleanly.
package runtimes

import (
	"fmt"

	"spwn.sh/packages/dependency/tool"
)

// RegisterDefaults registers every built-in runtime's install recipe
// (Adapter.Tool) into the given tool registry. Runtimes with no Tool
// facet (renderer-only adapters) are skipped silently. Returns the
// first registration error (typically a name collision — a programmer
// error in the catalog).
func RegisterDefaults(r tool.Registry) error {
	for _, a := range all {
		if a.Tool == nil {
			continue
		}
		if err := r.Register(a.Tool); err != nil {
			return fmt.Errorf("register built-in runtime %q: %w", a.Name, err)
		}
	}
	return nil
}
