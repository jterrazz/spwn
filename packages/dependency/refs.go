// Package dependency parses and resolves dependency references in
// agent.yaml and spwn.yaml.
//
// Spwn accepts two equivalent ref syntaxes:
//
//	spwn:unix                     (canonical)
//	@spwn/unix                    (legacy, still accepted)
//	github:jterrazz/foo           (canonical remote; not yet resolved)
//	@jterrazz/foo                 (legacy "registry" — not yet resolved)
//
// The scheme form scales cleanly to future sources (`git+ssh:`,
// `https:`, `local:`) while the `@owner/name` form is an npm-ism
// that doesn't. Both point at the same Ref; Canonical emits the
// scheme form for display.
//
// Spwn projects reference dependencies in three ways:
//
//  1. Local — a bare name, resolved against ./spwn/tools/<name>/
//     (directory form, with its own tool.yaml) or
//     ./spwn/skills/<name>.md (bare-markdown skill form).
//  2. spwn:<name> (or legacy @spwn/<name>) — a built-in dependency
//     compiled into the spwn binary.
//  3. github:<owner>/<repo> (or legacy @<owner>/<repo>) — a remote
//     dependency reserved for a future registry; resolved today as
//     "unsupported" so users aren't told their ref is a typo.
//
// ParseRef classifies a ref string. ResolveTool and ResolveSkill
// answer whether the target exists. Versioned refs like
// `spwn:unix@24.04` are handled via SplitVersion — strip the
// version first, then call ParseRef on the name part.
//
// This package has no dependencies on the validator or compiler so
// it can be imported from CLI commands and the build pipeline alike.
package dependency

import (
	"os"
	"path/filepath"
	"strings"
)

// Kind classifies a reference.
type RefKind int

const (
	// KindLocal is a bare name authored in ./spwn/tools/<name>/
	// (directory form) or ./spwn/skills/<name>.md (bare-markdown
	// skill form).
	KindLocal RefKind = iota
	// KindSpwnBuiltin is a @spwn/<name> dependency compiled into
	// the binary.
	KindSpwnBuiltin
	// KindRegistry is @<owner>/<name> with owner != "spwn" — reserved
	// for a future community registry, not yet supported.
	KindRegistry
)

// Ref is a parsed, classified reference.
type Ref struct {
	Raw   string
	Kind  RefKind
	Owner string // "" for local, "spwn" for builtin, user/org for registry
	Name  string // dependency name without scope
}

// Parse classifies a ref string. Trims whitespace. Does NOT strip a
// `@version` suffix — call SplitVersion first if the caller accepts
// versioned refs.
//
// Rules (canonical scheme form listed first; legacy form in parens):
//   - "spwn:<name>" or "@spwn/<name>": KindSpwnBuiltin,
//     Owner="spwn", Name=<name>.
//   - "github:<owner>/<repo>" or "@<owner>/<name>" (owner != spwn):
//     KindRegistry — reserved for future remote resolution; today
//     callers surface these as "not yet supported".
//   - Anything else without a scheme prefix: KindLocal, Owner="",
//     Name=trimmed.
//   - Malformed refs (empty scheme target, `@` without slash, …):
//     KindRegistry with empty Name so callers reject as malformed.
func ParseRef(s string) Ref {
	raw := s
	s = strings.TrimSpace(s)

	// Scheme form: <scheme>:<target>.
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
		// Unknown scheme — treat as a local bare name so existing
		// behaviour (unknown refs surface via ResolveTool) still
		// fires rather than silently skipping the check.
		return Ref{Raw: raw, Kind: KindLocal, Name: s}
	}

	// Legacy @owner/name form.
	if strings.HasPrefix(s, "@") {
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

	// Bare name: local.
	return Ref{Raw: raw, Kind: KindLocal, Name: s}
}

// splitScheme peels off a leading `<scheme>:<target>` when the
// scheme is a known recognised one. Returns ok=false when the
// string has no colon, when the colon is the first character
// (legacy `:something`), or when the scheme isn't one spwn
// understands. Restricting to known schemes avoids mis-parsing
// things like `Windows:C:\Path` if a user ever wrote one.
func splitScheme(s string) (scheme, target string, ok bool) {
	colon := strings.IndexByte(s, ':')
	if colon <= 0 {
		return "", "", false
	}
	scheme = s[:colon]
	// Scheme must be a-z only — reject anything with a slash,
	// dot, dash, uppercase, or digits so user slugs like
	// `my-tool:thing` stay local.
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
// For `@spwn/unix@24.04` returns ("@spwn/unix", "24.04"). For a bare
// `local-tool` returns ("local-tool", ""). The version is whatever
// follows the last `@` that isn't at position zero.
//
// Scheme-form refs (`spwn:unix@24.04`) work the same way: the only
// `@` in the string is the version separator, so LastIndex finds it
// cleanly without the prefix peel-and-stitch the @owner/name form
// needs.
func SplitVersion(ref string) (name, version string) {
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

// Canonical returns the scheme-form display string for a ref, with
// any `@version` suffix stripped. `@spwn/unix` and `spwn:unix` both
// return `spwn:unix`; `@jterrazz/foo` returns `github:jterrazz/foo`.
// Malformed inputs fall through to the original string unchanged
// so callers can still display them in error messages.
//
// Use this for user-facing output (CLI messages, scaffolded files,
// lockfile entries going forward). Existing registry keys may still
// be on the legacy `@spwn/<name>` form — use RegistryKey for those
// lookups so a scheme-form input still resolves.
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

// RegistryKey returns the string form used as a key in the tool
// registry, normalising either ref syntax to the legacy form. The
// image.Registry stores tools by their Tool.Name() — today that's
// `@spwn/unix`, not `spwn:unix` — so lookups keyed on user input
// must canonicalise first.
//
// This lives as a separate helper from Canonical so the migration
// to the scheme form can proceed in stages: user-facing output can
// switch today without rewriting every registry Register/Get call
// site.
func RegistryKey(ref string) string {
	name, _ := SplitVersion(ref)
	r := ParseRef(name)
	switch r.Kind {
	case KindSpwnBuiltin:
		if r.Name == "" {
			return name
		}
		return "@spwn/" + r.Name
	case KindRegistry:
		if r.Owner == "" || r.Name == "" {
			return name
		}
		return "@" + r.Owner + "/" + r.Name
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
	// ResolveRegistryUnsupported means the ref is a @<owner>/<name>
	// registry ref other than @spwn — reserved for a future community
	// registry, not yet implemented.
	ResolveRegistryUnsupported
)

// ResolveTool answers whether a Ref resolves to a real dependency
// (directory form — for skill-only markdown refs use ResolveSkill).
//
//   - KindLocal: checks that <root>/spwn/tools/<name>/ is a directory.
//   - KindSpwnBuiltin: checks that @spwn/<name> is in `builtin` when
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
// — either the bare-markdown form (spwn/skills/<name>.md) or a
// full directory-form dependency (spwn/tools/<name>/ which may
// ship skills).
//
//   - KindLocal: checks spwn/skills/<name>.md (bare markdown) first,
//     then spwn/tools/<name>/ (directory-form dependency).
//   - KindSpwnBuiltin: checks that @spwn/<name> is in `builtin` when
//     `haveCatalog` is true, else accepts any well-formed ref.
//   - KindRegistry: always returns ResolveRegistryUnsupported.
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
