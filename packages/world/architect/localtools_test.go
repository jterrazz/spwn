package architect

import (
	"os"
	"path/filepath"
	"testing"

	ib "spwn.sh/packages/image"
)

func writePack(t *testing.T, root, name, yaml string) {
	t.Helper()
	dir := filepath.Join(root, "spwn", "packs", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLocalPackage_happyPath(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "my-tool", `name: my-tool
version: "1.2.3"
install:
  packages:
    - build-essential
  commands:
    - echo hi
  user-commands:
    - echo user
  env:
    FOO: bar
verify:
  - command -v my-tool
dependencies:
  - "@spwn/unix"
`)

	tool, err := loadLocalPack(root, "my-tool")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := tool.Name(); got != "local:my-tool" {
		t.Errorf("name: want local:my-tool, got %q", got)
	}
	if got := tool.Version(); got != "1.2.3" {
		t.Errorf("version: want 1.2.3, got %q", got)
	}
	if got := tool.Kind(); got != ib.KindTool {
		t.Errorf("kind: want Tool, got %v", got)
	}
	spec := tool.Install()
	if len(spec.AptPackages) != 1 || spec.AptPackages[0] != "build-essential" {
		t.Errorf("packages: %v", spec.AptPackages)
	}
	if len(spec.Commands) != 1 || spec.Commands[0] != "echo hi" {
		t.Errorf("commands: %v", spec.Commands)
	}
	if spec.Env["FOO"] != "bar" {
		t.Errorf("env: %v", spec.Env)
	}
	if got := tool.Verify(); len(got) != 1 || got[0] != "command -v my-tool" {
		t.Errorf("verify: %v", got)
	}
	if got := tool.Dependencies(); len(got) != 1 || got[0] != "@spwn/unix" {
		t.Errorf("deps: %v", got)
	}
}

func TestLoadLocalPackage_missingManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "packs", "empty-pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := loadLocalPack(root, "empty-pkg")
	if err == nil {
		t.Fatal("want error for missing plugin.yaml")
	}
}

func TestLoadLocalPackage_skillsDirExposed(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "toolish", `name: toolish
install:
  packages: [curl]
verify:
  - command -v curl
`)
	skillsDir := filepath.Join(root, "spwn", "packs", "toolish", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool, err := loadLocalPack(root, "toolish")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if tool.Skills() == nil {
		t.Error("want non-nil Skills fs for package with skills/ dir")
	}
}

func TestHydrateLocalPackages_passThroughAtRefs(t *testing.T) {
	root := t.TempDir()
	reg := ib.NewRegistry()

	list := []string{"@spwn/unix", "@spwn/git"}
	got, err := hydrateLocalPacks(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 2 || got[0] != "@spwn/unix" || got[1] != "@spwn/git" {
		t.Errorf("passthrough mangled: %v", got)
	}
}

func TestHydrateLocalPackages_rewritesBareNames(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "mine", `name: mine
install:
  packages: [curl]
verify:
  - command -v curl
`)
	reg := ib.NewRegistry()

	list := []string{"@spwn/unix", "mine", "@spwn/git"}
	got, err := hydrateLocalPacks(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 3 || got[1] != "local:mine" {
		t.Errorf("want mine -> local:mine, got %v", got)
	}
}

// TestHydrateLocalPackages_mixedListOrderPreserved locks in that the
// rewritten list preserves the original ordering and deduplicates
// bare names after the first registration. The registry only sees
// Register once per unique name.
func TestHydrateLocalPackages_mixedListOrderPreserved(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "tool-a", `name: tool-a
install:
  packages: [curl]
verify:
  - command -v curl
`)
	writePack(t, root, "tool-b", `name: tool-b
install:
  packages: [jq]
verify:
  - command -v jq
`)
	reg := ib.NewRegistry()

	list := []string{"@spwn/unix", "tool-a", "@spwn/git", "tool-b", "tool-a"}
	got, err := hydrateLocalPacks(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	want := []string{"@spwn/unix", "local:tool-a", "@spwn/git", "local:tool-b", "local:tool-a"}
	if len(got) != len(want) {
		t.Fatalf("length: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("at %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHydrateLocalPackages_missingPackageErrors(t *testing.T) {
	root := t.TempDir()
	reg := ib.NewRegistry()

	_, err := hydrateLocalPacks(reg, root, []string{"nonexistent"})
	if err == nil {
		t.Fatal("want error for missing local pack dir")
	}
}

func TestHydrateLocalPackages_idempotentOnDuplicate(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "mine", `name: mine
install:
  packages: [curl]
verify:
  - command -v curl
`)
	reg := ib.NewRegistry()

	_, err := hydrateLocalPacks(reg, root, []string{"mine", "mine"})
	if err != nil {
		t.Fatalf("duplicate should not error: %v", err)
	}
}
