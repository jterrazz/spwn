package manifest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency/internal/manifest"
	"spwn.sh/packages/dependency/tool"
)

func writeManifest(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spwn.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParse_minimal(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "spwn:git"
kind: tool
install:
  packages: [git]
verify:
  - command -v git
`)

	parsed, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Schema.Name != "spwn:git" {
		t.Errorf("name: want spwn:git, got %q", parsed.Schema.Name)
	}
	if parsed.Kind != tool.KindTool {
		t.Errorf("kind: want Tool, got %v", parsed.Kind)
	}
	spec := parsed.Schema.Install
	if len(spec.AptPackages) != 1 || spec.AptPackages[0] != "git" {
		t.Errorf("packages: %v", spec.AptPackages)
	}
	if got := parsed.Schema.Verify; len(got) != 1 || got[0] != "command -v git" {
		t.Errorf("verify: %v", got)
	}
}

func TestParse_defaults(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `install:
  packages: [curl]
`)

	parsed, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{
		DefaultName:    "local-tool",
		DefaultVersion: "0.0.0-local",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Schema.Name != "local-tool" {
		t.Errorf("default name: want local-tool, got %q", parsed.Schema.Name)
	}
	if parsed.Schema.Version != "0.0.0-local" {
		t.Errorf("default version: want 0.0.0-local, got %q", parsed.Schema.Version)
	}
	if parsed.Kind != tool.KindTool {
		t.Errorf("default kind: want Tool, got %v", parsed.Kind)
	}
}

func TestParse_runtimeKindAndProvider(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "spwn:claude-code"
kind: runtime
version: latest
runtime-provider: claude-code
install:
  commands:
    - curl -fsSL https://claude.ai/install.sh | bash
verify:
  - command -v claude
`)

	parsed, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Kind != tool.KindRuntime {
		t.Errorf("kind: want Runtime, got %v", parsed.Kind)
	}
	if parsed.Schema.RuntimeProvider != "claude-code" {
		t.Errorf("want runtime-provider claude-code")
	}
}

func TestParse_filesBakedIn(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "spwn:architect"
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

	parsed, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	spec := parsed.Schema.Install
	_ = spec
	got, ok := parsed.FileBytes["/usr/local/bin/entrypoint.sh"]
	if !ok {
		t.Fatalf("file not baked in, files=%v", parsed.FileBytes)
	}
	if string(got) != "#!/bin/sh\nexec sleep infinity\n" {
		t.Errorf("file content: %q", string(got))
	}
}

func TestParse_skillsDirExposed(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "spwn:qmd"
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

	parsed, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if func() fs.FS { sf, _ := parsed.SkillsFS.(fs.FS); return sf }() == nil {
		t.Error("want non-nil skills fs")
	}
}

func TestParse_unknownKindErrors(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: bogus
kind: weird-kind
`)
	if _, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{}); err == nil {
		t.Fatal("want error for unknown kind")
	}
}

func TestParse_missingNameErrors(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `install:
  packages: [git]
`)
	if _, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{}); err == nil {
		t.Fatal("want error for missing name + no default")
	}
}

