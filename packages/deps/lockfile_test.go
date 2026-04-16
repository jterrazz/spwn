package deps_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/deps"
)

func TestLoad_missing(t *testing.T) {
	root := t.TempDir()
	l, err := deps.LoadLockfile(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l != nil {
		t.Errorf("missing file should yield nil, got %+v", l)
	}
}

func TestLoadOrEmpty_missing(t *testing.T) {
	root := t.TempDir()
	l, err := deps.LoadLockfileOrEmpty(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Error("want fresh lockfile, got nil")
	}
}

func TestSaveLoad_roundtrip(t *testing.T) {
	root := t.TempDir()
	l := deps.EmptyLockfile()
	l.Add("@spwn/unix", deps.LockEntry{Version: "24.04", Source: deps.SourceBuiltin})
	l.Add("@spwn/git", deps.LockEntry{Version: "2.43", Source: deps.SourceBuiltin})
	l.Add("@spwn/mempalace", deps.LockEntry{Version: "0.1.0", Source: deps.SourceBuiltin})

	if err := deps.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := deps.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, ref := range []string{"@spwn/unix", "@spwn/git", "@spwn/mempalace"} {
		if !got.Has(ref) {
			t.Errorf("lost %s after round-trip", ref)
		}
	}
	e := got.Deps["@spwn/unix"]
	if e.Version != "24.04" || e.Source != deps.SourceBuiltin {
		t.Errorf("entry mangled: %+v", e)
	}
}

func TestSave_lineOriented(t *testing.T) {
	root := t.TempDir()
	l := deps.EmptyLockfile()
	l.Add("@spwn/unix", deps.LockEntry{Version: "24.04", Source: deps.SourceBuiltin})

	if err := deps.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, deps.LockFileName))
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
	l := deps.EmptyLockfile()
	l.Add("@spwn/zebra", deps.LockEntry{Version: "1", Source: deps.SourceBuiltin})
	l.Add("@spwn/alpha", deps.LockEntry{Version: "1", Source: deps.SourceBuiltin})
	l.Add("@spwn/mango", deps.LockEntry{Version: "1", Source: deps.SourceBuiltin})

	if err := deps.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, deps.LockFileName))
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
	l := deps.EmptyLockfile()
	l.Add("@spwn/unix", deps.LockEntry{Version: "1", Source: deps.SourceBuiltin})
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
	if err := os.WriteFile(filepath.Join(root, deps.LockFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := deps.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load legacy: %v", err)
	}
	if !l.Has("@spwn/unix") || !l.Has("@spwn/git") {
		t.Errorf("legacy parse failed: %+v", l.Deps)
	}
}

func TestRefs(t *testing.T) {
	l := deps.EmptyLockfile()
	l.Add("@spwn/zebra", deps.LockEntry{Version: "1", Source: deps.SourceBuiltin})
	l.Add("@spwn/alpha", deps.LockEntry{Version: "1", Source: deps.SourceBuiltin})
	got := l.Refs()
	if len(got) != 2 || got[0] != "@spwn/alpha" || got[1] != "@spwn/zebra" {
		t.Errorf("Refs sort broken: %v", got)
	}
}
