package runtimestate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/platform"
)

func projectStore(t *testing.T) (s *Store, states, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	root, outside := filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	for _, d := range []string{filepath.Join(root, ".spwn"), filepath.Join(outside, "w1")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "w1", "runtime.json"), []byte(`{"display_name":"outside"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	was := platform.ProjectRoot()
	platform.SetProjectRoot(root)
	t.Cleanup(func() { platform.SetProjectRoot(was) })
	states = filepath.Join(root, ".spwn", "world-states")
	return &Store{dir: states, tree: stateTreeAsFound, written: platform.StateTree}, states, outside
}

func outsideStateUntouched(t *testing.T, outside string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(outside, "w1", "runtime.json"))
	if err != nil || string(data) != `{"display_name":"outside"}` {
		t.Errorf("the outside runtime.json was removed or written: %q %v", data, err)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 1 {
		t.Errorf("files were created outside: %v", entries)
	}
}

func TestALinkedWorldStatesFolderIsNeitherCollectedReadNorWritten(t *testing.T) {
	s, states, outside := projectStore(t)
	if err := os.Symlink(outside, states); err != nil {
		t.Fatal(err)
	}
	_ = s.GC(nil)
	if name := s.GetSessionID("w1", "neo"); name != "" {
		t.Errorf("an outside runtime.json was read: %q", name)
	}
	if f, err := s.Load("w1"); err == nil && f.DisplayName != "" {
		t.Errorf("an outside runtime.json was read: %+v", f)
	}
	if err := s.SetDisplayName("w2", "x"); err == nil {
		t.Error("a runtime.json was written through a linked world-states folder")
	}
	if err := s.Delete("w1"); err == nil {
		t.Error("a delete went through a linked world-states folder")
	}
	outsideStateUntouched(t, outside)
}

func TestALinkedWorldFolderIsNeitherCollectedNorWrittenThrough(t *testing.T) {
	s, states, outside := projectStore(t)
	if err := os.MkdirAll(states, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "w1"), filepath.Join(states, "w1")); err != nil {
		t.Fatal(err)
	}
	if err := s.GC(nil); err != nil {
		t.Fatal(err)
	}
	_ = s.Delete("w1")
	err := s.SetDisplayName("w1", "x")
	if err == nil || !strings.Contains(err.Error(), "w1") {
		t.Errorf("a write through a linked world folder: %v", err)
	}
	outsideStateUntouched(t, outside)
}

func TestAWorldStateInsideTheProjectIsKeptAndCollected(t *testing.T) {
	s, states, _ := projectStore(t)
	if err := s.SetDisplayName("w1", "inside"); err != nil {
		t.Fatal(err)
	}
	if f, err := s.Load("w1"); err != nil || f.DisplayName != "inside" {
		t.Fatalf("Load = %+v %v", f, err)
	}
	if err := s.GC(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(states, "w1", "runtime.json")); err == nil {
		t.Error("GC left a dead world's runtime.json")
	}
}
