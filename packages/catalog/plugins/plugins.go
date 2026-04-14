// Package plugins collects the built-in spwn plugins.
//
// A plugin is a Tool that also implements image.Plugin: it targets one
// or more runtime backends (e.g. @spwn/claude-code) and can inject
// runtime-specific configuration into the container's runtime config
// file (e.g. ~/.claude/settings.json) at spawn time.
//
// Plugins are sugar for bundling existing primitives with runtime
// config injection — they coexist with plain tools in the agent
// manifest via the dedicated plugins: field. Under the hood both lists
// are resolved through the same image.Registry.
package plugins

import (
	"fmt"

	"spwn.sh/packages/catalog/plugins/mempalace"
	ib "spwn.sh/packages/image"
)

// All is the list of every built-in plugin. Adding a new plugin?
// Import it here and append.
var All = []ib.Tool{
	mempalace.Tool,
}

// RegisterDefaults registers all built-in plugins into the given
// registry. Returns an error on naming collisions (programmer error).
func RegisterDefaults(r *ib.Registry) error {
	for _, p := range All {
		if err := r.Register(p); err != nil {
			return fmt.Errorf("register plugin %q: %w", p.Name(), err)
		}
	}
	return nil
}
