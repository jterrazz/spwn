package cli

import (
	"sort"
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
// built-in shipped with spwn — dependencies (the unified tool/skill
// concept) and runtimes. Used to power the "did you mean X?" hints
// in `spwn check`.
//
// Order: dependencies first (sorted), then runtimes (sorted). Kept
// stable so user-facing hints and the golden tests that pin the
// hint string don't drift when Go's init() order of the runtime
// subpackages changes.
func catalogToolNames() []string {
	deps := dependency.BuiltinTools()
	depNames := make([]string, 0, len(deps))
	for _, t := range deps {
		depNames = append(depNames, t.Name())
	}
	sort.Strings(depNames)

	adapters := runtimes.All()
	rtNames := make([]string, 0, len(adapters))
	for _, a := range adapters {
		if a.Tool != nil {
			rtNames = append(rtNames, a.Tool.Name())
		}
	}
	sort.Strings(rtNames)

	return append(depNames, rtNames...)
}

// supportedRuntimes returns the identifiers of every runtime adapter
// the CLI knows about, taken from catalog/runtimes. Used to validate
// each agent's runtime.backend at `spwn check` time. Both the short
// name ("claude-code") and the catalog ref ("spwn:claude-code") are
// accepted.
func supportedRuntimes() []string {
	adapters := runtimes.All()
	out := make([]string, 0, 2*len(adapters))
	for _, a := range adapters {
		out = append(out, a.Name)
		if a.Tool != nil {
			out = append(out, a.Tool.Name())
		}
	}
	return out
}
