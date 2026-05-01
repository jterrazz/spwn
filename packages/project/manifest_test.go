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
		"spwn/knowledge/.gitkeep",
		".gitignore",
		// One example per local-ref scheme so the scaffold demonstrates
		// skill/, tool/, hook/, and command/ authoring patterns
		// end-to-end. All four are iso: a path-style ref selected per
		// agent in agent.yaml#dependencies, resolving to one file/dir
		// on disk.
		"spwn/skills/focus.md",
		"spwn/tools/greet/tool.yaml",
		"spwn/hooks/session-banner.yaml",
		"spwn/commands/refactor.md",
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
// scheme (skill/, tool/, hook/) and the agent.yaml subscribes to all
// three. Without this the "it just works on first init" promise slips.
func TestInit_scaffoldsLocalRefExamples(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "local-refs"}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	agentYAML, err := os.ReadFile(filepath.Join(dir, "spwn/agents/neo/agent.yaml"))
	if err != nil {
		t.Fatalf("read agent.yaml: %v", err)
	}
	for _, ref := range []string{"skill/focus", "tool/greet", "hook/session-banner", "command/refactor"} {
		if !strings.Contains(string(agentYAML), ref) {
			t.Errorf("agent.yaml missing %q, got:\n%s", ref, agentYAML)
		}
	}
	// The retired colon-form local schemes (skill:/tool:/hook:)
	// should no longer leak into new projects — a fresh scaffold
	// must use the path-style form exclusively.
	for _, retired := range []string{"skill:", "tool:", "hook:"} {
		if strings.Contains(string(agentYAML), retired) {
			t.Errorf("agent.yaml still references the retired %s scheme:\n%s", retired, agentYAML)
		}
	}

	// spwn/hooks/<name>.yaml carries the runtime-hook examples now —
	// one file per hook, body is just the entry shape.
	hookYAML, err := os.ReadFile(filepath.Join(dir, "spwn/hooks/session-banner.yaml"))
	if err != nil {
		t.Fatalf("stat hooks/session-banner.yaml: %v", err)
	}
	for _, part := range []string{"event: SessionStart", "command:"} {
		if !strings.Contains(string(hookYAML), part) {
			t.Errorf("hooks/session-banner.yaml missing %q, got:\n%s", part, hookYAML)
		}
	}
	// The legacy single-file project hooks.yaml must NOT be created —
	// it's the migration error condition the loader rejects.
	if _, err := os.Stat(filepath.Join(dir, "spwn/hooks.yaml")); err == nil {
		t.Errorf("legacy spwn/hooks.yaml leaked into the scaffold output")
	}

	// spwn/commands/<name>.md is the slash-invoked prompt shape.
	// One file per command; body is the prompt body verbatim.
	cmdMD, err := os.ReadFile(filepath.Join(dir, "spwn/commands/refactor.md"))
	if err != nil {
		t.Fatalf("stat commands/refactor.md: %v", err)
	}
	for _, part := range []string{"name: refactor", "Refactor"} {
		if !strings.Contains(string(cmdMD), part) {
			t.Errorf("commands/refactor.md missing %q, got:\n%s", part, cmdMD)
		}
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

// TestInit_defaultScaffoldOmitsRuntimeBackend pins the 2026-04 decision
// to ship a blank scaffold without pinning runtime.backend — agents
// resolve their backend at spawn time, so `spwn init` should not
// hard-pick one on the user's behalf. Regressions here would bring
// back the opinionated default and silently lock new projects to
// claude-code even when the user is logged into codex.
func TestInit_defaultScaffoldOmitsRuntimeBackend(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "no-backend"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	agentYAML, err := os.ReadFile(filepath.Join(dir, "spwn/agents/neo/agent.yaml"))
	if err != nil {
		t.Fatalf("read agent.yaml: %v", err)
	}
	// Walk non-comment lines: if any line starts with "runtime:" or
	// Has a "backend:" YAML key (leading spaces + backend:), the
	// Scaffold is hard-pinning a backend.
	for _, line := range strings.Split(string(agentYAML), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "runtime:" {
			t.Errorf("scaffold should omit runtime block by default, got:\n%s", agentYAML)
		}
		if strings.HasPrefix(trimmed, "backend:") {
			t.Errorf("scaffold should not hard-pin runtime.backend by default, got:\n%s", agentYAML)
		}
	}
}

// TestInit_backendOptEmitsRuntimeBlock pins the --backend flag
// contract: when the caller pins a backend, the scaffolded agent.yaml
// emits the runtime.backend line verbatim. Accepts the value as-is
// (canonicalisation happens at the CLI layer in resolveInitBackend).
func TestInit_backendOptEmitsRuntimeBlock(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir, InitOpts{Name: "pinned", Backend: "spwn:codex"}); err != nil {
		t.Fatalf("Init: %v", err)
	}
	agentYAML, err := os.ReadFile(filepath.Join(dir, "spwn/agents/neo/agent.yaml"))
	if err != nil {
		t.Fatalf("read agent.yaml: %v", err)
	}
	if !strings.Contains(string(agentYAML), "runtime:") {
		t.Errorf("scaffold should contain runtime block when --backend pinned, got:\n%s", agentYAML)
	}
	if !strings.Contains(string(agentYAML), `backend: "spwn:codex"`) {
		t.Errorf(`scaffold should contain backend: "spwn:codex", got:\n%s`, agentYAML)
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
