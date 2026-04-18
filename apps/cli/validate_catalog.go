package cli

import (
	"strings"

	clidep "spwn.sh/apps/cli/dependency"
	"spwn.sh/packages/dependency"
	"spwn.sh/packages/runtimes"
)

func init() {
	// Wire the built-in catalog into the install verbs so
	// `spwn install spwn:bogus` can fail with a crisp error
	// instead of silently pinning garbage. The bare-name list
	// (without the spwn: prefix) feeds the CLI resolver so
	// `spwn install qmd` auto-promotes to `spwn install spwn:qmd`.
	clidep.SetCatalogLookup(
		func(ref string) bool {
			for _, t := range dependency.BuiltinTools() {
				if t.Name() == ref {
					return true
				}
			}
			return false
		},
		func() []string {
			out := make([]string, 0, len(dependency.BuiltinTools()))
			for _, t := range dependency.BuiltinTools() {
				out = append(out, strings.TrimPrefix(t.Name(), "spwn:"))
			}
			return out
		},
	)
}

// catalogToolNames returns the @scope/name identifier of every
// built-in shipped with spwn — dependencies (the unified
// tool/skill/runtime-config concept) and runtimes. Used to power
// the "did you mean X?" hints in `spwn check`.
func catalogToolNames() []string {
	adapters := runtimes.All()
	out := make([]string, 0, len(dependency.BuiltinTools())+len(adapters))
	for _, t := range dependency.BuiltinTools() {
		out = append(out, t.Name())
	}
	for _, a := range adapters {
		if a.Tool != nil {
			out = append(out, a.Tool.Name())
		}
	}
	return out
}

// supportedRuntimes returns the identifiers of every runtime adapter
// the CLI knows about, taken from catalog/runtimes. Used to validate
// each agent's runtime.backend at `spwn check` time.
func supportedRuntimes() []string {
	adapters := runtimes.All()
	out := make([]string, 0, 2*len(adapters))
	for _, a := range adapters {
		out = append(out, a.Name)
		if a.CatalogRef != "" {
			out = append(out, a.CatalogRef)
		}
	}
	return out
}
