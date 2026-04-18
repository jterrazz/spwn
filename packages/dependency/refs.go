// Package dependency parses and resolves dependency references in
// agent.yaml and spwn.yaml.
//
// Every ref is `scheme:target`. There is no bare-name form. Five schemes
// are recognised:
//
//	spwn:<name>                  built-in catalog entry (compiled into the binary)
//	github:<owner>/<repo>        remote dependency (planned; not yet resolved)
//	skill:<name>                 local, resolves to ./spwn/skills/<name>.md
//	tool:<name>                  local, resolves to ./spwn/tools/<name>/
//	hook:<name>                  local, resolves to ./spwn/hooks/<name>.sh
//
// Anything else — bare names, unknown schemes, legacy `@owner/name`,
// the retired `local:<name>` alias — parses to KindInvalid so callers
// can surface a clear error pointing at the new grammar.
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
	// KindInvalid is a ref that didn't match any recognised scheme.
	// Callers should reject it with a hint pointing at skill:, tool:,
	// hook:, spwn:, or github:.
	KindInvalid RefKind = iota
	// KindSpwnBuiltin is a spwn:<name> dependency compiled into the
	// binary.
	KindSpwnBuiltin
	// KindRegistry is github:<owner>/<repo> — reserved for a future
	// remote resolver, not yet implemented.
	KindRegistry
	// KindLocalSkill is skill:<name>, resolving to spwn/skills/<name>.md.
	KindLocalSkill
	// KindLocalTool is tool:<name>, resolving to spwn/tools/<name>/.
	KindLocalTool
	// KindLocalHook is hook:<name>, resolving to spwn/hooks/<name>.sh.
	KindLocalHook
)

// Ref is a parsed, classified reference.
type Ref struct {
	Raw   string
	Kind  RefKind
	Owner string // "spwn" for builtin, owner for registry, "" otherwise
	Name  string // dependency name (or repo for registry refs)
}

// ParseRef classifies a ref string. Trims whitespace. Does NOT strip a
// `@version` suffix — call SplitVersion first if the caller accepts
// versioned refs.
//
// Rules:
//   - "spwn:<name>": KindSpwnBuiltin, Owner="spwn", Name=<name>.
//   - "github:<owner>/<repo>": KindRegistry, Owner=<owner>, Name=<repo>.
//   - "skill:<name>": KindLocalSkill, Name=<name>.
//   - "tool:<name>": KindLocalTool, Name=<name>.
//   - "hook:<name>": KindLocalHook, Name=<name>.
//   - Anything else — bare names, unknown schemes, `@`-prefixed, the
//     retired `local:<name>` alias — parses to KindInvalid.
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
		case "skill":
			return Ref{Raw: raw, Kind: KindLocalSkill, Name: target}
		case "tool":
			return Ref{Raw: raw, Kind: KindLocalTool, Name: target}
		case "hook":
			return Ref{Raw: raw, Kind: KindLocalHook, Name: target}
		}
	}

	// Everything else (bare names, unknown schemes, `@`-prefixed refs,
	// the retired `local:<name>` alias) is invalid under the new
	// grammar.
	return Ref{Raw: raw, Kind: KindInvalid}
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
	case "spwn", "github", "skill", "tool", "hook":
		return scheme, s[colon+1:], true
	}
	return "", "", false
}

// SplitVersion separates a ref from its optional `@version` suffix.
// `spwn:unix@24.04` returns ("spwn:unix", "24.04"). `skill:focus`
// returns ("skill:focus", ""). The version is whatever follows the
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
	case KindLocalSkill:
		if r.Name == "" {
			return name
		}
		return "skill:" + r.Name
	case KindLocalTool:
		if r.Name == "" {
			return name
		}
		return "tool:" + r.Name
	case KindLocalHook:
		if r.Name == "" {
			return name
		}
		return "hook:" + r.Name
	}
	return name
}

// ResolveResult is the tri-state outcome of resolving a Ref.
type ResolveResult int

const (
	// ResolveOK means the ref points to something real.
	ResolveOK ResolveResult = iota
	// ResolveNotFound means the ref looks valid but the target is
	// missing (typo, unknown builtin, scheme-form ref whose file is
	// absent).
	ResolveNotFound
	// ResolveRegistryUnsupported means the ref is a github:<owner>/<repo>
	// registry ref — reserved for a future remote resolver, not yet
	// implemented.
	ResolveRegistryUnsupported
	// ResolveInvalid means the ref didn't match any recognised scheme
	// (bare name, unknown scheme, legacy syntax). Callers surface a
	// helpful error pointing at the five valid schemes.
	ResolveInvalid
)

// ResolveTool answers whether a Ref resolves to a real dependency.
//
//   - KindLocalTool: checks that <root>/spwn/tools/<name>/ is a directory.
//   - KindLocalSkill: checks that <root>/spwn/skills/<name>.md is a file.
//   - KindLocalHook: checks that <root>/spwn/hooks/<name>.sh is a file.
//   - KindSpwnBuiltin: checks that spwn:<name> is in `builtin` when
//     `haveCatalog` is true, else accepts any well-formed ref.
//   - KindRegistry: always returns ResolveRegistryUnsupported.
//   - KindInvalid: returns ResolveInvalid so the caller can surface a
//     helpful error pointing at the valid schemes.
func ResolveTool(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case KindLocalTool:
		if ref.Name == "" {
			return ResolveNotFound
		}
		localPath := filepath.Join(root, "spwn", "tools", ref.Name)
		if info, err := os.Stat(localPath); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindLocalSkill:
		if ref.Name == "" {
			return ResolveNotFound
		}
		filePath := filepath.Join(root, "spwn", "skills", ref.Name+".md")
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindLocalHook:
		if ref.Name == "" {
			return ResolveNotFound
		}
		filePath := filepath.Join(root, "spwn", "hooks", ref.Name+".sh")
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
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

	case KindInvalid:
		return ResolveInvalid

	default:
		return ResolveNotFound
	}
}

// ResolveSkill answers whether a Ref resolves to a real skill. Only
// skill:<name> resolves against spwn/skills/<name>.md; tool: and hook:
// are not skills. spwn:/github: keep their catalog/remote semantics.
func ResolveSkill(root string, ref Ref, builtin map[string]struct{}, haveCatalog bool) ResolveResult {
	switch ref.Kind {
	case KindLocalSkill:
		if ref.Name == "" {
			return ResolveNotFound
		}
		filePath := filepath.Join(root, "spwn", "skills", ref.Name+".md")
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindLocalTool:
		if ref.Name == "" {
			return ResolveNotFound
		}
		dirPath := filepath.Join(root, "spwn", "tools", ref.Name)
		if info, err := os.Stat(dirPath); err == nil && info.IsDir() {
			return ResolveOK
		}
		return ResolveNotFound

	case KindLocalHook:
		if ref.Name == "" {
			return ResolveNotFound
		}
		filePath := filepath.Join(root, "spwn", "hooks", ref.Name+".sh")
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
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

	case KindInvalid:
		return ResolveInvalid

	default:
		return ResolveNotFound
	}
}

// IsLocalKind reports whether the RefKind refers to an on-disk local
// block (skill, tool, or hook) — i.e. not a spwn: catalog ref, not a
// github: registry ref, and not KindInvalid.
func IsLocalKind(k RefKind) bool {
	switch k {
	case KindLocalSkill, KindLocalTool, KindLocalHook:
		return true
	}
	return false
}
