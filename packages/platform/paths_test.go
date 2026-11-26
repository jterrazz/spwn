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

func TestWithProjectRoot_TwoGoroutinesNeverSeeEachOthersRoot(t *testing.T) {
	roots := []string{filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")}
	var wg sync.WaitGroup
	for _, root := range roots {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				seen, err := WithProjectRoot(root, func() (string, error) {
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

func TestWithProjectRoot_RestoresTheRootItFound(t *testing.T) {
	outer := filepath.Join(t.TempDir(), "outer")
	SetProjectRoot(outer)
	t.Cleanup(func() { SetProjectRoot("") })
	_, _ = WithProjectRoot("/p", func() (int, error) { return 0, nil })
	if got := ProjectRoot(); got != outer {
		t.Errorf("root after the act = %q, want %q", got, outer)
	}
}

func TestWithProjectRoot_ReturnsTheActsError(t *testing.T) {
	want := errors.New("boom")
	_, err := WithProjectRoot("/p", func() (int, error) { return 0, want })
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
	if got := ProjectRoot(); got != "" {
		t.Errorf("root after a failing act = %q, want it cleared", got)
	}
}
