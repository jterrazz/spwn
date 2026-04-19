package project

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
		"spwn.lock",
		"spwn/agents/neo/agent.yaml",
		"spwn/agents/neo/AGENTS.md",
		"spwn/agents/neo/SOUL.md",
		"spwn/agents/neo/playbooks/.gitkeep",
		"spwn/agents/neo/journal/.gitkeep",
		"knowledge/.gitkeep",
		".gitignore",
		// One example per local-ref scheme so the scaffold demonstrates
		// skill: / tool: / hook: authoring patterns end-to-end.
		"spwn/skills/focus.md",
		"spwn/tools/greet/tool.yaml",
		"spwn/hooks/pre-spawn.sh",
	}
	for _, rel := range required {
		path := filepath.Join(dir, rel)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing %s after Init: %v", rel, err)
		}
	}

	// Agent no longer owns a knowledge layer — it moved to the world.
	if _, err := os.Stat(filepath.Join(dir, "spwn", "agents", "neo", "knowledge")); err == nil {
		t.Errorf("spwn/agents/neo/knowledge/ should not exist (knowledge moved to world scope)")
	}

	// The old per-world nested location should be gone — knowledge now
	// lives flat at the project root by convention.
	if _, err := os.Stat(filepath.Join(dir, "spwn", "worlds")); err == nil {
		t.Errorf("spwn/worlds/ should not exist (retired in favour of explicit manifest paths)")
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
	if !strings.Contains(string(manifest), "worlds:") {
		t.Errorf("expected worlds: map in spwn.yaml, got:\n%s", manifest)
	}

	// Lockfile should be seeded with the three default spwn:* refs
	// so `spwn check` passes on a brand-new project (no drift between
	// the scaffolded agent.yaml and the initial lockfile).
	lock, err := os.ReadFile(filepath.Join(dir, "spwn.lock"))
	if err != nil {
		t.Fatalf("read spwn.lock: %v", err)
	}
	lockStr := string(lock)
	for _, ref := range []string{"spwn:unix", "spwn:git", "spwn:python"} {
		if !strings.Contains(lockStr, ref) {
			t.Errorf("lockfile missing %s:\n%s", ref, lockStr)
		}
	}
}

