package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_createsManifestAndLayout(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir, InitOpts{Name: "acme-api"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	required := []string{
		"spwn.yaml",
		"spwn/worlds/default.yaml",
		"spwn/agents/default/agent.yaml",
		"spwn/agents/default/CLAUDE.md",
		"spwn/agents/default/core/profile.md",
		"spwn/agents/default/skills/.gitkeep",
		"spwn/agents/default/knowledge/.gitkeep",
		"spwn/agents/default/playbooks/.gitkeep",
		"spwn/agents/default/journal/.gitkeep",
		".spwn/state.json",
		".gitignore",
	}
	for _, rel := range required {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s after Init: %v", rel, err)
		}
	}

	gitignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), ".spwn/") {
		t.Errorf("expected .gitignore to contain .spwn/, got: %q", string(gitignore))
	}

	manifest, err := os.ReadFile(filepath.Join(dir, "spwn.yaml"))
	if err != nil {
		t.Fatalf("read spwn.yaml: %v", err)
	}
	if !strings.Contains(string(manifest), "name: acme-api") {
		t.Errorf("expected name: acme-api in spwn.yaml, got:\n%s", manifest)
	}
}

func TestInit_refusesOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{}); err != nil {
		t.Fatalf("first Init: %v", err)
	}
	if err := Init(dir, InitOpts{}); err == nil {
		t.Fatal("expected second Init without Force to error, got nil")
	}
	if err := Init(dir, InitOpts{Force: true}); err != nil {
		t.Fatalf("Force Init: %v", err)
	}
}

func TestFind_walksUpFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	if err := Init(root, InitOpts{Name: "walk-test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	deep := filepath.Join(root, "src", "nested", "path")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	p, err := Find(deep)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if p == nil {
		t.Fatal("expected to find a project, got nil")
	}
	// t.TempDir can return a path with a /private/ prefix on macOS,
	// so compare via EvalSymlinks to normalize both sides.
	gotRoot, _ := filepath.EvalSymlinks(p.Root)
	wantRoot, _ := filepath.EvalSymlinks(root)
	if gotRoot != wantRoot {
		t.Errorf("root = %q, want %q", gotRoot, wantRoot)
	}
	if p.Manifest.Name != "walk-test" {
		t.Errorf("Manifest.Name = %q, want %q", p.Manifest.Name, "walk-test")
	}
}

func TestFind_returnsNilWhenNoManifest(t *testing.T) {
	dir := t.TempDir()
	p, err := Find(dir)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if p != nil {
		t.Fatalf("expected nil project in empty dir, got %+v", p)
	}
}

func TestLoad_resolvesRefs(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "resolve-test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	p, err := Load(filepath.Join(dir, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Agents) != 1 || p.Agents[0].Name != "default" {
		t.Fatalf("Agents = %+v, want [default]", p.Agents)
	}
	if !p.Agents[0].Exists {
		t.Errorf("default agent should exist after Init")
	}
	if p.World.Name != "default" {
		t.Errorf("World.Name = %q, want default", p.World.Name)
	}
	if !p.World.Exists {
		t.Errorf("default world config should exist after Init")
	}
}

func TestValidate_cleanProjectHasNoErrors(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "valid"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	p, err := Load(filepath.Join(dir, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	issues := Validate(p)
	if HasErrors(issues) {
		t.Errorf("clean project should have no errors, got: %+v", issues)
	}
}

func TestValidate_missingAgentDirIsError(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "missing-agent-test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Remove the default agent dir to simulate a broken project.
	if err := os.RemoveAll(filepath.Join(dir, "spwn", "agents", "default")); err != nil {
		t.Fatalf("rm agent dir: %v", err)
	}
	p, err := Load(filepath.Join(dir, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	issues := Validate(p)
	if !HasErrors(issues) {
		t.Fatalf("expected error issue for missing agent dir, got: %+v", issues)
	}
}

func TestBuild_flattensProjectIntoArtifact(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "build-test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	p, err := Load(filepath.Join(dir, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	result, err := Build(p)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil BuildResult")
	}

	buildDir := filepath.Join(dir, ".spwn", "build")
	expected := []string{
		"build.json",
		"manifest.json",
		"worlds/default.yaml",
		"agents/default/agent.yaml",
		"agents/default/CLAUDE.md",
		"agents/default/core/profile.md",
	}
	for _, rel := range expected {
		if _, err := os.Stat(filepath.Join(buildDir, rel)); err != nil {
			t.Errorf("missing %s in build artifact: %v", rel, err)
		}
	}

	meta, err := LoadBuildMetadata(p)
	if err != nil {
		t.Fatalf("LoadBuildMetadata: %v", err)
	}
	if meta == nil {
		t.Fatal("expected build metadata, got nil")
	}
	if meta.Project != "build-test" {
		t.Errorf("meta.Project = %q, want build-test", meta.Project)
	}
	if len(meta.Agents) != 1 || meta.Agents[0] != "default" {
		t.Errorf("meta.Agents = %v, want [default]", meta.Agents)
	}
	if meta.ContentHash == "" {
		t.Error("meta.ContentHash should be set")
	}
}

func TestBuild_repeatedCallsProduceSameHash(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "deterministic"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	p, _ := Load(filepath.Join(dir, "spwn.yaml"))

	r1, err := Build(p)
	if err != nil {
		t.Fatalf("Build 1: %v", err)
	}
	m1, _ := LoadBuildMetadata(p)

	r2, err := Build(p)
	if err != nil {
		t.Fatalf("Build 2: %v", err)
	}
	m2, _ := LoadBuildMetadata(p)

	if m1.ContentHash != m2.ContentHash {
		t.Errorf("build hash changed across runs: %s vs %s", m1.ContentHash, m2.ContentHash)
	}
	if r1.FileCount != r2.FileCount {
		t.Errorf("file count changed: %d vs %d", r1.FileCount, r2.FileCount)
	}
}
