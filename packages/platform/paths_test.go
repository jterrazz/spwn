package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestBaseDir_WithSpwnHome(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")

	got := BaseDir()
	if got != "/tmp/test-spwn" {
		t.Errorf("BaseDir() = %q, want %q", got, "/tmp/test-spwn")
	}
}

func TestBaseDir_DefaultPath(t *testing.T) {
	t.Setenv("SPWN_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get user home dir: %v", err)
	}

	want := filepath.Join(home, ".spwn")
	got := BaseDir()
	if got != want {
		t.Errorf("BaseDir() = %q, want %q", got, want)
	}
}

func TestWorldsDir(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")

	want := "/tmp/test-spwn/worlds"
	got := WorldsDir()
	if got != want {
		t.Errorf("WorldsDir() = %q, want %q", got, want)
	}
}

func TestAgentsDir(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")

	want := "/tmp/test-spwn/agents"
	got := AgentsDir()
	if got != want {
		t.Errorf("AgentsDir() = %q, want %q", got, want)
	}
}

func TestAProjectsAgentsAndSkillsAreTheFoldersItsLayoutNames(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	got, _ := WithProject("/p", Folders{Agents: "crew", Skills: "lib/skills"}, func() ([2]string, error) {
		return [2]string{AgentsDir(), SkillsDir()}, nil
	})
	if got != [2]string{"/p/crew", "/p/lib/skills"} {
		t.Errorf("agents and skills in a project = %q, want /p/crew and /p/lib/skills", got)
	}
}

func TestWorldsDirIsTheUsersEvenInAProject(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	got, _ := WithProject("/p", Folders{Agents: "agents", Skills: "skills"}, func() (string, error) {
		return WorldsDir(), nil
	})
	if got != "/tmp/test-spwn/worlds" {
		t.Errorf("WorldsDir() in a project = %q, want the user's /tmp/test-spwn/worlds, never a folder of the project's root", got)
	}
}


func TestOrgPath(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")

	want := "/tmp/test-spwn/org.yaml"
	got := OrgPath()
	if got != want {
		t.Errorf("OrgPath() = %q, want %q", got, want)
	}
}

func TestSkillsDir(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")

	want := "/tmp/test-spwn/skills"
	got := SkillsDir()
	if got != want {
		t.Errorf("SkillsDir() = %q, want %q", got, want)
	}
}

func TestWithProject_TwoGoroutinesNeverSeeEachOthersRoot(t *testing.T) {
	roots := []string{filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")}
	var wg sync.WaitGroup
	for _, root := range roots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				seen, err := WithProject(root, Folders{}, func() (string, error) {
					runtime.Gosched()
					return ProjectRoot(), nil
				})
				if err != nil || seen != root {
					t.Errorf("act under %q saw root %q, err %v", root, seen, err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if got := ProjectRoot(); got != "" {
		t.Errorf("root after the acts = %q, want it cleared", got)
	}
}

func TestWithProject_RestoresTheProjectItFound(t *testing.T) {
	outer := filepath.Join(t.TempDir(), "outer")
	SetProject(outer, Folders{Agents: "crew"})
	t.Cleanup(func() { SetProject("", Folders{}) })
	_, _ = WithProject("/p", Folders{Agents: "agents"}, func() (int, error) { return 0, nil })
	if got := ProjectRoot(); got != outer {
		t.Errorf("root after the act = %q, want %q", got, outer)
	}
	if got, want := AgentsDir(), filepath.Join(outer, "crew"); got != want {
		t.Errorf("agents after the act = %q, want %q", got, want)
	}
}

func TestTheZeroFoldersAreTheDefaultNamesNeverTheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "site")
	got, _ := WithProject(root, Folders{Skills: "lib/skills"}, func() ([2]string, error) {
		return [2]string{AgentsDir(), SkillsDir()}, nil
	})
	if want := [2]string{filepath.Join(root, "agents"), filepath.Join(root, "lib", "skills")}; got != want {
		t.Errorf("folders = %q, want %q", got, want)
	}
	folders, _ := WithProject(root, Folders{}, func() (Folders, error) { return ProjectFolders(), nil })
	if want := (Folders{"agents", "skills", "tools", "hooks", "commands", "profiles"}); folders != want {
		t.Errorf("ProjectFolders() = %+v, want %+v", folders, want)
	}
}

func TestWithProject_ReturnsTheActsError(t *testing.T) {
	want := errors.New("boom")
	_, err := WithProject("/p", Folders{}, func() (int, error) { return 0, want })
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
	if got := ProjectRoot(); got != "" {
		t.Errorf("root after a failing act = %q, want it cleared", got)
	}
}
