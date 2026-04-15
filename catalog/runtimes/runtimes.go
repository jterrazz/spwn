// Package runtimes collects the agent runtimes spwn ships with.
// A runtime is the adapter that actually executes an agent's thoughts
// (claude-code, codex, ...) — as distinct from tools, which are the
// things an agent reaches for while thinking.
//
// Every runtime lives as a YAML manifest under
// catalog/runtimes/<name>/spwn-tool.yaml and is picked up automatically
// by loadYAMLRuntimes. Runtimes that need host-side spawn-time
// behavior (credential sync, default config files, prelaunch shell)
// declare `runtime-provider: <name>` in their manifest; the spawn
// pipeline looks up the Go impl by that name via
// packages/world/internal/runtime.Get.
package runtimes

import (
	"fmt"

	ib "spwn.sh/packages/image"
)

// All is populated at package init from every YAML-backed runtime.
var All []ib.Tool

func init() {
	yaml, err := loadYAMLRuntimes()
	if err != nil {
		panic(fmt.Errorf("catalog: load yaml runtimes: %w", err))
	}
	All = yaml
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
