// Package spwn is the built-in catalog adapter for dependency
// resolution. Every entry lives as a YAML manifest under
// /catalog/<name>/spwn.yaml (mirrored into ./content at generate
// time) and is picked up automatically by loadYAMLTools.
//
// Adding a new dependency is a directory + a yaml + a git add, no
// Go edit required. Runtimes that need host-side Go behavior at
// spawn time (credential sync, default config materialisation,
// prelaunch shell) are kept Go-only and registered via
// packages/world/runtime/.
package spwn

import (
	"fmt"

	ib "spwn.sh/packages/compile"
	"spwn.sh/packages/dependency"
)

// All is populated at package init from every YAML-backed tool in
// the embedded catalog tree. Init panics if any manifest fails to
// parse — a malformed catalog is a programmer error and should fail
// loudly at CLI startup.
var All []dependency.Tool

func init() {
	yaml, err := loadYAMLTools()
	if err != nil {
		panic(fmt.Errorf("spwn adapter: load yaml tools: %w", err))
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
