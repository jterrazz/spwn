package project

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"spwn.sh/packages/platform"
)

func projectAgents(t *testing.T) (agents, outside string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	root, outside := filepath.Join(dir, "site"), filepath.Join(dir, "outside")
	agents = filepath.Join(root, "spwn", "agents")
	for _, d := range []string{agents, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(outside, "rc"), []byte("team: ops\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	was := platform.ProjectRoot()
	platform.SetProjectRoot(root)
	t.Cleanup(func() { platform.SetProjectRoot(was) })
	return agents, outside
}

func TestATeamWritesNoAgentYAMLThroughALinkOut(t *testing.T) {
	refusals := map[bool]string{}
	for _, exists := range []bool{true, false} {
		agents, outside := projectAgents(t)
		target := filepath.Join(outside, "rc")
		if !exists {
			target = filepath.Join(outside, "missing")
		}
		if err := os.Mkdir(filepath.Join(agents, "neo"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(agents, "neo", "agent.yaml")); err != nil {
			t.Fatal(err)
		}
		err := SetAgentTeam("neo", "red")
		if err == nil {
			t.Fatal("an agent.yaml linked out was written through")
		}
		refusals[exists] = strings.ReplaceAll(err.Error(), agents, "<agents>")
		if data, _ := os.ReadFile(filepath.Join(outside, "rc")); string(data) != "team: ops\n" {
			t.Errorf("the outside file was written: %q", data)
		}
		if _, err := os.Lstat(filepath.Join(outside, "missing")); err == nil {
			t.Error("a file was created outside")
		}
		if members, _ := TeamMembers("ops"); len(members) != 0 {
			t.Errorf("an outside agent.yaml was read: %v", members)
		}
	}
	if refusals[true] != refusals[false] {
		t.Errorf("the refusal tells what is outside: %q vs %q", refusals[true], refusals[false])
	}
}

func TestATeamDoesNotWriteIntoALinkedAgentFolder(t *testing.T) {
	agents, outside := projectAgents(t)
	if err := os.Symlink(outside, filepath.Join(agents, "neo")); err != nil {
		t.Fatal(err)
	}
	if err := SetAgentTeam("neo", "red"); err == nil {
		t.Fatal("a linked agent folder was written into")
	}
	if _, err := os.Lstat(filepath.Join(outside, "agent.yaml")); err == nil {
		t.Error("agent.yaml was created outside")
	}
	if members, _ := TeamMembers("ops"); len(members) != 0 {
		t.Errorf("a linked agent folder was read: %v", members)
	}
}

func TestTeamMembersDoesNotBlockOnAFIFO(t *testing.T) {
	agents, _ := projectAgents(t)
	if err := os.Mkdir(filepath.Join(agents, "neo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(agents, "neo", "agent.yaml"), 0o644); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	done := make(chan error, 2)
	go func() { _, err := TeamMembers("ops"); done <- err }()
	go func() { done <- SetAgentTeam("neo", "red") }()
	for range 2 {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Fatal("a FIFO agent.yaml blocked the team command")
		}
	}
}

func TestATeamIsAssignedInsideTheProject(t *testing.T) {
	agents, _ := projectAgents(t)
	if err := os.MkdirAll(filepath.Join(agents, "neo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agents, "neo", "agent.yaml"), []byte("runtime: claude-code\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := SetAgentTeam("neo", "ops"); err != nil {
		t.Fatal(err)
	}
	if members, err := TeamMembers("ops"); err != nil || len(members) != 1 || members[0] != "neo" {
		t.Fatalf("members = %v %v", members, err)
	}
	if err := SetAgentTeam("../x", "ops"); err == nil {
		t.Error("an agent name that climbs was taken")
	}
}
