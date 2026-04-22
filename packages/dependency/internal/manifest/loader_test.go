package manifest_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"spwn.sh/packages/dependency/internal/manifest"
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
install:
  packages:
    apt: [git]
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
	spec := parsed.Schema.Install
	if len(spec.Packages.Apt) != 1 || spec.Packages.Apt[0] != "git" {
		t.Errorf("packages.apt: %v", spec.Packages.Apt)
	}
	if got := parsed.Schema.Verify; len(got) != 1 || got[0] != "command -v git" {
		t.Errorf("verify: %v", got)
	}
}

func TestParse_defaults(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `install:
  packages:
    apt: [curl]
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
}

func TestParse_runtimeProvider(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "spwn:claude-code"
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
	if parsed.Schema.RuntimeProvider != "claude-code" {
		t.Errorf("want runtime-provider claude-code")
	}
}

func TestParse_filesBakedIn(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `name: "spwn:architect"
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

func TestParse_missingNameErrors(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `install:
  packages:
    apt: [git]
`)
	if _, err := manifest.Parse(manifest.DirResolver{Root: dir}, manifest.ParseOptions{}); err == nil {
		t.Fatal("want error for missing name + no default")
	}
}
