package world

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var unloadedManifests = map[string]struct{ manifest, code string }{
	"folders-invalid": {"version: 1\nname: site\nfolders:\n  agents: ..\nworlds:\n  ship: {}\n", "spwn-yaml.folders"},
	"not-yaml":        {"version: 1\nname: site\nworlds: [ship\n", "spwn-yaml.parse"},
}

func inUnloadedProject(t *testing.T, manifest string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	root := filepath.Join(dir, "site")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	saved := startWorld
	t.Cleanup(func() { startWorld = saved })
	startWorld = func(*cobra.Command, []string, *plannedWorld) error {
		t.Error("a world started from a spwn.yaml that does not load")
		return nil
	}
}

func TestUpStopsOnASpwnYamlThatDoesNotLoad(t *testing.T) {
	for name, c := range unloadedManifests {
		t.Run(name, func(t *testing.T) {
			inUnloadedProject(t, c.manifest)
			err := composeUpRunE(&cobra.Command{}, nil)
			if err == nil || !strings.HasPrefix(err.Error(), c.code+": ") {
				t.Fatalf("spwn up = %v, want the refusal coded %s", err, c.code)
			}
		})
	}
}

func TestDownStopsOnASpwnYamlThatDoesNotLoad(t *testing.T) {
	for name, c := range unloadedManifests {
		t.Run(name, func(t *testing.T) {
			inUnloadedProject(t, c.manifest)
			err := composeDownRunE(&cobra.Command{}, nil)
			if err == nil || !strings.HasPrefix(err.Error(), c.code+": ") {
				t.Fatalf("spwn down = %v, want the refusal coded %s", err, c.code)
			}
		})
	}
}
