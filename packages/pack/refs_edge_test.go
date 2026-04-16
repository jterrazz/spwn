package pack_test

import (
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/pack"
)

// ---------- Parse edge cases ----------

func TestParse_EmptyString(t *testing.T) {
	got := pack.ParseRef("")
	if got.Kind != pack.KindLocal {
		t.Errorf("kind: want KindLocal, got %v", got.Kind)
	}
	if got.Name != "" {
		t.Errorf("name: want empty, got %q", got.Name)
	}
}

func TestParse_WhitespaceOnly(t *testing.T) {
	got := pack.ParseRef("  ")
	if got.Kind != pack.KindLocal {
		t.Errorf("kind: want KindLocal, got %v", got.Kind)
	}
	if got.Name != "" {
		t.Errorf("name: want empty after trim, got %q", got.Name)
	}
}

func TestParse_ScopeWithNoName(t *testing.T) {
	got := pack.ParseRef("@spwn/")
	if got.Kind != pack.KindSpwnBuiltin {
		t.Errorf("kind: want KindSpwnBuiltin, got %v", got.Kind)
	}
	if got.Owner != "spwn" {
		t.Errorf("owner: want %q, got %q", "spwn", got.Owner)
	}
	if got.Name != "" {
		t.Errorf("name: want empty, got %q", got.Name)
	}
}

func TestParse_EmptyOwner(t *testing.T) {
	got := pack.ParseRef("@/foo")
	// Owner is empty string, which is != "spwn", so KindRegistry.
	if got.Kind != pack.KindRegistry {
		t.Errorf("kind: want KindRegistry, got %v", got.Kind)
	}
	if got.Owner != "" {
		t.Errorf("owner: want empty, got %q", got.Owner)
	}
	if got.Name != "foo" {
		t.Errorf("name: want %q, got %q", "foo", got.Name)
	}
}

func TestParse_VersionedRefNotStripped(t *testing.T) {
	// Parse does NOT strip the @version suffix — that is the caller's job
	// via SplitVersion. So "@spwn/unix@24.04" should be parsed with
	// the version baked into the name.
	got := pack.ParseRef("@spwn/unix@24.04")
	if got.Kind != pack.KindSpwnBuiltin {
		t.Errorf("kind: want KindSpwnBuiltin, got %v", got.Kind)
	}
	if got.Name != "unix@24.04" {
		t.Errorf("name: want %q, got %q", "unix@24.04", got.Name)
	}
}

// ---------- SplitVersion edge cases ----------

func TestSplitVersion_ScopedVersioned(t *testing.T) {
	pack, version := pack.SplitVersion("@spwn/unix@24.04")
	if pack != "@spwn/unix" {
		t.Errorf("pack: want %q, got %q", "@spwn/unix", pack)
	}
	if version != "24.04" {
		t.Errorf("version: want %q, got %q", "24.04", version)
	}
}

func TestSplitVersion_NoVersion(t *testing.T) {
	pack, version := pack.SplitVersion("@spwn/unix")
	if pack != "@spwn/unix" {
		t.Errorf("pack: want %q, got %q", "@spwn/unix", pack)
	}
	if version != "" {
		t.Errorf("version: want empty, got %q", version)
	}
}

func TestSplitVersion_GithubVersioned(t *testing.T) {
	pack, version := pack.SplitVersion("github.com/owner/repo@v1.2.3")
	if pack != "github.com/owner/repo" {
		t.Errorf("pack: want %q, got %q", "github.com/owner/repo", pack)
	}
	if version != "v1.2.3" {
		t.Errorf("version: want %q, got %q", "v1.2.3", version)
	}
}

// ---------- ResolveTool edge cases ----------

func TestResolveTool_EmptyRoot(t *testing.T) {
	// Empty root means the path spwn/packs/<name> is relative to "".
	// The directory almost certainly does not exist, so expect NotFound.
	got := pack.ResolveTool("", pack.ParseRef("something"), nil, false)
	if got != pack.ResolveNotFound {
		t.Errorf("empty root local: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalNameWithSlash(t *testing.T) {
	root := t.TempDir()
	got := pack.ResolveTool(root, pack.Ref{Kind: pack.KindLocal, Name: "foo/bar"}, nil, false)
	if got != pack.ResolveNotFound {
		t.Errorf("name with slash: want NotFound, got %v", got)
	}
}

func TestResolveTool_LocalNameWithDotDot(t *testing.T) {
	root := t.TempDir()
	got := pack.ResolveTool(root, pack.Ref{Kind: pack.KindLocal, Name: "../escape"}, nil, false)
	if got != pack.ResolveNotFound {
		t.Errorf("name with ..: want NotFound, got %v", got)
	}
}

func TestResolveTool_BuiltinWithoutCatalog(t *testing.T) {
	// haveCatalog=false should accept any well-formed builtin.
	got := pack.ResolveTool("", pack.ParseRef("@spwn/anything"), nil, false)
	if got != pack.ResolveOK {
		t.Errorf("builtin without catalog: want OK, got %v", got)
	}
}

func TestResolveTool_BuiltinWithCatalogMissing(t *testing.T) {
	catalog := map[string]struct{}{
		"@spwn/unix": {},
	}
	got := pack.ResolveTool("", pack.ParseRef("@spwn/not-in-catalog"), catalog, true)
	if got != pack.ResolveNotFound {
		t.Errorf("builtin missing from catalog: want NotFound, got %v", got)
	}
}

// ---------- ResolveSkill edge cases ----------

func TestResolveSkill_MdPathIsDirectory(t *testing.T) {
	root := t.TempDir()
	// Create a directory named "trick.md" instead of a file.
	mustMkdirEdge(t, filepath.Join(root, "spwn", "skills", "trick.md"))

	got := pack.ResolveSkill(root, pack.ParseRef("trick"), nil, false)
	// The .md path exists but is a directory, not a file — should NOT resolve
	// via the file-form path. And there is no packs/trick/ dir either.
	if got != pack.ResolveNotFound {
		t.Errorf("md-is-directory skill: want NotFound, got %v", got)
	}
}

func TestResolveSkill_EmptyPackDir(t *testing.T) {
	root := t.TempDir()
	// Create an empty pack directory (no spwn.yaml or anything).
	mustMkdirEdge(t, filepath.Join(root, "spwn", "tools", "empty-pack"))

	got := pack.ResolveSkill(root, pack.ParseRef("empty-pack"), nil, false)
	// The directory exists, so ResolveSkill should return OK (it does not
	// validate contents).
	if got != pack.ResolveOK {
		t.Errorf("empty pack dir skill: want OK, got %v", got)
	}
}

func TestResolveSkill_RegistryAlwaysUnsupported(t *testing.T) {
	got := pack.ResolveSkill("", pack.ParseRef("@acme/foo"), nil, false)
	if got != pack.ResolveRegistryUnsupported {
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
