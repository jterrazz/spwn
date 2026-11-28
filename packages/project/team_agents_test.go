package project

import (
	"os"
	"path/filepath"
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
	agents = filepath.Join(root, "agents")
	for _, d := range []string{agents, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"rc", "agent.yaml"} {
		if err := os.WriteFile(filepath.Join(outside, f), []byte("team: ops\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	was := platform.ProjectRoot()
	platform.SetProject(root, platform.Folders{Agents: "agents", Skills: "skills", Tools: "tools", Hooks: "hooks", Commands: "commands", Profiles: "profiles"})
	t.Cleanup(func() {
		platform.SetProject(was, platform.Folders{Agents: "agents", Skills: "skills", Tools: "tools", Hooks: "hooks", Commands: "commands", Profiles: "profiles"})
	})
	return agents, outside
}

func TestTeamMembersReadsNoAgentYAMLThroughALinkOut(t *testing.T) {
	agents, outside := projectAgents(t)
	if err := os.Mkdir(filepath.Join(agents, "neo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "rc"), filepath.Join(agents, "neo", "agent.yaml")); err != nil {
		t.Fatal(err)
	}
	if members, _ := TeamMembers("ops"); len(members) != 0 {
		t.Errorf("an outside agent.yaml was read: %v", members)
	}
}

func TestTeamMembersReadsNoLinkedAgentFolder(t *testing.T) {
	agents, outside := projectAgents(t)
	if err := os.Symlink(outside, filepath.Join(agents, "neo")); err != nil {
		t.Fatal(err)
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
	done := make(chan error, 1)
	go func() { _, err := TeamMembers("ops"); done <- err }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO agent.yaml blocked the team read")
	}
}

func TestTeamMembersReadsTheAgentsInsideTheProject(t *testing.T) {
	agents, _ := projectAgents(t)
	if err := os.MkdirAll(filepath.Join(agents, "neo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agents, "neo", "agent.yaml"), []byte("runtime: claude-code\nteam: ops\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if members, err := TeamMembers("ops"); err != nil || len(members) != 1 || members[0] != "neo" {
		t.Fatalf("members = %v %v", members, err)
	}
}
