package pack_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/pack"
)

func TestLoad_missing(t *testing.T) {
	root := t.TempDir()
	l, err := pack.LoadLockfile(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l != nil {
		t.Errorf("missing file should yield nil, got %+v", l)
	}
}

func TestLoadOrEmpty_missing(t *testing.T) {
	root := t.TempDir()
	l, err := pack.LoadLockfileOrEmpty(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil {
		t.Error("want fresh lockfile, got nil")
	}
}

func TestSaveLoad_roundtrip(t *testing.T) {
	root := t.TempDir()
	l := pack.EmptyLockfile()
	l.Add("@spwn/unix", pack.LockEntry{Version: "24.04", Source: pack.SourceBuiltin})
	l.Add("@spwn/git", pack.LockEntry{Version: "2.43", Source: pack.SourceBuiltin})
	l.Add("@spwn/mempalace", pack.LockEntry{Version: "0.1.0", Source: pack.SourceBuiltin})

	if err := pack.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := pack.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, ref := range []string{"@spwn/unix", "@spwn/git", "@spwn/mempalace"} {
		if !got.Has(ref) {
			t.Errorf("lost %s after round-trip", ref)
		}
	}
	e := got.Deps["@spwn/unix"]
	if e.Version != "24.04" || e.Source != pack.SourceBuiltin {
		t.Errorf("entry mangled: %+v", e)
	}
}

func TestSave_lineOriented(t *testing.T) {
	root := t.TempDir()
	l := pack.EmptyLockfile()
	l.Add("@spwn/unix", pack.LockEntry{Version: "24.04", Source: pack.SourceBuiltin})

	if err := pack.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, pack.LockFileName))
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
	l := pack.EmptyLockfile()
	l.Add("@spwn/zebra", pack.LockEntry{Version: "1", Source: pack.SourceBuiltin})
	l.Add("@spwn/alpha", pack.LockEntry{Version: "1", Source: pack.SourceBuiltin})
	l.Add("@spwn/mango", pack.LockEntry{Version: "1", Source: pack.SourceBuiltin})

	if err := pack.SaveLockfile(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, pack.LockFileName))
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
	l := pack.EmptyLockfile()
	l.Add("@spwn/unix", pack.LockEntry{Version: "1", Source: pack.SourceBuiltin})
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
	if err := os.WriteFile(filepath.Join(root, pack.LockFileName), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err := pack.LoadLockfile(root)
	if err != nil {
		t.Fatalf("load legacy: %v", err)
	}
	if !l.Has("@spwn/unix") || !l.Has("@spwn/git") {
		t.Errorf("legacy parse failed: %+v", l.Deps)
	}
}

func TestRefs(t *testing.T) {
	l := pack.EmptyLockfile()
	l.Add("@spwn/zebra", pack.LockEntry{Version: "1", Source: pack.SourceBuiltin})
	l.Add("@spwn/alpha", pack.LockEntry{Version: "1", Source: pack.SourceBuiltin})
	got := l.Refs()
	if len(got) != 2 || got[0] != "@spwn/alpha" || got[1] != "@spwn/zebra" {
		t.Errorf("Refs sort broken: %v", got)
	}
}
