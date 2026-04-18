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
	out := make([]string, 0, len(dependency.BuiltinTools())+len(runtimes.All))
	for _, t := range dependency.BuiltinTools() {
		out = append(out, t.Name())
	}
	for _, t := range runtimes.All {
		out = append(out, t.Name())
	}
	return out
}

// supportedRuntimes returns the identifiers of every runtime adapter
// the CLI knows about, taken from catalog/runtimes. Used to validate
// each agent's runtime.backend at `spwn check` time.
func supportedRuntimes() []string {
	out := make([]string, 0, len(runtimes.All))
	for _, r := range runtimes.All {
		out = append(out, r.Name())
	}
	return out
}
