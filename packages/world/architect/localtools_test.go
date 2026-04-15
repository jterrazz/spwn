package architect

import (
	"os"
	"path/filepath"
	"testing"

	ib "spwn.sh/packages/image"
)

func writeTool(t *testing.T, root, name, yaml string) {
	t.Helper()
	dir := filepath.Join(root, "spwn", "tools", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLocalTool_happyPath(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "my-tool", `name: my-tool
version: "1.2.3"
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

	tool, err := loadLocalTool(root, "my-tool")
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
	if len(spec.Packages) != 1 || spec.Packages[0] != "build-essential" {
		t.Errorf("packages: %v", spec.Packages)
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

func TestLoadLocalTool_missingManifest(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spwn", "tools", "empty-tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := loadLocalTool(root, "empty-tool")
	if err == nil {
		t.Fatal("want error for missing package.yaml")
	}
}

func TestLoadLocalTool_skillsDirExposed(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "toolish", `name: toolish
packages: [curl]
`)
	skillsDir := filepath.Join(root, "spwn", "tools", "toolish", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("# skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool, err := loadLocalTool(root, "toolish")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if tool.Skills() == nil {
		t.Error("want non-nil Skills fs for tool with skills/ dir")
	}
}

func TestHydrateLocalTools_passThroughAtRefs(t *testing.T) {
	root := t.TempDir()
	reg := ib.NewRegistry()

	list := []string{"@spwn/unix", "@spwn/git"}
	got, err := hydrateLocalTools(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 2 || got[0] != "@spwn/unix" || got[1] != "@spwn/git" {
		t.Errorf("passthrough mangled: %v", got)
	}
}

func TestHydrateLocalTools_rewritesBareNames(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "mine", `name: mine
packages: [curl]
`)
	reg := ib.NewRegistry()

	list := []string{"@spwn/unix", "mine", "@spwn/git"}
	got, err := hydrateLocalTools(reg, root, list)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	if len(got) != 3 || got[1] != "local:mine" {
		t.Errorf("want mine -> local:mine, got %v", got)
	}
}

func TestHydrateLocalTools_missingToolErrors(t *testing.T) {
	root := t.TempDir()
	reg := ib.NewRegistry()

	_, err := hydrateLocalTools(reg, root, []string{"nonexistent"})
	if err == nil {
		t.Fatal("want error for missing local tool dir")
	}
}

func TestHydrateLocalTools_idempotentOnDuplicate(t *testing.T) {
	root := t.TempDir()
	writeTool(t, root, "mine", `name: mine`)
	reg := ib.NewRegistry()

	_, err := hydrateLocalTools(reg, root, []string{"mine", "mine"})
	if err != nil {
		t.Fatalf("duplicate should not error: %v", err)
	}
}
