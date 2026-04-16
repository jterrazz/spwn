// Package refs parses and resolves pack references in agent.yaml
// and spwn.yaml.
//
// Spwn projects reference plugins in three ways:
//
//  1. Local — a bare name, resolved against ./spwn/packs/<name>/
//     (directory form, with its own plugin.yaml) or
//     ./spwn/packs/<name>.md (bare-markdown skill form).
//  2. @spwn/<name> — a built-in pack compiled into the spwn
//     binary, looked up against the catalog provided by the caller.
//  3. @<owner>/<name> — a remote registry plugin. Reserved for a
//     future community registry; resolved today as "unsupported" so
//     users aren't told their ref is a typo.
//
// ParseRef classifies a ref string. ResolveTool and ResolveSkill
// answer whether the target exists. Versioned refs like
// `@spwn/unix@24.04` are handled via SplitVersion — strip the version
// first, then call ParseRef on the pack part.
//
// This package has no dependencies on the validator or compiler so
// it can be imported from CLI commands and the build pipeline alike.
package refs

import (
	"os"
	"path/filepath"
	"strings"
)

// Kind classifies a reference.
type Kind int

const (
	// KindLocal is a bare name authored in ./spwn/packs/<name>/
	// (directory form) or ./spwn/packs/<name>.md (bare-markdown
	// skill form).
	KindLocal Kind = iota
	// KindSpwnBuiltin is a @spwn/<name> plugin compiled into the binary.
	KindSpwnBuiltin
	// KindRegistry is @<owner>/<name> with owner != "spwn" — reserved
	// for a future community registry, not yet supported.
	KindRegistry
)

// Ref is a parsed, classified reference.
type Ref struct {
	Raw   string
	Kind  Kind
	Owner string // "" for local, "spwn" for builtin, user/org for registry
	Name  string // pack name without scope
}

// Parse classifies a ref string. Trims whitespace. Does NOT strip a
// `@version` suffix — call SplitVersion first if the caller accepts
// versioned refs.
//
// Rules:
//   - No leading "@": KindLocal, Owner="", Name=trimmed.
//   - "@spwn/<name>": KindSpwnBuiltin, Owner="spwn", Name=<name>.
//   - "@<owner>/<name>" (owner != spwn): KindRegistry.
//   - Malformed "@" or "@<owner>" without a slash: KindRegistry with
//     empty Name — callers should reject it as malformed.
func Parse(s string) Ref {
	raw := s
	s = strings.TrimSpace(s)

	if !strings.HasPrefix(s, "@") {
		return Ref{Raw: raw, Kind: KindLocal, Name: s}
	}

	rest := s[1:]
	slash := strings.Index(rest, "/")
	if slash < 0 {
		return Ref{Raw: raw, Kind: KindRegistry, Owner: rest, Name: ""}
	}
	owner := rest[:slash]
	name := rest[slash+1:]
	if owner == "spwn" {
		return Ref{Raw: raw, Kind: KindSpwnBuiltin, Owner: "spwn", Name: name}
	}
	return Ref{Raw: raw, Kind: KindRegistry, Owner: owner, Name: name}
}

// SplitVersion separates a ref from its optional `@version` suffix.
// For `@spwn/unix@24.04` returns ("@spwn/unix", "24.04"). For a bare
// `local-tool` returns ("local-tool", ""). The version is whatever
// follows the last `@` that isn't at position zero.
func SplitVersion(ref string) (pack, version string) {
	if !strings.HasPrefix(ref, "@") {
		idx := strings.LastIndex(ref, "@")
		if idx > 0 {
			return ref[:idx], ref[idx+1:]
		}
		return ref, ""
	}
	rest := ref[1:]
	if idx := strings.LastIndex(rest, "@"); idx >= 0 {
		return "@" + rest[:idx], rest[idx+1:]
	}
	return ref, ""
}

// Canonical returns the scope-and-name form of a ref with any
// `@version` suffix stripped. Useful as a map key when deduping
// refs across agents.
func Canonical(ref string) string {
	pack, _ := SplitVersion(ref)
	return pack
}

// ResolveResult is the tri-state outcome of resolving a Ref.
type ResolveResult int

const (
	// ResolveOK means the ref points to something real.
	ResolveOK ResolveResult = iota
	// ResolveNotFound means the ref looks valid but the target is
	// missing (typo, unknown builtin, bare name with no local dir).
	ResolveNotFound
	// ResolveRegistryUnsupported means the ref is a @<owner>/<name>
	// registry ref other than @spwn — reserved for a future community
	// registry, not yet implemented.
	ResolveRegistryUnsupported
)

// ResolveTool answers whether a Ref resolves to a real package
// (directory form — for skill-only markdown refs use ResolveSkill).
//
//   - KindLocal: checks that <root>/spwn/packs/<name>/ is a directory.
//   - KindSpwnBuiltin: checks that @spwn/<name> is in `builtin` when
//     `haveCatalog` is true, else accepts any well-formed ref.
//   - KindRegistry: always returns ResolveRegistryUnsupported.
func ResolveTool(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case KindLocal:
		if ref.Name == "" {
			return ResolveNotFound
		}
		localPath := filepath.Join(root, "spwn", "plugins", ref.Name)
		if info, err := os.Stat(localPath); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindSpwnBuiltin:
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
		return ResolveOK

	case KindRegistry:
		return ResolveRegistryUnsupported

	default:
		return ResolveNotFound
	}
}

// ResolveSkill answers whether a Ref resolves to a real skill
// package — either the bare-markdown form or a full directory
// package (same layout as a tool).
//
//   - KindLocal: checks that <root>/spwn/packs/<name>.md exists
//     as a file (bare-markdown skill), or that
//     <root>/spwn/packs/<name>/ exists as a directory (full
//     package that may or may not ship skills via its own plugin.yaml).
//   - KindSpwnBuiltin: checks that @spwn/<name> is in `builtin` when
//     `haveCatalog` is true, else accepts any well-formed ref.
//   - KindRegistry: always returns ResolveRegistryUnsupported.
func ResolveSkill(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case KindLocal:
		if ref.Name == "" {
			return ResolveNotFound
		}
		fileForm := filepath.Join(root, "spwn", "plugins", ref.Name+".md")
		if info, err := os.Stat(fileForm); err == nil && !info.IsDir() {
			return ResolveOK
		}
		dirForm := filepath.Join(root, "spwn", "plugins", ref.Name)
		if info, err := os.Stat(dirForm); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindSpwnBuiltin:
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
		return ResolveOK

	case KindRegistry:
		return ResolveRegistryUnsupported

	default:
		return ResolveNotFound
	}
}
