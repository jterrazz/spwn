package foundation

import (
	"os"
	"path/filepath"
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

func TestStatePath(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")

	want := "/tmp/test-spwn/state.json"
	got := StatePath()
	if got != want {
		t.Errorf("StatePath() = %q, want %q", got, want)
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