// TestInit_scaffoldsLocalRefExamples locks in the end-to-end story:
// the scaffold materialises one concrete example for each local-ref
// scheme (skill:/tool:/hook:), wires all three into the default
// agent's dependencies list, and makes the hook executable so the
// spawn pipeline can actually launch it. Without this the "it just
// works on first init" promise slips.
func TestInit_scaffoldsLocalRefExamples(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "local-refs"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	agentYAML, err := os.ReadFile(filepath.Join(dir, "spwn/agents/neo/agent.yaml"))
	if err != nil {
		t.Fatalf("read agent.yaml: %v", err)
	}
	for _, ref := range []string{"skill:focus", "tool:greet", "hook:pre-spawn"} {
		if !strings.Contains(string(agentYAML), ref) {
			t.Errorf("agent.yaml missing %q, got:\n%s", ref, agentYAML)
		}
	}

	// Hook must be executable — runtime launches it directly.
	info, err := os.Stat(filepath.Join(dir, "spwn/hooks/pre-spawn.sh"))
	if err != nil {
		t.Fatalf("stat hook: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("hook pre-spawn.sh is not executable, mode=%v", info.Mode())
	}

	// Skill must be a valid frontmatter-first markdown block: starts
	// with --- so the skill loader picks up its name/description.
	skill, err := os.ReadFile(filepath.Join(dir, "spwn/skills/focus.md"))
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	if !strings.HasPrefix(string(skill), "---\n") {
		t.Errorf("skill focus.md should start with YAML frontmatter, got:\n%s", skill)
	}

	// Tool must declare a name and a verify command — the two fields
	// the dep resolver cares about. No YAML parse needed; substring
	// checks are enough to catch regressions in the template.
	tool, err := os.ReadFile(filepath.Join(dir, "spwn/tools/greet/tool.yaml"))
	if err != nil {
		t.Fatalf("read tool: %v", err)
	}
	toolStr := string(tool)
	if !strings.Contains(toolStr, "name: greet") {
		t.Errorf("tool.yaml missing `name: greet`:\n%s", toolStr)
	}
	if !strings.Contains(toolStr, "verify:") {
		t.Errorf("tool.yaml missing verify block:\n%s", toolStr)
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

func TestLoad_resolvesAgents(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "resolve-test"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	p, err := Load(filepath.Join(dir, "spwn.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(p.Agents) != 1 || p.Agents[0].Name != "neo" {
		t.Fatalf("Agents = %+v, want [neo]", p.Agents)
	}
	if !p.Agents[0].Exists {
		t.Errorf("neo agent should exist after Init")
	}
	if _, ok := p.Manifest.Worlds["neo"]; !ok {
		t.Errorf("expected worlds['neo'] entry after Init")
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
	if err := os.RemoveAll(filepath.Join(dir, "spwn", "agents", "neo")); err != nil {
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

func TestValidate_oneAgentOneWorld(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "shared"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	// Append a second world that also references neo.
	manifestPath := filepath.Join(dir, "spwn.yaml")
	data, _ := os.ReadFile(manifestPath)
	tail := "\n  matrix:\n    agents: [neo]\n    workspaces: [.]\n"
	if err := os.WriteFile(manifestPath, append(data, []byte(tail)...), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	issues := Validate(p)
	found := false
	for _, i := range issues {
		if i.Level == LevelError && strings.Contains(i.Message, "already deployed") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected one-agent-one-world error, got: %+v", issues)
	}
}

func TestValidate_workspaceMountRules(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "wsm"}); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "spwn.yaml")

	// Two bare entries are fine under the unified form — each gets
	// auto-named workspace0, workspace1 at spawn time. A container-
	// path-on-RHS entry (legacy manifest form) is rejected because
	// users should never write container platform.
	body := `version: 1
name: wsm
worlds:
  neo:
    agents: [neo]
    workspaces:
      - .
      - ./data:/workspace/data
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, _ := Load(manifestPath)
	issues := Validate(p)
	found := false
	for _, i := range issues {
		if i.Level == LevelError && strings.Contains(i.Message, "container-path form") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected container-path-form error, got: %+v", issues)
	}
}

// TestValidate_workspaceBareEntriesOK locks in that multiple bare
// path entries are valid under the unified form.
func TestValidate_workspaceBareEntriesOK(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "bare"}); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "spwn.yaml")
	body := `version: 1
name: bare
worlds:
  neo:
    agents: [neo]
    workspaces:
      - .
      - ./data
      - web=./src
`
	if err := os.WriteFile(manifestPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, _ := Load(manifestPath)
	for _, i := range Validate(p) {
		if i.Level == LevelError && strings.Contains(i.Message, "workspace") {
			t.Errorf("unexpected workspace error: %+v", i)
		}
	}
}

func TestAddAgentToManifest_appendsWorld(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "addtest"}); err != nil {
		t.Fatal(err)
	}
	// Create a second agent dir.
	if err := os.MkdirAll(filepath.Join(dir, "spwn", "agents", "trinity"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(dir, "spwn.yaml")
	if err := AddAgentToManifest(manifestPath, "trinity"); err != nil {
		t.Fatalf("AddAgentToManifest: %v", err)
	}
	data, _ := os.ReadFile(manifestPath)
	if !strings.Contains(string(data), "trinity") {
		t.Errorf("expected trinity entry in manifest, got:\n%s", data)
	}
	// Idempotency: second call should not error or duplicate.
	if err := AddAgentToManifest(manifestPath, "trinity"); err != nil {
		t.Fatalf("idempotent AddAgentToManifest: %v", err)
	}
}
