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
//  4. Authenticated provider — exactly one provider is connected
//     and maps to a registered runtime (via Adapter.DefaultProvider).
//  5. Hardcoded default "claude-code".
//
// Returns an error when (2) has conflicts OR (4) is ambiguous — the
// user is logged into multiple providers and hasn't pinned a backend
// anywhere. The error names the candidates and suggests how to
// disambiguate.
//
// When src is nil (legacy global-mode spawn), falls straight through
// to the auth-state / hardcoded default cascade.
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
//   - 2+ runtime-mapped providers → error with a disambiguation hint.
func fromAuth() (string, error) {
	providerToRuntime := runtimeByProvider()

	candidates := make([]string, 0, 2)
	for _, p := range connectedProviders() {
		if rt, ok := providerToRuntime[string(p)]; ok {
			candidates = append(candidates, rt)
		}
	}
	sort.Strings(candidates)

	switch len(candidates) {
	case 0:
		return "claude-code", nil
	case 1:
		return candidates[0], nil
	}
	return "", fmt.Errorf(
		"multiple providers authenticated (%s) and no runtime pinned. "+
			"Pin one in agent.yaml#runtime.backend (per agent) or "+
			"spwn.yaml#runtime.backend (project-wide), or override with "+
			"`spwn up --backend %s`",
		strings.Join(candidates, ", "),
		candidates[0],
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
