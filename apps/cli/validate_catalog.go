package cli

import (
	"spwn.sh/packages/imagebuilder/catalog"
)

// catalogToolNames returns the @scope/name identifier of every
// built-in tool pack shipped with spwn. Used to power the "did you
// mean X?" hints in `spwn check`.
func catalogToolNames() []string {
	out := make([]string, 0, len(catalog.All))
	for _, t := range catalog.All {
		out = append(out, t.Name())
	}
	return out
}

// supportedRuntimes returns the identifiers of every runtime adapter
// the CLI knows about. Currently a single hard-coded entry; when we
// ship additional runtimes this should be replaced with a registry
// lookup.
func supportedRuntimes() []string {
	return []string{"claude-code"}
}
