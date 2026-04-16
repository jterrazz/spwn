package packyaml_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ib "spwn.sh/packages/image"
	"spwn.sh/packages/image/packyaml"
)

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pack.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParse_minimal(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "@spwn/git"
kind: tool
install:
  packages: [git]
verify:
  - command -v git
`)

	tool, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tool.Name() != "@spwn/git" {
		t.Errorf("name: want @spwn/git, got %q", tool.Name())
	}
	if tool.Kind() != ib.KindTool {
		t.Errorf("kind: want Tool, got %v", tool.Kind())
	}
	spec := tool.Install()
	if len(spec.AptPackages) != 1 || spec.AptPackages[0] != "git" {
		t.Errorf("packages: %v", spec.AptPackages)
	}
	if got := tool.Verify(); len(got) != 1 || got[0] != "command -v git" {
		t.Errorf("verify: %v", got)
	}
}

func TestParse_defaults(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `install:
  packages: [curl]
`)

	tool, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{
		DefaultName:    "local-tool",
		DefaultVersion: "0.0.0-local",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tool.Name() != "local-tool" {
		t.Errorf("default name: want local-tool, got %q", tool.Name())
	}
	if tool.Version() != "0.0.0-local" {
		t.Errorf("default version: want 0.0.0-local, got %q", tool.Version())
	}
	if tool.Kind() != ib.KindTool {
		t.Errorf("default kind: want Tool, got %v", tool.Kind())
	}
}

func TestParse_runtimeKindAndProvider(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "@spwn/claude-code"
kind: runtime
version: latest
runtime-provider: claude-code
install:
  commands:
    - curl -fsSL https://claude.ai/install.sh | bash
verify:
  - command -v claude
`)

	tool, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tool.Kind() != ib.KindRuntime {
		t.Errorf("kind: want Runtime, got %v", tool.Kind())
	}
	if rp, ok := tool.(interface{ RuntimeProvider() string }); !ok || rp.RuntimeProvider() != "claude-code" {
		t.Errorf("want runtime-provider claude-code")
	}
}

func TestParse_filesBakedIn(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "@spwn/architect"
kind: platform
files:
  /usr/local/bin/entrypoint.sh: files/entrypoint.sh
install:
  commands:
    - chmod +x /usr/local/bin/entrypoint.sh
verify:
  - test -x /usr/local/bin/entrypoint.sh
`)
	// Create the source file.
	if err := os.MkdirAll(filepath.Join(dir, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "files", "entrypoint.sh"), []byte("#!/bin/sh\nexec sleep infinity\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	tool, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	spec := tool.Install()
	got, ok := spec.Files["/usr/local/bin/entrypoint.sh"]
	if !ok {
		t.Fatalf("file not baked in, files=%v", spec.Files)
	}
	if string(got) != "#!/bin/sh\nexec sleep infinity\n" {
		t.Errorf("file content: %q", string(got))
	}
}

func TestParse_runtimeConfigSection(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "@spwn/mempalace"
kind: tool
runtime-config:
  runtimes:
    - "@spwn/claude-code"
  configs:
    "@spwn/claude-code":
      mcpServers:
        mempalace:
          command: python3
          args: ["-m", "mempalace.mcp_server"]
install:
  commands:
    - pip install mempalace
verify:
  - command -v mempalace
`)

	tool, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	// The pack: section surfaces via Runtimes() and Config() on the
	// unified image.Tool interface — no type assertion needed.
	runtimes := tool.Runtimes()
	if len(runtimes) != 1 || runtimes[0] != "@spwn/claude-code" {
		t.Errorf("runtimes: %v", runtimes)
	}

	cfg := tool.Config("@spwn/claude-code")
	if len(cfg) == 0 {
		t.Fatal("empty config")
	}
	// Round-trip through JSON to verify shape.
	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	mcp, ok := parsed["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("missing mcpServers: %v", parsed)
	}
	mem, ok := mcp["mempalace"].(map[string]any)
	if !ok {
		t.Fatalf("missing mempalace: %v", mcp)
	}
	if mem["command"] != "python3" {
		t.Errorf("command: %v", mem["command"])
	}

	// Non-matching runtime returns nil.
	if got := tool.Config("@spwn/codex"); got != nil {
		t.Errorf("codex should get nil, got %s", got)
	}
}

func TestParse_skillsDirExposed(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "@spwn/qmd"
install:
  commands: [npm install -g qmd]
verify:
  - command -v qmd
`)
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skills", "SKILL.md"), []byte("# skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	tool, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if tool.Skills() == nil {
		t.Error("want non-nil skills fs")
	}
}

func TestParse_unknownKindErrors(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: bogus
kind: weird-kind
`)
	if _, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{}); err == nil {
		t.Fatal("want error for unknown kind")
	}
}

func TestParse_missingNameErrors(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `install:
  packages: [git]
`)
	if _, err := packyaml.Parse(packyaml.DirResolver{Root: dir}, packyaml.ParseOptions{}); err == nil {
		t.Fatal("want error for missing name + no default")
	}
}
