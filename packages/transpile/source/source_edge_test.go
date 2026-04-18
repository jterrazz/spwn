package source

import (
	"os"
	"path/filepath"
	"testing"
)

// scaffoldProject creates a minimal spwn project in a temp dir and returns
// the project root path. The spwn.yaml content is provided by the caller.
func scaffoldProject(t *testing.T, manifest string, agents map[string]string) string {
	t.Helper()
	root := t.TempDir()

	// Write spwn.yaml.
	if err := os.WriteFile(filepath.Join(root, "spwn.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create agent directories with agent.yaml.
	for name, agentYAML := range agents {
		dir := filepath.Join(root, "spwn", "agents", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if agentYAML != "" {
			if err := os.WriteFile(filepath.Join(dir, "agent.yaml"), []byte(agentYAML), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	return root
}

// ── ToCompileInput merges project-level deps with agent-level deps ──────────

func TestToCompileInput_MergesProjectAndAgentDeps(t *testing.T) {
	manifest := `version: 2
name: merge-test
dependencies:
  - "spwn:unix"
  - shared-tool
worlds:
  home:
    agents: [neo]
    workspaces: [.]
`
	agentYAML := `name: neo
role: worker
dependencies:
  - "spwn:git"
  - agent-only-tool
`
	root := scaffoldProject(t, manifest, map[string]string{"neo": agentYAML})

	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	in, err := ToCompileInput(src, "home")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}

	// The merged set should contain both project-level and agent-level dependency.
	depsSet := map[string]bool{}
	for _, d := range in.Deps {
		depsSet[d] = true
	}

	for _, want := range []string{"spwn:unix", "shared-tool", "spwn:git", "agent-only-tool"} {
		if !depsSet[want] {
			t.Errorf("missing expected dep %q in merged list: %v", want, in.Deps)
		}
	}

	// VerifiedTools should match Manifest.Deps.
	if !equalStrings(in.VerifiedTools, in.Deps) {
		t.Errorf("VerifiedTools = %v, Manifest.Deps = %v — should match", in.VerifiedTools, in.Deps)
	}
}

// ── Agent with no deps but project has deps → agent inherits them ───────────

func TestToCompileInput_AgentInheritsProjectDeps(t *testing.T) {
	manifest := `version: 2
name: inherit-test
dependencies:
  - "spwn:unix"
  - "spwn:git"
worlds:
  home:
    agents: [bare]
    workspaces: [.]
`
	// Agent has no deps at all.
	agentYAML := `name: bare
role: worker
`
	root := scaffoldProject(t, manifest, map[string]string{"bare": agentYAML})

	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	in, err := ToCompileInput(src, "home")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}

	// Agent has 0 deps, project has 2 — the compile input should have
	// the project-level dependency.
	want := []string{"spwn:git", "spwn:unix"}
	if !equalStrings(in.Deps, want) {
		t.Errorf("Manifest.Deps = %v, want %v", in.Deps, want)
	}
}

// ── Agent with no agent.yaml at all → project deps still present ────────────

func TestToCompileInput_NoAgentYAML(t *testing.T) {
	manifest := `version: 2
name: noconfig
dependencies:
  - "spwn:python"
worlds:
  home:
    agents: [ghost]
    workspaces: [.]
`
	// Pass empty string for agent YAML — no agent.yaml will be written.
	root := scaffoldProject(t, manifest, map[string]string{"ghost": ""})

	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	in, err := ToCompileInput(src, "home")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}

	if len(in.Deps) != 1 || in.Deps[0] != "spwn:python" {
		t.Errorf("Manifest.Deps = %v, want [spwn:python]", in.Deps)
	}
}

// ── Duplicate deps across project and agent are deduplicated ────────────────

func TestToCompileInput_DeduplicatesDeps(t *testing.T) {
	manifest := `version: 2
name: dedup-test
dependencies:
  - "spwn:unix"
  - "spwn:git"
worlds:
  home:
    agents: [neo]
    workspaces: [.]
`
	agentYAML := `name: neo
role: worker
dependencies:
  - "spwn:unix"
  - "spwn:git"
  - extra
`
	root := scaffoldProject(t, manifest, map[string]string{"neo": agentYAML})

	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	in, err := ToCompileInput(src, "home")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}

	// Should be exactly 3 unique deps, not 5.
	want := []string{"extra", "spwn:git", "spwn:unix"}
	if !equalStrings(in.Deps, want) {
		t.Errorf("Manifest.Deps = %v, want %v (deduplicated)", in.Deps, want)
	}
}

// ── No project deps, no agent deps → empty list ────────────────────────────

func TestToCompileInput_NoDepsAnywhere(t *testing.T) {
	manifest := `version: 2
name: bare-project
worlds:
  home:
    agents: [a]
    workspaces: [.]
`
	agentYAML := `name: a
role: worker
`
	root := scaffoldProject(t, manifest, map[string]string{"a": agentYAML})

	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	in, err := ToCompileInput(src, "home")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}

	if len(in.Deps) != 0 {
		t.Errorf("expected 0 deps, got %v", in.Deps)
	}
}

// ── Multi-agent world merges deps from all agents + project ─────────────────

func TestToCompileInput_MultiAgentMerge(t *testing.T) {
	manifest := `version: 2
name: multi
dependencies:
  - shared
worlds:
  team:
    agents: [alpha, beta]
    workspaces: [.]
`
	agents := map[string]string{
		"alpha": "name: alpha\nrole: worker\ndependencies:\n  - tool-a\n",
		"beta":  "name: beta\nrole: worker\ndependencies:\n  - tool-b\n",
	}
	root := scaffoldProject(t, manifest, agents)

	src, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	in, err := ToCompileInput(src, "team")
	if err != nil {
		t.Fatalf("ToCompileInput: %v", err)
	}

	depsSet := map[string]bool{}
	for _, d := range in.Deps {
		depsSet[d] = true
	}

	for _, want := range []string{"shared", "tool-a", "tool-b"} {
		if !depsSet[want] {
			t.Errorf("missing %q in merged deps: %v", want, in.Deps)
		}
	}
	if len(in.Deps) != 3 {
		t.Errorf("expected 3 deps, got %d: %v", len(in.Deps), in.Deps)
	}
}
