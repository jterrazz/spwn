// Package tools aggregates every built-in tool pack into a single
// registrable list. Every tool lives as a YAML manifest under
// catalog/tools/<name>/spwn-tool.yaml and is picked up automatically
// by loadYAMLTools. The Go-based tool layer is gone — adding a new
// tool is a directory + a yaml + a git add, no Go edit required.
//
// Tools that need host-side Go behavior at spawn time (credential
// sync, default config materialisation, prelaunch shell) declare a
// `runtime-provider:` in their manifest and register the Go impl
// via packages/world/internal/runtime/providers.
package pkg

import (
	"fmt"

	ib "spwn.sh/packages/image"
)

// All is populated at package init from every YAML-backed tool in
// the embedded catalog tree. Init panics if any manifest fails to
// parse — a malformed catalog is a programmer error and should fail
// loudly at CLI startup.
var All []ib.Tool

func init() {
	yaml, err := loadYAMLTools()
	if err != nil {
		panic(fmt.Errorf("catalog: load yaml tools: %w", err))
	}
	All = yaml
}

// RegisterDefaults registers every built-in tool into the given
// registry. Returns an error if any tool fails to register
// (typically a naming collision — indicates a programmer error in
// the catalog).
func RegisterDefaults(r *ib.Registry) error {
	for _, t := range All {
		if err := r.Register(t); err != nil {
			return fmt.Errorf("register built-in tool %q: %w", t.Name(), err)
		}
	}
	return nil
}
