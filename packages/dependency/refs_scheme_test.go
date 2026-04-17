package dependency_test

import (
	"testing"

	"spwn.sh/packages/dependency"
)

// TestParse_SchemeFormSpwn: `spwn:<name>` parses to the canonical
// KindSpwnBuiltin shape.
func TestParse_SchemeFormSpwn(t *testing.T) {
	got := dependency.ParseRef("spwn:unix")
	if got.Kind != dependency.KindSpwnBuiltin {
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
	got := dependency.ParseRef("github:jterrazz/foo")
	if got.Kind != dependency.KindRegistry {
		t.Errorf("kind: want KindRegistry, got %v", got.Kind)
	}
	if got.Owner != "jterrazz" {
		t.Errorf("owner: want jterrazz, got %q", got.Owner)
	}
	if got.Name != "foo" {
		t.Errorf("name: want foo, got %q", got.Name)
	}
}

// TestParse_SchemeFormLocal: an explicit `local:<name>` scheme maps
// to KindLocal just like a bare name. Useful when authors want to
// be explicit in lists that mix local and external refs.
func TestParse_SchemeFormLocal(t *testing.T) {
	got := dependency.ParseRef("local:my-parser")
	if got.Kind != dependency.KindLocal {
		t.Errorf("kind: want KindLocal, got %v", got.Kind)
	}
	if got.Name != "my-parser" {
		t.Errorf("name: want my-parser, got %q", got.Name)
	}
}

// TestParse_UnknownSchemeFallsBackToLocal: a ref with an unknown
// scheme (`gitlab:…`, `foo:bar`) is treated as a local bare name
// so existing error paths fire rather than a silent reject.
func TestParse_UnknownSchemeFallsBackToLocal(t *testing.T) {
	got := dependency.ParseRef("gitlab:x/y")
	if got.Kind != dependency.KindLocal {
		t.Errorf("kind: want KindLocal (unknown scheme), got %v", got.Kind)
	}
	if got.Name != "gitlab:x/y" {
		t.Errorf("name: want raw string for unknown scheme, got %q", got.Name)
	}
}

// TestParse_LocalNameWithColon: a bare name that happens to contain
// a colon (unusual but legal on the filesystem) must NOT be parsed
// as a scheme.
func TestParse_LocalNameWithColon(t *testing.T) {
	got := dependency.ParseRef("my-tool:extra")
	if got.Kind != dependency.KindLocal {
		t.Errorf("kind: want KindLocal (not a recognised scheme), got %v", got.Kind)
	}
}

// TestParse_LegacyAtPrefixRejected: the removed `@owner/name` form
// now parses as malformed (empty Name in a KindRegistry ref) so the
// resolver can surface a clear error instead of silently treating it
// as a local name.
func TestParse_LegacyAtPrefixRejected(t *testing.T) {
	for _, in := range []string{"@spwn/unix", "@jterrazz/foo", "@"} {
		got := dependency.ParseRef(in)
		if got.Kind != dependency.KindRegistry {
			t.Errorf("ParseRef(%q) kind = %v, want KindRegistry (legacy rejected)", in, got.Kind)
		}
		if got.Name != "" {
			t.Errorf("ParseRef(%q) name = %q, want empty (malformed)", in, got.Name)
		}
	}
}

// TestCanonical_EmitsSchemeForm: Canonical is the display helper; it
// always emits the scheme form, with `@version` suffixes stripped.
func TestCanonical_EmitsSchemeForm(t *testing.T) {
	cases := map[string]string{
		"spwn:unix":           "spwn:unix",
		"github:jterrazz/foo": "github:jterrazz/foo",
		"my-parser":           "my-parser",
		"local:my-parser":     "my-parser",
		"spwn:unix@24.04":     "spwn:unix",
	}
	for in, want := range cases {
		if got := dependency.Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSplitVersion_SchemeForm: `spwn:unix@24.04` splits cleanly off
// the final `@`.
func TestSplitVersion_SchemeForm(t *testing.T) {
	name, version := dependency.SplitVersion("spwn:unix@24.04")
	if name != "spwn:unix" || version != "24.04" {
		t.Errorf("split: got (%q, %q), want (spwn:unix, 24.04)", name, version)
	}
	name, version = dependency.SplitVersion("spwn:unix")
	if name != "spwn:unix" || version != "" {
		t.Errorf("split no-version: got (%q, %q), want (spwn:unix, )", name, version)
	}
}
