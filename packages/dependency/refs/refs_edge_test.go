package refs_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency/refs"
)

// ---------- Parse edge cases ----------

func TestParse_EmptyString(t *testing.T) {
	got := refs.ParseRef("")
	if got.Kind != refs.KindInvalid {
		t.Errorf("kind: want KindInvalid, got %v", got.Kind)
	}
	if got.Name != "" {
		t.Errorf("name: want empty, got %q", got.Name)
	}
}

func TestParse_WhitespaceOnly(t *testing.T) {
	got := refs.ParseRef("  ")
	if got.Kind != refs.KindInvalid {
		t.Errorf("kind: want KindInvalid, got %v", got.Kind)
	}
	if got.Name != "" {
		t.Errorf("name: want empty after trim, got %q", got.Name)
	}
}

func TestParse_ScopeWithNoName(t *testing.T) {
	got := refs.ParseRef("spwn:")
	if got.Kind != refs.KindSpwnBuiltin {
		t.Errorf("kind: want KindSpwnBuiltin, got %v", got.Kind)
	}
	if got.Owner != "spwn" {
		t.Errorf("owner: want %q, got %q", "spwn", got.Owner)
	}
	if got.Name != "" {
		t.Errorf("name: want empty, got %q", got.Name)
	}
}

// TestParse_AtPrefixMalformed: any `@`-prefixed ref now parses as
// malformed (KindInvalid) under the new scheme-only grammar.
func TestParse_AtPrefixMalformed(t *testing.T) {
	for _, in := range []string{"@/foo", "@spwn", "@"} {
		got := refs.ParseRef(in)
		if got.Kind != refs.KindInvalid {
			t.Errorf("ParseRef(%q) kind = %v, want KindInvalid", in, got.Kind)
		}
	}
}

func TestParse_VersionedRefNotStripped(t *testing.T) {
	// Parse does NOT strip the @version suffix — that is the caller's job
	// via SplitVersion. So "spwn:unix@24.04" should be parsed with
	// the version baked into the name.
	got := refs.ParseRef("spwn:unix@24.04")
	if got.Kind != refs.KindSpwnBuiltin {
		t.Errorf("kind: want KindSpwnBuiltin, got %v", got.Kind)
	}
	if got.Name != "unix@24.04" {
		t.Errorf("name: want %q, got %q", "unix@24.04", got.Name)
	}
}

// ---------- SplitVersion edge cases ----------

func TestSplitVersion_ScopedVersioned(t *testing.T) {
	dependency, version := refs.SplitVersion("spwn:unix@24.04")
	if dependency != "spwn:unix" {
		t.Errorf("dependency: want %q, got %q", "spwn:unix", dependency)
	}
	if version != "24.04" {
		t.Errorf("version: want %q, got %q", "24.04", version)
	}
}

func TestSplitVersion_NoVersion(t *testing.T) {
	dependency, version := refs.SplitVersion("spwn:unix")
	if dependency != "spwn:unix" {
		t.Errorf("dependency: want %q, got %q", "spwn:unix", dependency)
	}
	if version != "" {
		t.Errorf("version: want empty, got %q", version)
	}
}

func TestSplitVersion_GithubVersioned(t *testing.T) {
	dependency, version := refs.SplitVersion("github.com/owner/repo@v1.2.3")
	if dependency != "github.com/owner/repo" {
		t.Errorf("dependency: want %q, got %q", "github.com/owner/repo", dependency)
	}
	if version != "v1.2.3" {
		t.Errorf("version: want %q, got %q", "v1.2.3", version)
	}
}

// ---------- ResolveTool edge cases ----------

func TestResolveTool_EmptyRoot(t *testing.T) {
	// Empty root means the path spwn/tools/<name>/ is relative to "".
	// The directory almost certainly does not exist, so expect NotFound.
	got := refs.ResolveTool("", refs.ParseRef("tool:something"), nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("empty root local: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalNameWithSlash(t *testing.T) {
	root := t.TempDir()
	got := refs.ResolveTool(root, refs.Ref{Kind: refs.KindLocalTool, Name: "foo/bar"}, nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("name with slash: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalNameWithDotDot(t *testing.T) {
	root := t.TempDir()
	got := refs.ResolveTool(root, refs.Ref{Kind: refs.KindLocalTool, Name: "../escape"}, nil, false)
	if got != refs.ResolveNotFound {
		t.Errorf("name with ..: want NotFound, got %v", got)
	}
}

func TestResolveTool_BuiltinWithoutCatalog(t *testing.T) {
	// haveCatalog=false should accept any well-formed builtin.
	got := refs.ResolveTool("", refs.ParseRef("spwn:anything"), nil, false)
	if got != refs.ResolveOK {
		t.Errorf("builtin without catalog: want OK, got %v", got)
	}
}

func TestResolveTool_BuiltinWithCatalogMissing(t *testing.T) {
	catalog := map[string]struct{}{
		"spwn:unix": {},
	}
	got := refs.ResolveTool("", refs.ParseRef("spwn:not-in-catalog"), catalog, true)
	if got != refs.ResolveNotFound {
		t.Errorf("builtin missing from catalog: want NotFound, got %v", got)
	}
}

// ---------- ResolveSkill edge cases ----------

func TestResolveSkill_MdPathIsDirectory(t *testing.T) {
	root := t.TempDir()
	// Create a directory named "trick.md" instead of a file.
	mustMkdirEdge(t, filepath.Join(root, "spwn", "skills", "trick.md"))

	got := refs.ResolveSkill(root, refs.ParseRef("skill:trick"), nil, false)
	// The .md path exists but is a directory, not a file — should NOT resolve.
	if got != refs.ResolveNotFound {
		t.Errorf("md-is-directory skill: want NotFound, got %v", got)
	}
}

func TestResolveSkill_EmptyToolDir(t *testing.T) {
	root := t.TempDir()
	// Create an empty tool directory (no spwn.yaml or anything).
	mustMkdirEdge(t, filepath.Join(root, "spwn", "tools", "empty-tool"))

	got := refs.ResolveSkill(root, refs.ParseRef("tool:empty-tool"), nil, false)
	// The directory exists, so ResolveSkill should return OK (it does not
	// validate contents).
	if got != refs.ResolveOK {
		t.Errorf("empty tool dir: want OK, got %v", got)
	}
}

func TestResolveSkill_RegistryAlwaysUnsupported(t *testing.T) {
	got := refs.ResolveSkill("", refs.ParseRef("github:acme/foo"), nil, false)
	if got != refs.ResolveRegistryUnsupported {
		t.Errorf("registry skill: want RegistryUnsupported, got %v", got)
	}
}

// ---------- helper ----------

func mustMkdirEdge(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
