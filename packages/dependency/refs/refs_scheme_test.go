package refs_test

import (
	"testing"

	"spwn.sh/packages/dependency/refs"
)

// TestParse_SchemeFormSpwn: `spwn:<name>` parses to the canonical
// KindSpwnBuiltin shape.
func TestParse_SchemeFormSpwn(t *testing.T) {
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

// TestParse_SchemeFormGitHub: `github:owner/repo` parses as a registry
// ref with the owner/name split out.
func TestParse_SchemeFormGitHub(t *testing.T) {
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

// TestParse_SchemeFormSkill: skill:<name> parses as a local skill ref.
func TestParse_SchemeFormSkill(t *testing.T) {
	got := refs.ParseRef("skill:code-review")
	if got.Kind != refs.KindLocalSkill {
		t.Errorf("kind: want KindLocalSkill, got %v", got.Kind)
	}
	if got.Name != "code-review" {
		t.Errorf("name: want code-review, got %q", got.Name)
	}
}

// TestParse_SchemeFormTool: tool:<name> parses as a local tool ref.
func TestParse_SchemeFormTool(t *testing.T) {
	got := refs.ParseRef("tool:ffmpeg")
	if got.Kind != refs.KindLocalTool {
		t.Errorf("kind: want KindLocalTool, got %v", got.Kind)
	}
	if got.Name != "ffmpeg" {
		t.Errorf("name: want ffmpeg, got %q", got.Name)
	}
}

// TestParse_SchemeFormHook: hook:<name> parses as a local hook ref.
func TestParse_SchemeFormHook(t *testing.T) {
	got := refs.ParseRef("hook:pre-spawn")
	if got.Kind != refs.KindLocalHook {
		t.Errorf("kind: want KindLocalHook, got %v", got.Kind)
	}
	if got.Name != "pre-spawn" {
		t.Errorf("name: want pre-spawn, got %q", got.Name)
	}
}

// TestParse_RetiredLocalScheme: the old `local:<name>` alias is gone;
// it now parses as KindInvalid so callers point the user at skill:,
// tool:, or hook: instead.
func TestParse_RetiredLocalScheme(t *testing.T) {
	got := refs.ParseRef("local:my-parser")
	if got.Kind != refs.KindInvalid {
		t.Errorf("local: alias should be invalid, got kind=%v", got.Kind)
	}
}

// TestParse_UnknownSchemeInvalid: a ref with an unknown scheme
// (`gitlab:…`, `foo:bar`) is invalid under the new grammar.
func TestParse_UnknownSchemeInvalid(t *testing.T) {
	got := refs.ParseRef("gitlab:x/y")
	if got.Kind != refs.KindInvalid {
		t.Errorf("kind: want KindInvalid (unknown scheme), got %v", got.Kind)
	}
}

// TestParse_LocalNameWithColon: a bare name that happens to contain
// a colon but doesn't match a recognised scheme is invalid.
func TestParse_LocalNameWithColon(t *testing.T) {
	got := refs.ParseRef("my-tool:extra")
	if got.Kind != refs.KindInvalid {
		t.Errorf("kind: want KindInvalid (not a recognised scheme), got %v", got.Kind)
	}
}

// TestParse_LegacyAtPrefixRejected: the removed `@owner/name` form
// now parses as KindInvalid so the resolver can surface a clear error.
func TestParse_LegacyAtPrefixRejected(t *testing.T) {
	for _, in := range []string{"@spwn/unix", "@jterrazz/foo", "@"} {
		got := refs.ParseRef(in)
		if got.Kind != refs.KindInvalid {
			t.Errorf("ParseRef(%q) kind = %v, want KindInvalid", in, got.Kind)
		}
	}
}

// TestCanonical_EmitsSchemeForm: Canonical is the display helper; it
// always emits the scheme form, with `@version` suffixes stripped.
func TestCanonical_EmitsSchemeForm(t *testing.T) {
	cases := map[string]string{
		"spwn:unix":           "spwn:unix",
		"github:jterrazz/foo": "github:jterrazz/foo",
		"skill:code-review":   "skill:code-review",
		"tool:my-parser":      "tool:my-parser",
		"hook:pre-spawn":      "hook:pre-spawn",
		"spwn:unix@24.04":     "spwn:unix",
		// Invalid inputs fall through unchanged for display.
		"bare-name":       "bare-name",
		"local:my-parser": "local:my-parser",
	}
	for in, want := range cases {
		if got := refs.Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSplitVersion_SchemeForm: `spwn:unix@24.04` splits cleanly off
// the final `@`.
func TestSplitVersion_SchemeForm(t *testing.T) {
	name, version := refs.SplitVersion("spwn:unix@24.04")
	if name != "spwn:unix" || version != "24.04" {
		t.Errorf("split: got (%q, %q), want (spwn:unix, 24.04)", name, version)
	}
	name, version = refs.SplitVersion("spwn:unix")
	if name != "spwn:unix" || version != "" {
		t.Errorf("split no-version: got (%q, %q), want (spwn:unix, )", name, version)
	}
}
