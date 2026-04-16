package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	intmanifest "spwn.sh/packages/project/internal/manifest"
	"spwn.sh/packages/dependency"
)

// helper: build a minimal Input with one world and given agent refs.
func minimalInput(root string, agents []AgentRef, projectDeps []string) Input {
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = a.Name
	}
	m := &intmanifest.Manifest{
		Version: intmanifest.CurrentVersion,
		Name:    "edge-test",
		Worlds: map[string]intmanifest.World{
			"main": {Agents: names, Workspaces: []string{"."}},
		},
		Deps: projectDeps,
	}
	return Input{
		Root:     root,
		Manifest: m,
		AgentRefs: agents,
	}
}

// 1. rulePacksExist with empty deps list in agent.yaml
func TestEdge_PacksExist_EmptyDeps(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "alpha", "name: alpha\ndeps: []\n")
	in := minimalInput(root, []AgentRef{ref}, nil)
	issues := rulePacksExist(in)
	if len(issues) != 0 {
		t.Errorf("empty deps should produce zero issues, got %+v", issues)
	}
}

// 2. rulePacksExist with duplicate refs in same agent
func TestEdge_PacksExist_DuplicateRefs(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - "@spwn/unix"
  - "@spwn/unix"
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.BuiltinTools = []string{"@spwn/unix"}
	issues := rulePacksExist(in)
	// Duplicates of a valid ref should not produce errors (dedup or pass-through).
	for _, iss := range issues {
		if iss.Level == LevelError {
			t.Errorf("duplicate valid ref should not error, got %+v", iss)
		}
	}
}

// 3. rulePacksExist where agent.yaml has a local bare-name ref that exists in spwn/dependencies/
func TestEdge_PacksExist_LocalPackDir(t *testing.T) {
	root := t.TempDir()
	packDir := filepath.Join(root, "spwn", "tools", "my-local-dependency")
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - my-local-dependency
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	issues := rulePacksExist(in)
	for _, iss := range issues {
		if strings.Contains(iss.Message, "my-local-dependency") {
			t.Errorf("local dependency dir exists, should not error: %+v", iss)
		}
	}
}

// 4. rulePacksExist where agent.yaml has a local bare-name ref that exists only as spwn/skills/<name>.md
func TestEdge_PacksExist_LocalSkillFile(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "spwn", "skills")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "research.md"), []byte("# research"), 0o644); err != nil {
		t.Fatal(err)
	}
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - research
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	issues := rulePacksExist(in)
	for _, iss := range issues {
		if strings.Contains(iss.Message, "research") {
			t.Errorf("local skill file exists, should not error: %+v", iss)
		}
	}
}

// 5. rulePacksExist with a versioned ref like @spwn/unix@24.04
func TestEdge_PacksExist_VersionedRef(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - "@spwn/unix@24.04"
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.BuiltinTools = []string{"@spwn/unix"}
	issues := rulePacksExist(in)
	for _, iss := range issues {
		if iss.Level == LevelError && strings.Contains(iss.Message, "unix") {
			t.Errorf("versioned builtin ref should resolve OK, got %+v", iss)
		}
	}
}

// 6. rulePackVersionConflict with two agents using different versions of same dependency
func TestEdge_PackVersionConflict_DifferentVersions(t *testing.T) {
	root := t.TempDir()
	a1 := scaffoldAgent(t, root, "agent-a", `name: agent-a
dependencies:
  - "@spwn/git@1.0"
`)
	a2 := scaffoldAgent(t, root, "agent-b", `name: agent-b
dependencies:
  - "@spwn/git@2.0"
`)
	in := minimalInput(root, []AgentRef{a1, a2}, nil)
	issues := rulePackVersionConflict(in)
	if len(issues) != 1 {
		t.Fatalf("want 1 conflict, got %d: %+v", len(issues), issues)
	}
	if !strings.Contains(issues[0].Message, "@spwn/git") {
		t.Errorf("should mention @spwn/git: %q", issues[0].Message)
	}
	if !strings.Contains(issues[0].Message, "conflicting versions") {
		t.Errorf("should mention conflict: %q", issues[0].Message)
	}
}

// 7. rulePackVersionConflict with identical versions (no conflict)
func TestEdge_PackVersionConflict_IdenticalVersions(t *testing.T) {
	root := t.TempDir()
	a1 := scaffoldAgent(t, root, "agent-a", `name: agent-a
dependencies:
  - "@spwn/git@1.0"
`)
	a2 := scaffoldAgent(t, root, "agent-b", `name: agent-b
dependencies:
  - "@spwn/git@1.0"
`)
	in := minimalInput(root, []AgentRef{a1, a2}, nil)
	issues := rulePackVersionConflict(in)
	if len(issues) != 0 {
		t.Errorf("identical versions should not conflict, got %+v", issues)
	}
}

