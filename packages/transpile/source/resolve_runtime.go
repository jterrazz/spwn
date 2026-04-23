package source

import (
	"fmt"
	"sort"
	"strings"
)

// runtimeCanonical maps catalog-style runtime refs (e.g.
// "spwn:claude-code") to the short names used by the compile runtime
// registry (e.g. "claude-code"). Agents declare their runtime via
// `runtime.backend: spwn:claude-code` to match the tool-catalog
// namespace, but transpile.Register uses the short name. This map is
// the one-way bridge between the two.
var runtimeCanonical = map[string]string{
	"spwn:claude-code": "claude-code",
	"spwn:codex":       "codex",
}

// canonicalRuntime returns the short registry name for a runtime
// reference. Unknown names pass through unchanged, on the assumption
// they are already canonical (or will fail later at Compile time with
// a clearer "unknown runtime" error).
func canonicalRuntime(name string) string {
	if mapped, ok := runtimeCanonical[name]; ok {
		return mapped
	}
	return name
}

// ResolveRuntime picks the runtime name declared by a project.
// Precedence:
//
//  1. Explicit override (CLI flag) — canonicalised then returned.
//  2. Per-agent runtime (all agents must agree, after canonicalisation).
//  3. Empty string — "no preference declared". The caller chooses
//     how to break the tie (consult auth state, hardcode a default, …).
//
// Returns an error if agents declare conflicting runtimes and no
// override is set; the user must disambiguate with --runtime.
//
// The empty-on-no-preference contract (vs the old "claude-code"
// hardcoded fallback) lets the CLI layer consult auth state before
// silently defaulting to claude-code. Callers that just need any
// valid runtime (e.g. tests, tree-only builds) can still substitute
// "claude-code" themselves after seeing the empty return.
func ResolveRuntime(src *ProjectSource, override string) (string, error) {
	if override != "" {
		return canonicalRuntime(override), nil
	}
	if src == nil {
		return "", nil
	}

	seen := map[string]struct{}{}
	for _, a := range src.Agents {
		if a.Config.Runtime.Backend == "" {
			continue
		}
		seen[canonicalRuntime(a.Config.Runtime.Backend)] = struct{}{}
	}

	switch len(seen) {
	case 0:
		return "", nil
	case 1:
		for name := range seen {
			return name, nil
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return "", fmt.Errorf(
		"agents declare conflicting runtimes: %s. Pass --runtime to disambiguate",
		strings.Join(names, ", "),
	)
}
