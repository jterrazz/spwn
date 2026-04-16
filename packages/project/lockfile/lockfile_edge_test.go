package lockfile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/project/lockfile"
)

// 1. Empty lockfile (just the header comment)
func TestLoad_emptyWithHeaderOnly(t *testing.T) {
	root := t.TempDir()
	content := "# spwn.lock — DO NOT EDIT\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil lockfile for empty file")
	}
	if len(l.Deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(l.Deps))
	}
}

// 2. Lockfile with only comments and blank lines
func TestLoad_onlyCommentsAndBlanks(t *testing.T) {
	root := t.TempDir()
	content := "# spwn.lock — DO NOT EDIT\n\n# another comment\n\n# yet another\n\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil lockfile")
	}
	if len(l.Deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(l.Deps))
	}
}

// 3. Entry with no version or source (just a ref name)
func TestLoad_entryWithRefOnly(t *testing.T) {
	root := t.TempDir()
	content := "# spwn.lock — DO NOT EDIT\n@spwn/solo\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil lockfile")
	}
	// A single-field line has len(parts) < 2, so it should be skipped.
	if l.Has("@spwn/solo") {
		t.Error("single-field line should be skipped")
	}
}

// 4. Entry with version but no source
func TestLoad_entryWithVersionNoSource(t *testing.T) {
	root := t.TempDir()
	content := "# spwn.lock — DO NOT EDIT\n@spwn/partial v1.0\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !l.Has("@spwn/partial") {
		t.Fatal("expected entry to be parsed")
	}
	e := l.Deps["@spwn/partial"]
	if e.Version != "v1.0" {
		t.Errorf("version = %q, want %q", e.Version, "v1.0")
	}
	// Default source should be builtin when no 3rd field is present.
	if e.Source != lockfile.SourceBuiltin {
		t.Errorf("source = %q, want %q", e.Source, lockfile.SourceBuiltin)
	}
}

// 5. Entry with github.com/ style refs
func TestLoad_githubStyleRef(t *testing.T) {
	root := t.TempDir()
	content := "# spwn.lock — DO NOT EDIT\ngithub.com/jterrazz/research-skills v0.3.0 github\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !l.Has("github.com/jterrazz/research-skills") {
		t.Fatal("github ref not found")
	}
	e := l.Deps["github.com/jterrazz/research-skills"]
	if e.Version != "v0.3.0" {
		t.Errorf("version = %q, want %q", e.Version, "v0.3.0")
	}
	if e.Source != lockfile.SourceGitHub {
		t.Errorf("source = %q, want %q", e.Source, lockfile.SourceGitHub)
	}
}