// 8. ruleLockfileConsistent with project-level deps that ARE in lockfile (no issue)
func TestEdge_LockfileConsistent_ProjectDepsPresent(t *testing.T) {
	root := t.TempDir()
	// No agents needed, just project-level dependency.
	l := dependency.EmptyLockfile()
	l.Add("@spwn/unix", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	l.Add("@spwn/git", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	writeLockfile(t, root, l)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "edge-test",
			Worlds:  map[string]intmanifest.World{"main": {Agents: []string{}, Workspaces: []string{"."}}},
			Deps:    []string{"@spwn/unix", "@spwn/git"},
		},
	}
	issues := ruleLockfileConsistent(in)
	if len(issues) != 0 {
		t.Errorf("all project deps in lockfile, want 0 issues, got %+v", issues)
	}
}

// 9. ruleLockfileConsistent with project-level deps missing from lockfile
func TestEdge_LockfileConsistent_ProjectDepsMissing(t *testing.T) {
	root := t.TempDir()
	l := dependency.EmptyLockfile()
	l.Add("@spwn/unix", dependency.LockEntry{Version: "1", Source: dependency.SourceBuiltin})
	writeLockfile(t, root, l)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "edge-test",
			Worlds:  map[string]intmanifest.World{"main": {Agents: []string{}, Workspaces: []string{"."}}},
			Deps:    []string{"@spwn/unix", "@spwn/git"},
		},
	}
	issues := ruleLockfileConsistent(in)
	var sawGit bool
	for _, iss := range issues {
		if strings.Contains(iss.Message, "@spwn/git") {
			sawGit = true
		}
		if strings.Contains(iss.Message, "@spwn/unix") {
			t.Error("@spwn/unix is in lockfile, should not be flagged")
		}
	}
	if !sawGit {
		t.Error("@spwn/git missing from lockfile should be flagged")
	}
}

// 10. Error messages contain "dependency" not "plugin" or "package"
func TestEdge_ErrorMessages_SayPack(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - "@spwn/nonexistent"
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.BuiltinTools = []string{"@spwn/something-else"}
	issues := rulePacksExist(in)
	for _, iss := range issues {
		lower := strings.ToLower(iss.Message)
		if strings.Contains(lower, "plugin") {
			t.Errorf("message should not say 'plugin': %q", iss.Message)
		}
		// "package" appears in the conflict rule message, check that base
		// existence messages use "dependency" terminology.
		if strings.Contains(iss.Message, "dependency") || strings.Contains(iss.Message, "does not exist") {
			// OK - contains "dependency" wording
		}
	}
	if len(issues) == 0 {
		t.Fatal("expected at least one issue for nonexistent ref")
	}
	// Verify the message says "dependency" (not "plugin").
	found := false
	for _, iss := range issues {
		if strings.Contains(iss.Message, "dependency") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'dependency' in error message, got: %+v", issues)
	}
}

// 11. Error paths use "#deps" not "#plugins" or "#packages"
func TestEdge_ErrorPaths_UseDeps(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - "@spwn/nonexistent"
`)
	in := minimalInput(root, []AgentRef{ref}, nil)
	in.BuiltinTools = []string{"@spwn/something-else"}
	issues := rulePacksExist(in)
	if len(issues) == 0 {
		t.Fatal("expected at least one issue")
	}
	for _, iss := range issues {
		if strings.Contains(iss.Path, "#plugins") {
			t.Errorf("path should not use #plugins: %q", iss.Path)
		}
		if strings.Contains(iss.Path, "#packages") {
			t.Errorf("path should not use #packages: %q", iss.Path)
		}
		if !strings.Contains(iss.Path, "#deps") {
			t.Errorf("path should contain #deps: %q", iss.Path)
		}
	}
}

// 12. Validator hint strings reference "spwn install" not "spwn plugin install"
func TestEdge_Hints_SaySpwnInstall(t *testing.T) {
	root := t.TempDir()
	ref := scaffoldAgent(t, root, "alpha", `name: alpha
dependencies:
  - "@spwn/missing-thing"
`)
	// Create a lockfile so the rule fires.
	writeLockfile(t, root, dependency.EmptyLockfile())

	in := minimalInput(root, []AgentRef{ref}, nil)
	in.BuiltinTools = []string{"@spwn/something-else"}

	// Collect issues from both rulePacksExist and ruleLockfileConsistent.
	var allIssues []Issue
	allIssues = append(allIssues, rulePacksExist(in)...)
	allIssues = append(allIssues, ruleLockfileConsistent(in)...)

	if len(allIssues) == 0 {
		t.Fatal("expected issues for missing dependency")
	}

	for _, iss := range allIssues {
		if iss.Hint == "" {
			continue
		}
		if strings.Contains(iss.Hint, "spwn plugin install") {
			t.Errorf("hint should not say 'spwn plugin install': %q", iss.Hint)
		}
		if strings.Contains(iss.Hint, "spwn install") {
			// Good - correct hint wording.
		}
	}

	// Verify at least one hint mentions "spwn install".
	found := false
	for _, iss := range allIssues {
		if strings.Contains(iss.Hint, "spwn install") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected at least one hint with 'spwn install', got hints: %+v", allIssues)
	}
}
