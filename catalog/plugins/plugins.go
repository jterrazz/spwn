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
//
// Every plugin lives as a YAML manifest under
// catalog/plugins/<name>/spwn-tool.yaml with a `plugin:` section that
// declares its target runtimes and the config snippets to inject.
package plugins

import (
	"fmt"

	ib "spwn.sh/packages/image"
)

// All is populated at package init from every YAML-backed plugin.
var All []ib.Tool

func init() {
	yaml, err := loadYAMLPlugins()
	if err != nil {
		panic(fmt.Errorf("catalog: load yaml plugins: %w", err))
	}
	All = yaml
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
