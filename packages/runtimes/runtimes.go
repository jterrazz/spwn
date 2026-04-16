// Package runtimes collects the agent runtimes spwn ships with.
// A runtime is the adapter that actually executes an agent's thoughts
// (claude-code, codex, ...) — as distinct from tools, which are the
// things an agent reaches for while thinking.
//
// Runtimes stay in Go (unlike tools, which live as spwn.yaml) because
// they carry non-trivial spawn-time behavior — credential sync, default
// config materialisation, prelaunch shell, authentication flows — that
// declarative YAML can't express cleanly.
package runtimes

import (
	"fmt"

	"spwn.sh/packages/runtimes/claude_code"
	"spwn.sh/packages/runtimes/codex"
	ib "spwn.sh/packages/image"
)

// All is the list of every built-in runtime. Adding a new runtime?
// Import it and append.
var All = []ib.Tool{
	claude_code.Tool,
	codex.Tool,
}

// RegisterDefaults registers all built-in runtimes into the given registry.
func RegisterDefaults(r *ib.Registry) error {
	for _, t := range All {
		if err := r.Register(t); err != nil {
			return fmt.Errorf("register built-in runtime %q: %w", t.Name(), err)
		}
	}
	return nil
}
