package cli

import (
	"spwn.sh/catalog/plugins"
	"spwn.sh/catalog/runtimes"
	"spwn.sh/catalog/tools"
)

// catalogToolNames returns the @scope/name identifier of every
// built-in tool + runtime + plugin shipped with spwn. Used to power the
// "did you mean X?" hints in `spwn check`. Plugins share the tool
// resolution pipeline, so the validator treats them as tools for
// existence checks.
func catalogToolNames() []string {
	out := make([]string, 0, len(tools.All)+len(runtimes.All)+len(plugins.All))
	for _, t := range tools.All {
		out = append(out, t.Name())
	}
	for _, t := range runtimes.All {
		out = append(out, t.Name())
	}
	for _, p := range plugins.All {
		out = append(out, p.Name())
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
