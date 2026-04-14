// Package runtimes collects the agent runtimes spwn ships with.
// A runtime is the adapter that actually executes an agent's thoughts
// (claude-code, codex, ...) - as distinct from tools, which are the
// things an agent reaches for while thinking.
package runtimes

import (
	"fmt"

	"spwn.sh/catalog/runtimes/architect"
	"spwn.sh/catalog/runtimes/claude_code"
	"spwn.sh/catalog/runtimes/codex"
	ib "spwn.sh/packages/imagebuilder"
)

// All is the list of every built-in runtime.
// Adding a new runtime? Import it and add it here.
var All = []ib.Tool{
	claude_code.Tool,
	codex.Tool,
	architect.Tool,
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
