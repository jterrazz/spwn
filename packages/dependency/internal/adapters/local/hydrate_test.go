package local

import (
	compile "spwn.sh/packages/compile"
	"spwn.sh/packages/dependency/tool"
	"os"
	"path/filepath"
	"testing"

	)

func writePack(t *testing.T, root, name, yaml string) {
	t.Helper()
	dir := filepath.Join(root, "spwn", "tools", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tool.yaml"), []byte(yaml), 0o644); err != nil {
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
  - "spwn:unix"
`)

	tl, err := LoadTool(root, "my-tool")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := tl.Name(); got != "local:my-tool" {
		t.Errorf("name: want local:my-tool, got %q", got)
	}
	if got := tl.Version(); got != "1.2.3" {
		t.Errorf("version: want 1.2.3, got %q", got)
	}
	if got := tl.Kind(); got != tool.KindTool {
		t.Errorf("kind: want Tool, got %v", got)
	}
	spec := tl.Install()
	if len(spec.AptPackages) != 1 || spec.AptPackages[0] != "build-essential" {
		t.Errorf("packages: %v", spec.AptPackages)
	}
	if len(spec.Commands) != 1 || spec.Commands[0] != "echo hi" {
		t.Errorf("commands: %v", spec.Commands)
	}
	if spec.Env["FOO"] != "bar" {
		t.Errorf("env: %v", spec.Env)
	}
	if got := tl.Verify(); len(got) != 1 || got[0] != "command -v my-tool" {
		t.Errorf("verify: %v", got)
	}
	if got := tl.Dependencies(); len(got) != 1 || got[0] != "spwn:unix" {
		t.Errorf("deps: %v", got)
	}
}

func TestLoadLocalPackage_missingManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "tools", "empty-pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := LoadTool(root, "empty-pkg")
	if err == nil {
		t.Fatal("want error for missing tool.yaml")
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
	skillsDir := filepath.Join(root, "spwn", "tools", "toolish", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	tl, err := LoadTool(root, "toolish")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if tl.Skills() == nil {
		t.Error("want non-nil Skills fs for package with skills/ dir")
	}
}

func TestHydrateLocalPackages_passThroughAtRefs(t *testing.T) {
	root := t.TempDir()
	reg := compile.NewRegistry()

	list := []string{"spwn:unix", "spwn:git"}
	got, err := Hydrate(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 2 || got[0] != "spwn:unix" || got[1] != "spwn:git" {
		t.Errorf("passthrough mangled: %v", got)
	}
}

func TestHydrateLocalPackages_rewritesToolRefs(t *testing.T) {
	root := t.TempDir()
	writePack(t, root, "mine", `name: mine
install:
  packages: [curl]
verify:
  - command -v curl
`)
	reg := compile.NewRegistry()

	list := []string{"spwn:unix", "tool:mine", "spwn:git"}
	got, err := Hydrate(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 3 || got[1] != "local:mine" {
		t.Errorf("want tool:mine -> local:mine, got %v", got)
	}
}

// TestHydrateLocalPackages_mixedListOrderPreserved locks in that the
// rewritten list preserves the original ordering and deduplicates
// tool: refs after the first registration. The registry only sees
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
	reg := compile.NewRegistry()

	list := []string{"spwn:unix", "tool:tool-a", "spwn:git", "tool:tool-b", "tool:tool-a"}
	got, err := Hydrate(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	want := []string{"spwn:unix", "local:tool-a", "spwn:git", "local:tool-b", "local:tool-a"}
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
	reg := compile.NewRegistry()

	_, err := Hydrate(reg, root, []string{"tool:nonexistent"})
	if err == nil {
		t.Fatal("want error for missing local dependency dir")
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
	reg := compile.NewRegistry()

	_, err := Hydrate(reg, root, []string{"tool:mine", "tool:mine"})
	if err != nil {
		t.Fatalf("duplicate should not error: %v", err)
	}
}

// TestHydrateLocalPackages_bareRefPassesThrough: bare refs (invalid
// under the new grammar) are left as-is so the validator/resolver
// downstream surfaces the scheme-grammar error rather than crashing
// on an empty filesystem lookup.
func TestHydrateLocalPackages_bareRefPassesThrough(t *testing.T) {
	root := t.TempDir()
	reg := compile.NewRegistry()

	got, err := Hydrate(reg, root, []string{"bare-name"})
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 1 || got[0] != "bare-name" {
		t.Errorf("want bare ref passthrough, got %v", got)
	}
}
