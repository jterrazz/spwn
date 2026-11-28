package agent

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"spwn.sh/packages/platform"
)

func linkedProject(t *testing.T) (agents, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	root, outside := filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	agents = filepath.Join(root, "agents")
	for _, d := range []string{filepath.Join(agents, "neo"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "rc"), []byte("runtime: outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	was := platform.ProjectRoot()
	platform.SetProjectRoot(root)
	t.Cleanup(func() { platform.SetProjectRoot(was) })
	return agents, outside
}

func outsideUntouched(t *testing.T, outside string) {
	t.Helper()
	if data, _ := os.ReadFile(filepath.Join(outside, "rc")); string(data) != "runtime: outside\n" {
		t.Errorf("the outside file was written: %q", data)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 1 {
		t.Errorf("files were created outside: %v", entries)
	}
}

func TestAnAgentYAMLLinkedOutIsNeitherReadNorWrittenThrough(t *testing.T) {
	refusals := map[bool]string{}
	for _, exists := range []bool{true, false} {
		agents, outside := linkedProject(t)
		target := filepath.Join(outside, "rc")
		if !exists {
			target = filepath.Join(outside, "missing")
		}
		if err := os.Symlink(target, filepath.Join(agents, "neo", "agent.yaml")); err != nil {
			t.Fatal(err)
		}
		m, err := LoadManifestPath(AgentDir("neo"))
		if err == nil {
			t.Fatalf("an agent.yaml linked out was read: %+v", m)
		}
		refusals[exists] = strings.ReplaceAll(err.Error(), agents, "<agents>")
		if err := SaveManifest("neo", &Manifest{}); err == nil {
			t.Error("an agent.yaml linked out was written through")
		}
		outsideUntouched(t, outside)
	}
	if refusals[true] != refusals[false] {
		t.Errorf("the refusal tells what is outside: %q vs %q", refusals[true], refusals[false])
	}
}

func TestAJournalLinkedOutIsNotWrittenInto(t *testing.T) {
	agents, outside := linkedProject(t)
	if err := os.Symlink(outside, filepath.Join(agents, "neo", "journal")); err != nil {
		t.Fatal(err)
	}
	mind := filepath.Join(agents, "neo")
	if err := AppendJournal(mind, "w1", 0, time.Second); err == nil {
		t.Error("a journal entry was written through a link out")
	}
	if err := SaveSession(mind, &Session{ID: "s", AgentName: "neo", WorldID: "w1"}); err == nil {
		t.Error("a session was written through a link out")
	}
	outsideUntouched(t, outside)
}

func TestAnAgentYAMLThatIsAFIFODoesNotBlock(t *testing.T) {
	agents, _ := linkedProject(t)
	if err := syscall.Mkfifo(filepath.Join(agents, "neo", "agent.yaml"), 0o644); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	done := make(chan error, 1)
	go func() { _, err := LoadManifestPath(AgentDir("neo")); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a FIFO agent.yaml was read as a manifest")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO agent.yaml blocked the read")
	}
}

func TestAnAgentInsideTheProjectIsSavedAndLoaded(t *testing.T) {
	linkedProject(t)
	if err := SaveManifest("neo", &Manifest{Description: "inside"}); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifestPath(AgentDir("neo"))
	if err != nil || m == nil || m.Description != "inside" {
		t.Fatalf("loaded %+v %v", m, err)
	}
	if err := AppendJournal(AgentDir("neo"), "w1", 0, time.Second); err != nil {
		t.Fatal(err)
	}
}
