package dependency_test

import (
	"testing"

	"spwn.sh/packages/dependency"
)

// TestParse_SchemeFormSpwn: the canonical `spwn:<name>` scheme
// parses to the same Ref shape as the legacy `@spwn/<name>`.
func TestParse_SchemeFormSpwn(t *testing.T) {
	a := dependency.ParseRef("spwn:unix")
	b := dependency.ParseRef("@spwn/unix")
	for _, got := range []dependency.Ref{a, b} {
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
}

// TestParse_SchemeFormGitHub: `github:owner/repo` maps to the same
// registry shape as the legacy `@owner/repo`, so future resolver
// code can handle both through a single code path.
func TestParse_SchemeFormGitHub(t *testing.T) {
	a := dependency.ParseRef("github:jterrazz/foo")
	b := dependency.ParseRef("@jterrazz/foo")
	for _, got := range []dependency.Ref{a, b} {
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
// so existing error paths fire rather than a silent reject. Keeps
// us open to future schemes without having to ship a parser change
// for each one.
func TestParse_UnknownSchemeFallsBackToLocal(t *testing.T) {
	got := dependency.ParseRef("gitlab:x/y")
	if got.Kind != dependency.KindLocal {
		t.Errorf("kind: want KindLocal (unknown scheme), got %v", got.Kind)
	}
	// Name is the whole string — the resolver will fail to find a
	// `gitlab:x/y` directory under spwn/tools/, which is the right
	// user-visible error.
	if got.Name != "gitlab:x/y" {
		t.Errorf("name: want raw string for unknown scheme, got %q", got.Name)
	}
}

// TestParse_LocalNameWithColon: a bare name that happens to contain
// a colon (unusual but legal on the filesystem) must NOT be parsed
// as a scheme — the scheme allow-list (spwn/github/local) protects
// the bare path. Guards against surprises if a user ever authored
// `./spwn/tools/windows:c/`.
func TestParse_LocalNameWithColon(t *testing.T) {
	got := dependency.ParseRef("my-tool:extra")
	if got.Kind != dependency.KindLocal {
		t.Errorf("kind: want KindLocal (not a recognised scheme), got %v", got.Kind)
	}
}

// TestCanonical_EmitsSchemeForm: Canonical is the display helper;
// it must always emit the scheme form so scaffold output / CLI
// messages / docs are consistent regardless of which syntax the
// user wrote.
func TestCanonical_EmitsSchemeForm(t *testing.T) {
	cases := map[string]string{
		"@spwn/unix":          "spwn:unix",
		"spwn:unix":           "spwn:unix",
		"@jterrazz/foo":       "github:jterrazz/foo",
		"github:jterrazz/foo": "github:jterrazz/foo",
		"my-parser":           "my-parser",
		"local:my-parser":     "my-parser",
		"@spwn/unix@24.04":    "spwn:unix",
		"spwn:unix@24.04":     "spwn:unix",
	}
	for in, want := range cases {
		if got := dependency.Canonical(in); got != want {
			t.Errorf("Canonical(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestRegistryKey_ReturnsLegacyForm: the registry still keys
// tools under `@spwn/<name>`; this helper normalises any input
// syntax to that form so lookups work whether the user wrote
// `spwn:unix` or `@spwn/unix` in their agent.yaml.
func TestRegistryKey_ReturnsLegacyForm(t *testing.T) {
	cases := map[string]string{
		"@spwn/unix":          "@spwn/unix",
		"spwn:unix":           "@spwn/unix",
		"github:jterrazz/foo": "@jterrazz/foo",
		"@jterrazz/foo":       "@jterrazz/foo",
		"my-parser":           "my-parser",
	}
	for in, want := range cases {
		if got := dependency.RegistryKey(in); got != want {
			t.Errorf("RegistryKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestSplitVersion_SchemeForm: the scheme form carries versions the
// same way the legacy form does — `@24.04` suffix split off.
func TestSplitVersion_SchemeForm(t *testing.T) {
	name, version := dependency.SplitVersion("spwn:unix@24.04")
	if name != "spwn:unix" || version != "24.04" {
		t.Errorf("split: got (%q, %q), want (spwn:unix, 24.04)", name, version)
	}
	// Ref without version is unchanged.
	name, version = dependency.SplitVersion("spwn:unix")
	if name != "spwn:unix" || version != "" {
		t.Errorf("split no-version: got (%q, %q), want (spwn:unix, )", name, version)
	}
}
