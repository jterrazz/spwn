// Package dependency parses and resolves dependency references in
// agent.yaml and spwn.yaml.
//
// Spwn has three ref forms:
//
//	spwn:unix              built-in dependency compiled into the binary
//	github:owner/repo      remote dependency (planned; not yet resolved)
//	bare-name              local, resolved against ./spwn/tools/<name>/
//	                       (directory form) or ./spwn/skills/<name>.md
//	                       (bare-markdown skill form)
//
// Versioned refs like `spwn:unix@24.04` are handled via SplitVersion —
// strip the version first, then call ParseRef on the name part.
//
// This package has no dependencies on the validator or compiler so it
// can be imported from CLI commands and the build pipeline alike.
package dependency

import (
	"os"
	"path/filepath"
	"strings"
)

// RefKind classifies a reference.
type RefKind int

const (
	// KindLocal is a bare name authored in ./spwn/tools/<name>/
	// (directory form) or ./spwn/skills/<name>.md (bare-markdown
	// skill form).
	KindLocal RefKind = iota
	// KindSpwnBuiltin is a spwn:<name> dependency compiled into the
	// binary.
	KindSpwnBuiltin
	// KindRegistry is github:<owner>/<repo> — reserved for a future
	// remote resolver, not yet implemented.
	KindRegistry
)

// Ref is a parsed, classified reference.
type Ref struct {
	Raw   string
	Kind  RefKind
	Owner string // "" for local, "spwn" for builtin, owner for registry
	Name  string // dependency name (or repo for registry refs)
}

// ParseRef classifies a ref string. Trims whitespace. Does NOT strip a
// `@version` suffix — call SplitVersion first if the caller accepts
// versioned refs.
//
// Rules:
//   - "spwn:<name>": KindSpwnBuiltin, Owner="spwn", Name=<name>.
//   - "github:<owner>/<repo>": KindRegistry, Owner=<owner>, Name=<repo>.
//   - Anything else without a scheme prefix: KindLocal.
//   - Malformed (empty scheme target, unknown scheme, bare `@` prefix):
//     classified so callers can reject. Any ref starting with `@` is
//     treated as malformed — the legacy `@owner/name` syntax was
//     removed; use `spwn:name` or `github:owner/repo` instead.
func ParseRef(s string) Ref {
	raw := s
	s = strings.TrimSpace(s)

	if scheme, target, ok := splitScheme(s); ok {
		switch scheme {
		case "spwn":
			return Ref{Raw: raw, Kind: KindSpwnBuiltin, Owner: "spwn", Name: target}
		case "github":
			slash := strings.Index(target, "/")
			if slash < 0 {
				return Ref{Raw: raw, Kind: KindRegistry, Owner: target, Name: ""}
			}
			return Ref{Raw: raw, Kind: KindRegistry, Owner: target[:slash], Name: target[slash+1:]}
		case "local":
			return Ref{Raw: raw, Kind: KindLocal, Name: target}
		}
	}

	// Bare `@` prefix is reserved as malformed so the old `spwn:name`
	// syntax surfaces as a clear error rather than silently resolving
	// to a "local tool" lookup that will never match.
	if strings.HasPrefix(s, "@") {
		return Ref{Raw: raw, Kind: KindRegistry, Owner: "", Name: ""}
	}

	return Ref{Raw: raw, Kind: KindLocal, Name: s}
}

// splitScheme peels off a leading `<scheme>:<target>` when the scheme
// is one spwn recognises. Returns ok=false when the string has no
// colon, the colon is the first character, or the scheme is unknown.
// Restricting to known schemes avoids mis-parsing path-shaped strings.
func splitScheme(s string) (scheme, target string, ok bool) {
	colon := strings.IndexByte(s, ':')
	if colon <= 0 {
		return "", "", false
	}
	scheme = s[:colon]
	for _, c := range scheme {
		if c < 'a' || c > 'z' {
			return "", "", false
		}
	}
	switch scheme {
	case "spwn", "github", "local":
		return scheme, s[colon+1:], true
	}
	return "", "", false
}

// SplitVersion separates a ref from its optional `@version` suffix.
// `spwn:unix@24.04` returns ("spwn:unix", "24.04"). `local-tool`
// returns ("local-tool", ""). The version is whatever follows the
// last `@` in the string.
func SplitVersion(ref string) (name, version string) {
	idx := strings.LastIndex(ref, "@")
	if idx > 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, ""
}

// Canonical returns the canonical scheme-form display string for a
// ref, with any `@version` suffix stripped. Malformed inputs fall
// through to the original string unchanged so callers can still
// display them in error messages.
func Canonical(ref string) string {
	name, _ := SplitVersion(ref)
	r := ParseRef(name)
	switch r.Kind {
	case KindSpwnBuiltin:
		if r.Name == "" {
			return name
		}
		return "spwn:" + r.Name
	case KindRegistry:
		if r.Owner == "" || r.Name == "" {
			return name
		}
		return "github:" + r.Owner + "/" + r.Name
	case KindLocal:
		return r.Name
	}
	return name
}

// ResolveResult is the tri-state outcome of resolving a Ref.
type ResolveResult int

const (
	// ResolveOK means the ref points to something real.
	ResolveOK ResolveResult = iota
	// ResolveNotFound means the ref looks valid but the target is
	// missing (typo, unknown builtin, bare name with no local dir).
	ResolveNotFound
	// ResolveRegistryUnsupported means the ref is a github:<owner>/<repo>
	// registry ref — reserved for a future remote resolver, not yet
	// implemented.
	ResolveRegistryUnsupported
)

// ResolveTool answers whether a Ref resolves to a real dependency
// (directory form — for skill-only markdown refs use ResolveSkill).
//
//   - KindLocal: checks that <root>/spwn/tools/<name>/ is a directory.
//   - KindSpwnBuiltin: checks that spwn:<name> is in `builtin` when
//     `haveCatalog` is true, else accepts any well-formed ref.
//   - KindRegistry: always returns ResolveRegistryUnsupported.
func ResolveTool(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case KindLocal:
		if ref.Name == "" {
			return ResolveNotFound
		}
		localPath := filepath.Join(root, "spwn", "tools", ref.Name)
		if info, err := os.Stat(localPath); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindSpwnBuiltin:
		if ref.Name == "" {
			return ResolveNotFound
		}
		full := "spwn:" + ref.Name
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

// ResolveSkill answers whether a Ref resolves to a real skill — either
// the bare-markdown form (spwn/skills/<name>.md) or a full directory-
// form dependency (spwn/tools/<name>/ which may ship skills).
func ResolveSkill(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case KindLocal:
		if ref.Name == "" {
			return ResolveNotFound
		}
		fileForm := filepath.Join(root, "spwn", "skills", ref.Name+".md")
		if info, err := os.Stat(fileForm); err == nil && !info.IsDir() {
			return ResolveOK
		}
		dirForm := filepath.Join(root, "spwn", "tools", ref.Name)
		if info, err := os.Stat(dirForm); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindSpwnBuiltin:
		if ref.Name == "" {
			return ResolveNotFound
		}
		full := "spwn:" + ref.Name
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
