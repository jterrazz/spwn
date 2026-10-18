package refs

import (
	"fmt"
	"sort"
	"strings"
)

// ResolveCLI normalises a CLI-typed ref to its canonical form, with
// one layer of convenience the stricter ParseRef does not offer: a
// bare name (no source, no type) resolves to "spwn:<name>" when the
// name appears in `catalog`. This is how `spwn install qmd` becomes
// `spwn install spwn:qmd` in docs without losing the explicit-form
// invariant inside manifests.
//
// Resolution rules:
//   - Source-prefixed (`spwn:`, `github:`) passes through unchanged,
//     including any `@version` suffix.
//   - Path-style local (`tool/`, `skill/`, `hook/`, `command/`) passes
//     through unchanged, including any `@version` suffix.
//   - Bare identifier ([a-z0-9][a-z0-9-]*) that matches a catalog name
//     rewrites to "spwn:<name>" (again preserving `@version`).
//   - Bare identifier with no catalog match returns an error listing
//     the valid catalog entries.
//   - Anything else (uppercase, retired colon-form local schemes,
//     legacy `@owner/name`, unknown schemes, empty input) returns a
//     grammar error.
//
// This helper is ONLY for CLI input boundaries. Manifest parsers must
// keep calling ParseRef directly: the explicit-form grammar there is
// load-bearing.
func ResolveCLI(input string, catalog []string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", fmt.Errorf("ref is empty")
	}

	name, _ := SplitVersion(trimmed)
	parsed := ParseRef(name)

	switch parsed.Kind {
	case KindSpwnBuiltin, KindRegistry, KindLocalSkill, KindLocalTool, KindLocalHook, KindLocalCommand:
		return trimmed, nil
	}

	if isBareIdentifier(name) {
		for _, entry := range catalog {
			if entry == name {
				return "spwn:" + trimmed, nil
			}
		}
		return "", fmt.Errorf("%q is not in the catalog.\n%s\n\nhint: pass skill/%s, tool/%s, hook/%s, or command/%s for a local block", input, formatKnown(catalog), name, name, name, name)
	}

	return "", fmt.Errorf("ref %q is malformed: pass spwn:<name>, skill/<name>, tool/<name>, hook/<name>, command/<name>, or github:<owner>/<repo>", input)
}

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

func formatKnown(catalog []string) string {
	if len(catalog) == 0 {
		return "known: (catalog is empty)"
	}
	sorted := make([]string, len(catalog))
	copy(sorted, catalog)
	sort.Strings(sorted)
	return "known: " + strings.Join(sorted, ", ")
}
