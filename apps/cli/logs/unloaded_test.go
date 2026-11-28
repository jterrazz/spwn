package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTheWorldFilterStopsOnASpwnYamlThatDoesNotLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SPWN_HOME", filepath.Join(dir, "spwn-home"))
	t.Setenv("HOME", filepath.Join(dir, "home"))
	if err := os.WriteFile(filepath.Join(dir, "spwn.yaml"), []byte("version: 1\nname: site\nfolders:\n  agents: ..\nworlds:\n  ship: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	err := validateWorldFilter("ship")
	if err == nil || !strings.HasPrefix(err.Error(), "spwn-yaml.folders: ") {
		t.Fatalf("the world filter = %v, want the folders: refusal", err)
	}
}
