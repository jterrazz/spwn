package foundation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBaseDir_WithSpwnHome(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	got := BaseDir()
	if got != "/tmp/test-spwn" {
		t.Errorf("BaseDir() = %q, want %q", got, "/tmp/test-spwn")
	}
}

func TestBaseDir_WithUniverseHomeFallback(t *testing.T) {
	t.Setenv("SPWN_HOME", "")
	t.Setenv("UNIVERSE_HOME", "/tmp/test-universe")

	got := BaseDir()
	if got != "/tmp/test-universe" {
		t.Errorf("BaseDir() = %q, want %q", got, "/tmp/test-universe")
	}
}

func TestBaseDir_SpwnHomeTakesPrecedence(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/spwn-wins")
	t.Setenv("UNIVERSE_HOME", "/tmp/universe-loses")

	got := BaseDir()
	if got != "/tmp/spwn-wins" {
		t.Errorf("BaseDir() = %q, want %q when both envs set", got, "/tmp/spwn-wins")
	}
}

func TestBaseDir_DefaultPath(t *testing.T) {
	t.Setenv("SPWN_HOME", "")
	t.Setenv("UNIVERSE_HOME", "")

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

func TestUniversesDir(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	want := "/tmp/test-spwn/universes"
	got := UniversesDir()
	if got != want {
		t.Errorf("UniversesDir() = %q, want %q", got, want)
	}
}

func TestAgentsDir(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	want := "/tmp/test-spwn/agents"
	got := AgentsDir()
	if got != want {
		t.Errorf("AgentsDir() = %q, want %q", got, want)
	}
}

func TestStatePath(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	want := "/tmp/test-spwn/state.json"
	got := StatePath()
	if got != want {
		t.Errorf("StatePath() = %q, want %q", got, want)
	}
}

func TestOrgPath(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	want := "/tmp/test-spwn/org.yaml"
	got := OrgPath()
	if got != want {
		t.Errorf("OrgPath() = %q, want %q", got, want)
	}
}

func TestClawStatePath(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	want := "/tmp/test-spwn/claw/claw.json"
	got := ClawStatePath()
	if got != want {
		t.Errorf("ClawStatePath() = %q, want %q", got, want)
	}
}

func TestSkillsDir(t *testing.T) {
	t.Setenv("SPWN_HOME", "/tmp/test-spwn")
	t.Setenv("UNIVERSE_HOME", "")

	want := "/tmp/test-spwn/skills"
	got := SkillsDir()
	if got != want {
		t.Errorf("SkillsDir() = %q, want %q", got, want)
	}
}
