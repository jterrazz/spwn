// Package runtimes collects the agent runtimes spwn ships with.
// A runtime is the adapter that actually executes an agent's thoughts
// (claude-code, codex, ...) — as distinct from tools, which are the
// things an agent reaches for while thinking.
//
// Most runtimes live as YAML manifests under catalog/runtimes/<name>/
// spwn-tool.yaml and are picked up automatically by loadYAMLRuntimes.
// A shrinking set still live as Go files and are hand-listed in
// goRuntimes — those are being ported to YAML.
package runtimes

import (
	"fmt"

	"spwn.sh/catalog/runtimes/claude_code"
	ib "spwn.sh/packages/image"
)

// goRuntimes is the transitional list of runtimes still in Go form.
var goRuntimes = []ib.Tool{
	claude_code.Tool,
}

// All is the union of YAML-backed and Go-backed runtimes.
var All []ib.Tool

func init() {
	yaml, err := loadYAMLRuntimes()
	if err != nil {
		panic(fmt.Errorf("catalog: load yaml runtimes: %w", err))
	}
	All = append(All, goRuntimes...)
	All = append(All, yaml...)
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
