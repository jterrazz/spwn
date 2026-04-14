// Ref parsing and resolution for tool/skill/profile references.
//
// Spwn projects reference packs in three ways:
//
//  1. Local — a bare name, resolved against ./spwn/tools/<name>/ on disk.
//  2. @spwn/<name> — a built-in pack, looked up in the BuiltinTools catalog.
//  3. @<owner>/<name> — a remote registry pack. Not yet implemented;
//     surfaced as an explicit error so users aren't told their ref is
//     a typo.
//
// ParseRef classifies a ref string and ResolveTool answers whether
// the target exists (or, for remote refs, returns a distinct
// "unsupported" result the validator can report with its own wording).
//
// ParseRef expects callers to strip any `@version` suffix first via
// splitToolVersion — version parsing lives in validate.go and is the
// single source of truth for that.

package validate

import (
	"os"
	"path/filepath"
	"strings"
)

// RefKind classifies a tool/skill/profile reference.
type RefKind int

const (
	// RefLocal is a bare name resolved against <root>/spwn/tools/<name>/.
	RefLocal RefKind = iota
	// RefSpwnBuiltin is a @spwn/<name> pack from the BuiltinTools catalog.
	RefSpwnBuiltin
	// RefRegistry is @<owner>/<name> with owner != "spwn" — a future
	// remote-registry ref, not yet supported.
	RefRegistry
)

// Ref is a parsed, classified reference.
type Ref struct {
	Raw   string
	Kind  RefKind
	Owner string // "" for local, "spwn" for builtin, user/org for registry
	Name  string // pack name without scope
}

// ParseRef classifies a tool/skill/profile reference. The input is
// trimmed of surrounding whitespace. Callers should strip any
// @version suffix first (see splitToolVersion); ParseRef itself
// does not touch versioning.
//
// Rules:
//   - No leading "@": RefLocal, Owner="", Name=trimmed.
//   - "@spwn/<name>": RefSpwnBuiltin, Owner="spwn", Name=<name>.
//   - "@<owner>/<name>" (owner != spwn): RefRegistry.
//   - Malformed "@" or "@<owner>" without a slash: RefRegistry with
//     empty Name — callers should reject it as malformed.
func ParseRef(s string) Ref {
	raw := s
	s = strings.TrimSpace(s)

	if !strings.HasPrefix(s, "@") {
		return Ref{Raw: raw, Kind: RefLocal, Name: s}
	}

	// Strip the leading "@" and split on the first "/".
	rest := s[1:]
	slash := strings.Index(rest, "/")
	if slash < 0 {
		// "@" or "@foo" with no slash — malformed registry ref.
		return Ref{Raw: raw, Kind: RefRegistry, Owner: rest, Name: ""}
	}
	owner := rest[:slash]
	name := rest[slash+1:]
	if owner == "spwn" {
		return Ref{Raw: raw, Kind: RefSpwnBuiltin, Owner: "spwn", Name: name}
	}
	return Ref{Raw: raw, Kind: RefRegistry, Owner: owner, Name: name}
}

// ResolveResult is the tri-state outcome of resolving a Ref.
type ResolveResult int

const (
	// ResolveOK means the ref points to something real.
	ResolveOK ResolveResult = iota
	// ResolveNotFound means the ref looks valid but the target is
	// missing (typo, uninstalled local pack, unknown builtin).
	ResolveNotFound
	// ResolveRegistryUnsupported means the ref is a @<owner>/<name>
	// registry ref other than @spwn — not yet implemented.
	ResolveRegistryUnsupported
)

// ResolveTool answers whether a Ref resolves to a real pack. It
// takes the already-parsed Ref so callers can branch on Kind without
// re-parsing. `builtin` is the set of known @spwn/* pack identifiers
// as returned by the catalog; when `haveCatalog` is false, ResolveTool
// falls back to a permissive @spwn/* prefix heuristic.
func ResolveTool(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case RefLocal:
		if ref.Name == "" {
			return ResolveNotFound
		}
		localPath := filepath.Join(root, "spwn", "tools", ref.Name)
		if info, err := os.Stat(localPath); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case RefSpwnBuiltin:
		if ref.Name == "" {
			return ResolveNotFound
		}
		full := "@spwn/" + ref.Name
		if haveCatalog {
			if _, ok := builtin[full]; ok {
				return ResolveOK
			}
			return ResolveNotFound
		}
		// No catalog: accept any well-formed @spwn/<name>.
		return ResolveOK

	case RefRegistry:
		return ResolveRegistryUnsupported

	default:
		return ResolveNotFound
	}
}