// 6. Round-trip: save then load preserves github refs
func TestRoundtrip_githubRefs(t *testing.T) {
	root := t.TempDir()
	l := lockfile.Empty()
	l.Add("github.com/jterrazz/research-skills", lockfile.Entry{
		Version: "v0.3.0",
		Source:  lockfile.SourceGitHub,
	})
	l.Add("@spwn/unix", lockfile.Entry{
		Version: "24.04",
		Source:  lockfile.SourceBuiltin,
	})

	if err := lockfile.Save(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if !got.Has("github.com/jterrazz/research-skills") {
		t.Error("github ref lost after round-trip")
	}
	if !got.Has("@spwn/unix") {
		t.Error("spwn ref lost after round-trip")
	}
	e := got.Deps["github.com/jterrazz/research-skills"]
	if e.Version != "v0.3.0" || e.Source != lockfile.SourceGitHub {
		t.Errorf("github entry mangled: %+v", e)
	}
}

// 7. Multiple Add() calls for same ref (last wins)
func TestAdd_lastWins(t *testing.T) {
	l := lockfile.Empty()
	l.Add("@spwn/unix", lockfile.Entry{Version: "1.0", Source: lockfile.SourceBuiltin})
	l.Add("@spwn/unix", lockfile.Entry{Version: "2.0", Source: lockfile.SourceGitHub})

	e := l.Deps["@spwn/unix"]
	if e.Version != "2.0" {
		t.Errorf("version = %q, want %q (last wins)", e.Version, "2.0")
	}
	if e.Source != lockfile.SourceGitHub {
		t.Errorf("source = %q, want %q (last wins)", e.Source, lockfile.SourceGitHub)
	}
}

// 8. Remove() on non-existent ref (no panic)
func TestRemove_nonExistent(t *testing.T) {
	l := lockfile.Empty()
	l.Add("@spwn/unix", lockfile.Entry{Version: "1.0", Source: lockfile.SourceBuiltin})

	// Should not panic.
	l.Remove("@spwn/nonexistent")

	if !l.Has("@spwn/unix") {
		t.Error("existing entry should not be affected")
	}
}

// 9. Has() on empty lockfile
func TestHas_emptyLockfile(t *testing.T) {
	l := lockfile.Empty()
	if l.Has("@spwn/anything") {
		t.Error("Has should return false on empty lockfile")
	}
}

// 10. Refs() on empty lockfile returns empty slice
func TestRefs_emptyLockfile(t *testing.T) {
	l := lockfile.Empty()
	refs := l.Refs()
	if refs == nil {
		t.Error("Refs() should return non-nil empty slice, got nil")
	}
	if len(refs) != 0 {
		t.Errorf("Refs() should return empty slice, got %v", refs)
	}
}

// 11. Save to non-existent directory fails gracefully
func TestSave_nonExistentDirectory(t *testing.T) {
	l := lockfile.Empty()
	l.Add("@spwn/unix", lockfile.Entry{Version: "1.0", Source: lockfile.SourceBuiltin})

	err := lockfile.Save("/nonexistent/path/that/does/not/exist", l)
	if err == nil {
		t.Fatal("expected error when saving to non-existent directory")
	}
	if !strings.Contains(err.Error(), "write") {
		t.Errorf("error should mention write, got: %v", err)
	}
}

// 12. Load legacy YAML with empty deps map
func TestLoad_legacyYAMLEmptyDeps(t *testing.T) {
	root := t.TempDir()
	yaml := "version: 1\ndeps:\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Fatal("expected non-nil lockfile")
	}
	if len(l.Deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(l.Deps))
	}
}

// 13. Load legacy YAML with missing version field
func TestLoad_legacyYAMLMissingVersion(t *testing.T) {
	root := t.TempDir()
	yaml := "version: 1\ndeps:\n  \"@spwn/unix\":\n    source: builtin\n"
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !l.Has("@spwn/unix") {
		t.Fatal("expected @spwn/unix to be present")
	}
	e := l.Deps["@spwn/unix"]
	if e.Version != "" {
		t.Errorf("version should be empty, got %q", e.Version)
	}
	if e.Source != lockfile.SourceBuiltin {
		t.Errorf("source = %q, want %q", e.Source, lockfile.SourceBuiltin)
	}
}

// 14. File with mixed comments and entries
func TestLoad_mixedCommentsAndEntries(t *testing.T) {
	root := t.TempDir()
	content := `# spwn.lock — DO NOT EDIT
# This is a comment
@spwn/unix 24.04 builtin

# Another comment between entries
@spwn/git 2.43 builtin
# Trailing comment
`
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(l.Deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(l.Deps))
	}
	if !l.Has("@spwn/unix") {
		t.Error("missing @spwn/unix")
	}
	if !l.Has("@spwn/git") {
		t.Error("missing @spwn/git")
	}
}

// 15. Very long ref names and versions
func TestLoad_veryLongRefAndVersion(t *testing.T) {
	root := t.TempDir()
	longRef := "github.com/org/" + strings.Repeat("a", 200)
	longVersion := "v" + strings.Repeat("1", 200)

	l := lockfile.Empty()
	l.Add(longRef, lockfile.Entry{Version: longVersion, Source: lockfile.SourceGitHub})

	if err := lockfile.Save(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !got.Has(longRef) {
		t.Error("long ref lost after round-trip")
	}
	e := got.Deps[longRef]
	if e.Version != longVersion {
		t.Errorf("long version mangled: len=%d, want %d", len(e.Version), len(longVersion))
	}
}
