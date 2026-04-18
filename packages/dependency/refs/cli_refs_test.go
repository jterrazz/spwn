package refs_test

import (
	"strings"
	"testing"

	"spwn.sh/packages/dependency/refs"
)

func TestResolveCLI(t *testing.T) {
	// Catalog for tests: a fixed set so the hint-list assertion is
	// deterministic. Mixes tool entries (qmd, unix, python) so the
	// bare-name match path has real names to hit.
	catalog := []string{"qmd", "unix", "python", "paper-reading"}

	cases := []struct {
		name   string
		input  string
		want   string
		errSub string // substring expected in error, "" ⇒ must succeed
	}{
		// Happy path: bare catalog match.
		{"bare match", "qmd", "spwn:qmd", ""},
		{"bare match with dash", "paper-reading", "spwn:paper-reading", ""},
		{"bare match preserves version", "qmd@1.0", "spwn:qmd@1.0", ""},
		{"bare match trims whitespace", "  unix  ", "spwn:unix", ""},

		// Explicit schemes pass through, including versions.
		{"explicit spwn", "spwn:python", "spwn:python", ""},
		{"explicit spwn version", "spwn:unix@24.04", "spwn:unix@24.04", ""},
		{"explicit skill", "skill:focus", "skill:focus", ""},
		{"explicit tool", "tool:my-parser", "tool:my-parser", ""},
		{"explicit hook", "hook:pre-spawn", "hook:pre-spawn", ""},
		{"explicit github", "github:acme/foo", "github:acme/foo", ""},
		{"explicit github version", "github:acme/foo@1.2.3", "github:acme/foo@1.2.3", ""},
		// Unknown spwn: passes grammar — caller validates existence.
		{"explicit spwn unknown", "spwn:nonesuch", "spwn:nonesuch", ""},

		// Bare name with no catalog match: error names the input and
		// lists the catalog so the user can correct their typo.
		{"bare miss", "nonesuch", "", "not in the catalog"},
		{"bare miss shows known list", "nonesuch", "", "known: paper-reading, python, qmd, unix"},
		{"bare miss suggests local scheme", "nonesuch", "", "skill:nonesuch"},

		// Invalid shapes fall through to the scheme-grammar error.
		{"legacy at prefix", "@acme/foo", "", "malformed"},
		{"retired local scheme", "local:foo", "", "malformed"},
		{"uppercase", "QMD", "", "malformed"},
		{"contains slash", "acme/foo", "", "malformed"},
		{"trailing dash", "qmd-", "", "malformed"},
		{"leading dash", "-qmd", "", "malformed"},
		{"empty", "", "", "empty"},
		{"whitespace only", "   ", "", "empty"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := refs.ResolveCLI(tc.input, catalog)
			if tc.errSub == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got != tc.want {
					t.Errorf("got %q, want %q", got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got success with %q", tc.errSub, got)
			}
			if !strings.Contains(err.Error(), tc.errSub) {
				t.Errorf("error %q missing substring %q", err.Error(), tc.errSub)
			}
		})
	}
}

// TestResolveCLI_EmptyCatalog verifies the resolver is usable even
// when the caller has no catalog entries to match against — explicit
// scheme refs still pass through, but bare names always miss.
func TestResolveCLI_EmptyCatalog(t *testing.T) {
	got, err := refs.ResolveCLI("spwn:unix", nil)
	if err != nil || got != "spwn:unix" {
		t.Errorf("explicit ref should pass through without catalog: got (%q, %v)", got, err)
	}

	_, err = refs.ResolveCLI("unix", nil)
	if err == nil {
		t.Error("bare ref should miss against empty catalog")
	}
	if !strings.Contains(err.Error(), "catalog is empty") {
		t.Errorf("empty-catalog error should say so: %v", err)
	}
}

// TestResolveCLI_NoAliasing verifies ResolveCLI never mutates its
// inputs and returns clean scheme-form strings. Regression guard for
// future edits that might accidentally append rather than rebuild.
func TestResolveCLI_NoAliasing(t *testing.T) {
	catalog := []string{"qmd"}
	input := "qmd@latest"
	got, err := refs.ResolveCLI(input, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if got != "spwn:qmd@latest" {
		t.Errorf("got %q, want spwn:qmd@latest", got)
	}
	if input != "qmd@latest" {
		t.Errorf("input was mutated: %q", input)
	}
}
