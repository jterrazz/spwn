package lockfile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/project/lockfile"
)

func TestLoad_missing(t *testing.T) {
	root := t.TempDir()
	l, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l != nil {
		t.Errorf("missing file should yield nil, got %+v", l)
	}
}

func TestLoadOrEmpty_missing(t *testing.T) {
	root := t.TempDir()
	l, err := lockfile.LoadOrEmpty(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if l == nil || l.Version != lockfile.CurrentVersion {
		t.Errorf("want fresh lockfile at current version, got %+v", l)
	}
}

func TestSaveLoad_roundtrip(t *testing.T) {
	root := t.TempDir()
	l := lockfile.Empty()
	l.Add("@spwn/unix", lockfile.Entry{Version: "24.04", Source: lockfile.SourceBuiltin})
	l.Add("@spwn/git", lockfile.Entry{Version: "2.43", Source: lockfile.SourceBuiltin})
	l.Add("@spwn/mempalace", lockfile.Entry{Version: "0.1.0", Source: lockfile.SourceBuiltin})

	if err := lockfile.Save(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := lockfile.Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, ref := range []string{"@spwn/unix", "@spwn/git", "@spwn/mempalace"} {
		if !got.Has(ref) {
			t.Errorf("lost %s after round-trip", ref)
		}
	}
	e := got.Plugins["@spwn/unix"]
	if e.Version != "24.04" || e.Source != lockfile.SourceBuiltin {
		t.Errorf("entry mangled: %+v", e)
	}
}

func TestSave_deterministicOrder(t *testing.T) {
	root := t.TempDir()
	l := lockfile.Empty()
	l.Add("@spwn/zebra", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})
	l.Add("@spwn/alpha", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})
	l.Add("@spwn/mango", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})

	if err := lockfile.Save(root, l); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, lockfile.FileName))
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
	l := lockfile.Empty()
	l.Add("@spwn/unix", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})
	l.Remove("@spwn/unix")
	if l.Has("@spwn/unix") {
		t.Error("Remove did not delete entry")
	}
}

func TestLoad_unsupportedVersion(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, lockfile.FileName),
		[]byte("version: 999\npackages: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lockfile.Load(root); err == nil {
		t.Error("expected error for version 999")
	}
}

func TestRefs(t *testing.T) {
	l := lockfile.Empty()
	l.Add("@spwn/zebra", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})
	l.Add("@spwn/alpha", lockfile.Entry{Version: "1", Source: lockfile.SourceBuiltin})
	got := l.Refs()
	if len(got) != 2 || got[0] != "@spwn/alpha" || got[1] != "@spwn/zebra" {
		t.Errorf("Refs sort broken: %v", got)
	}
}
