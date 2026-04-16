package dependency_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/dependency"
)

func TestLoad_missing(t *testing.T) {
	root := t.TempDir()
	l, err := dependency.LoadLockfile(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l != nil {
		t.Errorf("missing file should yield nil, got %+v", l)
	}
}

func TestLoadOrEmpty_missing(t *testing.T) {
	root := t.TempDir()
	l, err := dependency.LoadLockfileOrEmpty(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Error("want fresh lockfile, got nil")
	}
}

func TestSaveLoad_roundtrip(t *testing.T) {
	root := t.TempDir()
	l := dependency.EmptyLockfile()
	l.Add("@spwn/unix", dependency.LockEntry{Version: "24.04", Source: dependency.SourceBuiltin})
	l.Add("@spwn/git", dependency.LockEntry{Version: "2.43", Source: dependency.SourceBuiltin})
	l.Add("@spwn/mempalace", dependency.LockEntry{Version: "0.1.0", Source: dependency.SourceBuiltin})

	if err := dependency.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := dependency.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, ref := range []string{"@spwn/unix", "@spwn/git", "@spwn/mempalace"} {
		if !got.Has(ref) {
			t.Errorf("lost %s after round-trip", ref)
		}
	}
	e := got.Deps["@spwn/unix"]
	if e.Version != "24.04" || e.Source != dependency.SourceBuiltin {
		t.Errorf("entry mangled: %+v", e)
	}
}

func TestSave_lineOriented(t *testing.T) {
	root := t.TempDir()
	l := dependency.EmptyLockfile()
	l.Add("@spwn/unix", dependency.LockEntry{Version: "24.04", Source: dependency.SourceBuiltin})

	if err := dependency.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, dependency.LockFileName))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# spwn.lock") {
		t.Errorf("missing header comment")
	}
	if !strings.Contains(content, "@spwn/unix 24.04 builtin") {
		t.Errorf("expected line-oriented format, got:\n%s", content)
	}
}

func TestSave_deterministicOrder(t *testing.T) {
	root := t.TempDir()
	l := dependency.EmptyLockfile()
	l.Add("@spwn/zebra", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	l.Add("@spwn/alpha", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	l.Add("@spwn/mango", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})

	if err := dependency.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, dependency.LockFileName))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	alpha := strings.Index(content, "@spwn/alpha")
	mango := strings.Index(content, "@spwn/mango")
	zebra := strings.Index(content, "@spwn/zebra")
	if !(alpha < mango && mango < zebra) {
		t.Errorf("keys not sorted:\n%s", content)
	}
}

func TestRemove(t *testing.T) {
	l := dependency.EmptyLockfile()
	l.Add("@spwn/unix", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	l.Remove("@spwn/unix")
	if l.Has("@spwn/unix") {
		t.Error("Remove did not delete entry")
	}
}

func TestLoad_legacyYAML(t *testing.T) {
	root := t.TempDir()
	yaml := `version: 1
deps:
  "@spwn/unix":
    version: "24.04"
    source: builtin
  "@spwn/git":
    version: "2.43"
    source: builtin
`
	if err := os.WriteFile(filepath.Join(root, dependency.LockFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := dependency.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load legacy: %v", err)
	}
	if !l.Has("@spwn/unix") || !l.Has("@spwn/git") {
		t.Errorf("legacy parse failed: %+v", l.Deps)
	}
}

func TestRefs(t *testing.T) {
	l := dependency.EmptyLockfile()
	l.Add("@spwn/zebra", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	l.Add("@spwn/alpha", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	got := l.Refs()
	if len(got) != 2 || got[0] != "@spwn/alpha" || got[1] != "@spwn/zebra" {
		t.Errorf("Refs sort broken: %v", got)
	}
}
