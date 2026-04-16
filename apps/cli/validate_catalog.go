package cli

import (
	"spwn.sh/apps/cli/pack"
	packs "spwn.sh/catalog/packs"
	"spwn.sh/catalog/runtimes"
)

func init() {
	// Wire the built-in catalog into the install verbs so
	// `spwn plugin install @spwn/bogus` can fail with a crisp error
	// instead of silently pinning garbage. Lives here (not in
	// plugin.init()) so the pack package stays free of a catalog import.
	pack.SetCatalogLookup(func(ref string) bool {
		for _, t := range packs.All {
			if t.Name() == ref {
				return true
			}
		}
		return false
	})
}

// catalogToolNames returns the @scope/name identifier of every
// built-in package shipped with spwn — packages (the unified
// tool/plugin/skill concept) and runtimes. Used to power the
// "did you mean X?" hints in `spwn check`.
func catalogToolNames() []string {
	out := make([]string, 0, len(packs.All)+len(runtimes.All))
	for _, t := range packs.All {
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
