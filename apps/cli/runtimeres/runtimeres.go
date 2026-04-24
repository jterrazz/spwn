// Package runtimeres resolves the runtime name for a `spwn up` or
// `spwn build` invocation, layering auth-state awareness on top of
// the pure source.ResolveRuntime resolver.
//
// It is a small leaf package so it can be imported by both the top-
// Level cli package (for `spwn build`) and the cli/world subpackage
// (for `spwn up`) without creating an import cycle through the main
// cli package.
package runtimeres

import (
	"fmt"
	"sort"
	"strings"

	"spwn.sh/packages/auth"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/transpile/source"
)

// Resolve picks the runtime name for a project, with the full CLI
// Precedence cascade:
//
//  1. Explicit override (--runtime / --backend flag), canonicalised.
//  2. Per-agent declaration in agent.yaml (all agents must agree).
//  3. Project-wide default (spwn.yaml#runtime.backend).
//  4. Exactly one authenticated provider → its runtime (silent pick).
//  5. Multiple authenticated providers + user default (auth.yaml
//     `default_provider`) → that provider's runtime (silent pick).
//  6. Hardcoded default "claude-code".
//
// Returns an error when (2) has conflicts OR (4/5) is ambiguous — the
// User is logged into multiple providers and hasn't pinned a backend
// Anywhere. The error names the candidates and suggests how to
// Disambiguate (pin in config, set auth default, or pass --backend).
//
// When src is nil (legacy global-mode spawn), falls straight through
// To the auth-state / hardcoded default cascade.
func Resolve(src *source.ProjectSource, override string) (string, error) {
	declared, err := source.ResolveRuntime(src, override)
	if err != nil {
		return "", err
	}
	if declared != "" {
		return declared, nil
	}
	if proj := projectDefault(src); proj != "" {
		return proj, nil
	}
	return fromAuth()
}

// projectDefault returns the canonical runtime name declared at
// Project scope in spwn.yaml#runtime.backend. Empty when no default
// Is set, the source is nil, or the manifest is absent.
func projectDefault(src *source.ProjectSource) string {
	if src == nil || src.Manifest == nil {
		return ""
	}
	backend := src.Manifest.Runtime.Backend
	if backend == "" {
		return ""
	}
	return source.CanonicalRuntime(backend)
}

// fromAuth returns the runtime name that matches the user's
// Authenticated providers. See Resolve for the surrounding rules.
//
//   - 0 runtime-mapped providers → "claude-code".
//   - 1 runtime-mapped provider → the matching runtime.
//   - 2+ runtime-mapped providers + auth.DefaultProvider set → that
//     provider's runtime (silent pick).
//   - 2+ runtime-mapped providers + no default → disambiguation error.
func fromAuth() (string, error) {
	providerToRuntime := runtimeByProvider()

	// Keep provider identity alongside the runtime name so we can
	// Honor auth.DefaultProvider when the candidate list has more
	// Than one entry.
	type candidate struct {
		provider auth.Provider
		runtime  string
	}
	var cands []candidate
	for _, p := range connectedProviders() {
		if rt, ok := providerToRuntime[string(p)]; ok {
			cands = append(cands, candidate{provider: p, runtime: rt})
		}
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].runtime < cands[j].runtime })

	switch len(cands) {
	case 0:
		return "claude-code", nil
	case 1:
		return cands[0].runtime, nil
	}

	// Multi-provider: let the user's auth.yaml default break the tie
	// Before erroring. Disabled providers never reach the candidate
	// List (ResolveAll honors Disabled), so DefaultProvider pointing
	// At a disabled one is simply ignored.
	if def := auth.DefaultProvider(); def != "" {
		for _, c := range cands {
			if c.provider == def {
				return c.runtime, nil
			}
		}
	}

	names := make([]string, 0, len(cands))
	for _, c := range cands {
		names = append(names, c.runtime)
	}
	return "", fmt.Errorf(
		"multiple providers authenticated (%s) and no runtime pinned. "+
			"Set a default with `spwn auth default <provider>`, pin one in "+
			"agent.yaml#runtime.backend or spwn.yaml#runtime.backend, or "+
			"override with `spwn up --backend %s`",
		strings.Join(names, ", "),
		names[0],
	)
}

// runtimeByProvider builds a provider→runtime map from the registered
// Adapter list. A provider with no registered runtime (e.g. "google"
// today) is simply absent from the map.
func runtimeByProvider() map[string]string {
	out := make(map[string]string, len(runtimes.All()))
	for _, a := range runtimes.All() {
		if a.DefaultProvider != "" {
			out[a.DefaultProvider] = a.Name
		}
	}
	return out
}

// connectedProviders returns the deterministically-ordered list of
// Providers the user has credentials for. Only providers with a
// Non-none credential type are included.
func connectedProviders() []auth.Provider {
	resolved := auth.ResolveAll()
	ordered := []auth.Provider{auth.ProviderAnthropic, auth.ProviderOpenAI}
	var out []auth.Provider
	for _, p := range ordered {
		cred := resolved[p]
		if cred == nil || cred.Type == auth.CredTypeNone {
			continue
		}
		out = append(out, p)
	}
	return out
}
