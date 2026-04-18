package refs

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveCLI normalises a CLI-typed ref to its canonical scheme-form,
// with one layer of convenience the stricter ParseRef does not offer:
// a bare name (no scheme) resolves to "spwn:<name>" when the name
// appears in `catalog`. This is how `spwn init qmd` becomes
// `spwn init spwn:qmd` in docs without losing the explicit-scheme
// invariant inside manifests.
//
// Resolution rules:
//   - Explicit scheme (spwn:, skill:, tool:, hook:, github:) passes
//     through unchanged, including any `@version` suffix.
//   - Bare identifier ([a-z0-9][a-z0-9-]*) that matches a catalog name
//     rewrites to "spwn:<name>" (again preserving `@version`).
//   - Bare identifier with no catalog match returns an error listing
//     the valid catalog entries.
//   - Anything else (uppercase, slashes, legacy `@owner/name`, unknown
//     schemes, empty input) returns a grammar error pointing at the
//     five valid schemes.
//
// This helper is ONLY for CLI input boundaries. Manifest parsers must
// keep calling ParseRef directly — the explicit-scheme grammar there
// is load-bearing.
func ResolveCLI(input string, catalog []string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("ref is empty")
	}

	name, _ := SplitVersion(trimmed)
	parsed := ParseRef(name)

	// Explicit scheme: pass through unchanged. The caller still runs
	// its own existence check (catalog miss / local file missing);
	// ResolveCLI's job ends at grammar.
	switch parsed.Kind {
	case KindSpwnBuiltin, KindRegistry, KindLocalSkill, KindLocalTool, KindLocalHook:
		return trimmed, nil
	}

	// Bare name path: only accept the shape we want docs to show.
	if isBareIdentifier(name) {
		for _, entry := range catalog {
			if entry == name {
				return "spwn:" + trimmed, nil
			}
		}
		return "", fmt.Errorf("%q is not in the catalog.\n%s\n\nhint: pass skill:%s, tool:%s, or hook:%s for a local block", input, formatKnown(catalog), name, name, name)
	}

	// Anything else: malformed under the scheme grammar.
	return "", fmt.Errorf("ref %q is malformed: pass spwn:<name>, skill:<name>, tool:<name>, hook:<name>, or github:<owner>/<repo>", input)
}

// isBareIdentifier reports whether s is a lowercase-kebab identifier
// suitable for catalog lookup: [a-z0-9] then [a-z0-9-]*. Anything else
// (uppercase, whitespace, path separators, colons) falls through to
// the scheme grammar error.
func isBareIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			continue
		case r >= '0' && r <= '9':
			continue
		case r == '-' && i > 0 && i < len(s)-1:
			continue
		default:
			return false
		}
	}
	return true
}

// formatKnown renders the catalog list for the "known:" line of the
// bare-name-miss error. Sorted for stable output, wrapped as a single
// line so the error message stays scannable.
func formatKnown(catalog []string) string {
	if len(catalog) == 0 {
		return "known: (catalog is empty)"
	}
	sorted := make([]string, len(catalog))
	copy(sorted, catalog)
	sort.Strings(sorted)
	return "known: " + strings.Join(sorted, ", ")
}
