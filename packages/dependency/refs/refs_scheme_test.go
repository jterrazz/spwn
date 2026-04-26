package refs_test

import (
	"testing"

	"spwn.sh/packages/dependency/refs"
)

// TestParse_SourceFormSpwn: `spwn:<name>` parses to the canonical
// KindSpwnBuiltin shape.
func TestParse_SourceFormSpwn(t *testing.T) {
	got := refs.ParseRef("spwn:unix")
	if got.Kind != refs.KindSpwnBuiltin {
		t.Errorf("kind: want KindSpwnBuiltin, got %v (raw=%q)", got.Kind, got.Raw)
	}
	if got.Owner != "spwn" {
		t.Errorf("owner: want spwn, got %q (raw=%q)", got.Owner, got.Raw)
	}
	if got.Name != "unix" {
		t.Errorf("name: want unix, got %q (raw=%q)", got.Name, got.Raw)
	}
}

// TestParse_SourceFormGitHub: `github:owner/repo` parses as a registry
// ref with the owner/name split out.
func TestParse_SourceFormGitHub(t *testing.T) {
	got := refs.ParseRef("github:jterrazz/foo")
	if got.Kind != refs.KindRegistry {
		t.Errorf("kind: want KindRegistry, got %v", got.Kind)
	}
	if got.Owner != "jterrazz" {
		t.Errorf("owner: want jterrazz, got %q", got.Owner)
	}
	if got.Name != "foo" {
		t.Errorf("name: want foo, got %q", got.Name)
	}
}

// TestParse_PathFormSkill: skill/<name> parses as a local skill ref.
func TestParse_PathFormSkill(t *testing.T) {
	got := refs.ParseRef("skill/code-review")
	if got.Kind != refs.KindLocalSkill {
		t.Errorf("kind: want KindLocalSkill, got %v", got.Kind)
	}
	if got.Name != "code-review" {
		t.Errorf("name: want code-review, got %q", got.Name)
	}
}

// TestParse_PathFormTool: tool/<name> parses as a local tool ref.
func TestParse_PathFormTool(t *testing.T) {
	got := refs.ParseRef("tool/ffmpeg")
	if got.Kind != refs.KindLocalTool {
		t.Errorf("kind: want KindLocalTool, got %v", got.Kind)
	}
	if got.Name != "ffmpeg" {
		t.Errorf("name: want ffmpeg, got %q", got.Name)
	}
}

// TestParse_PathFormHook: hook/<name> parses as a local hook ref.
func TestParse_PathFormHook(t *testing.T) {
	got := refs.ParseRef("hook/pre-spawn")
	if got.Kind != refs.KindLocalHook {
		t.Errorf("kind: want KindLocalHook, got %v", got.Kind)
	}
	if got.Name != "pre-spawn" {
		t.Errorf("name: want pre-spawn, got %q", got.Name)
	}
}

// TestParse_RetiredColonForms: the colon-form local schemes
// (`skill:`, `tool:`, `hook:`, `local:`) are gone; they all parse as
// KindInvalid so callers can point the user at the new path-style
// forms.
func TestParse_RetiredColonForms(t *testing.T) {
	for _, in := range []string{"skill:focus", "tool:my-parser", "hook:pre-spawn", "local:my-parser"} {
		got := refs.ParseRef(in)
		if got.Kind != refs.KindInvalid {
			t.Errorf("ParseRef(%q) kind = %v, want KindInvalid", in, got.Kind)
		}
	}
}

// TestParse_PathFormUnknownTypeInvalid: a path-form ref with an
// unknown leading segment is invalid.
func TestParse_PathFormUnknownTypeInvalid(t *testing.T) {
	for _, in := range []string{"thing/foo", "agent/neo", "world/default"} {
		got := refs.ParseRef(in)
		if got.Kind != refs.KindInvalid {
			t.Errorf("ParseRef(%q) kind = %v, want KindInvalid", in, got.Kind)
		}
	}
}

// TestParse_PathFormDeepPathInvalid: only one slash is allowed in
// path-form local refs. `tool/foo/bar` is rejected — names with
// embedded slashes aren't supported, and accepting them now would
// box future grammars (e.g. `github:owner/repo/tool/foo`) into a
// reinterpretation.
func TestParse_PathFormDeepPathInvalid(t *testing.T) {
	for _, in := range []string{"tool/foo/bar", "skill/a/b", "hook/x/y/z"} {
		got := refs.ParseRef(in)
		if got.Kind != refs.KindInvalid {
			t.Errorf("ParseRef(%q) kind = %v, want KindInvalid (deep paths not allowed)", in, got.Kind)
		}
	}
}

// TestParse_UnknownSchemeInvalid: a ref with an unknown source
// scheme (`gitlab:…`, `foo:bar`) is invalid.
func TestParse_UnknownSchemeInvalid(t *testing.T) {
	got := refs.ParseRef("gitlab:x/y")
	if got.Kind != refs.KindInvalid {
		t.Errorf("kind: want KindInvalid (unknown scheme), got %v", got.Kind)
	}
}

// TestParse_LocalNameWithColon: a bare name that happens to contain
// a colon but doesn't match a recognised source scheme is invalid.
func TestParse_LocalNameWithColon(t *testing.T) {
	got := refs.ParseRef("my-tool:extra")
	if got.Kind != refs.KindInvalid {
		t.Errorf("kind: want KindInvalid (not a recognised scheme), got %v", got.Kind)
	}
}

// TestParse_LegacyAtPrefixRejected: the removed `@owner/name` form
// parses as KindInvalid so the resolver can surface a clear error.
func TestParse_LegacyAtPrefixRejected(t *testing.T) {
	for _, in := range []string{"@spwn/unix", "@jterrazz/foo", "@"} {
		got := refs.ParseRef(in)
		if got.Kind != refs.KindInvalid {
			t.Errorf("ParseRef(%q) kind = %v, want KindInvalid", in, got.Kind)
		}
	}
}

// TestCanonical_EmitsCanonicalForm: Canonical is the display helper;
// it always emits the canonical form, with `@version` suffixes
// stripped. Local refs canonicalise to the path form; spwn/github
// keep their colon source prefix.
func TestCanonical_EmitsCanonicalForm(t *testing.T) {
	cases := map[string]string{
		"spwn:unix":           "spwn:unix",
		"github:jterrazz/foo": "github:jterrazz/foo",
		"skill/code-review":   "skill/code-review",
		"tool/my-parser":      "tool/my-parser",
		"hook/pre-spawn":      "hook/pre-spawn",
		"spwn:unix@24.04":     "spwn:unix",
		// Invalid inputs (including retired colon-forms) fall through
		// unchanged for display.
		"bare-name":       "bare-name",
		"local:my-parser": "local:my-parser",
		"skill:focus":     "skill:focus",
		"tool:ffmpeg":     "tool:ffmpeg",
	}
	for in, want := range cases {
		if got := refs.Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSplitVersion_SourceForm: `spwn:unix@24.04` splits cleanly off
// the final `@`.
func TestSplitVersion_SourceForm(t *testing.T) {
	name, version := refs.SplitVersion("spwn:unix@24.04")
	if name != "spwn:unix" || version != "24.04" {
		t.Errorf("split: got (%q, %q), want (spwn:unix, 24.04)", name, version)
	}
	name, version = refs.SplitVersion("spwn:unix")
	if name != "spwn:unix" || version != "" {
		t.Errorf("split no-version: got (%q, %q), want (spwn:unix, )", name, version)
	}
}
