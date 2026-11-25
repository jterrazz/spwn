package manifest

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoad_MissingDefaultReadsTheBuiltInConfig(t *testing.T) {
	home := filepath.Join(t.TempDir(), "spwn-home")
	t.Setenv("SPWN_HOME", home)

	m, err := Load("default")
	if err != nil {
		t.Fatalf("Load(default): %v", err)
	}
	if want := []string{"spwn:unix", "spwn:git"}; !slices.Equal(m.Deps, want) {
		t.Errorf("deps = %v, want %v", m.Deps, want)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("Load(default) created %s (stat err %v)", home, err)
	}
}

func TestLoad_MissingNamedConfigFails(t *testing.T) {
	t.Setenv("SPWN_HOME", filepath.Join(t.TempDir(), "spwn-home"))

	if _, err := Load("ops"); err == nil {
		t.Fatal("Load(ops) on a fresh home returned no error")
	}
}

func TestLoad_WrittenDefaultWins(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)
	if err := os.MkdirAll(filepath.Join(home, "worlds"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "worlds", "default.yaml"), []byte("dependencies:\n  - \"spwn:node\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := Load("default")
	if err != nil {
		t.Fatalf("Load(default): %v", err)
	}
	if want := []string{"spwn:node"}; !slices.Equal(m.Deps, want) {
		t.Errorf("deps = %v, want %v", m.Deps, want)
	}
}

func TestCreateDefault_WritesTheBuiltInConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SPWN_HOME", home)

	if err := CreateDefault(); err != nil {
		t.Fatalf("CreateDefault: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(home, "worlds", "default.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != defaultContent {
		t.Errorf("default.yaml = %q, want %q", got, defaultContent)
	}
}
