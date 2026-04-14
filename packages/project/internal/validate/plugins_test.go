package validate

import (
	"os"
	"path/filepath"
	"testing"

	intmanifest "spwn.sh/packages/project/internal/manifest"
)

// scaffoldAgent writes a minimal agent directory with the given
// agent.yaml body. Validator rules that load the file will see the
// plugins: list.
func scaffoldAgent(t *testing.T, root, name, yamlBody string) AgentRef {
	t.Helper()
	dir := filepath.Join(root, "spwn", "agents", name)
	if err := os.MkdirAll(filepath.Join(dir, "core"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"CLAUDE.md", "agent.yaml"} {
		if p == "agent.yaml" {
			if err := os.WriteFile(filepath.Join(dir, p), []byte(yamlBody), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, p), []byte("# "+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "core", "profile.md"), []byte("# profile\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return AgentRef{Name: name, Path: dir, Exists: true}
}

func TestRuleToolsExist_PluginsResolveLikeTools(t *testing.T) {
	root := t.TempDir()

	ref := scaffoldAgent(t, root, "neo", `name: neo
plugins:
  - "@spwn/known-plugin"
  - "@spwn/bogus-plugin"
tools:
  - "@spwn/known-tool"
`)

	in := Input{
		Root: root,
		Manifest: &intmanifest.Manifest{
			Version: intmanifest.CurrentVersion,
			Name:    "p",
			Worlds: map[string]intmanifest.World{
				"main": {Agents: []string{"neo"}, Workspaces: []string{"."}},
			},
		},
		AgentRefs:    []AgentRef{ref},
		BuiltinTools: []string{"@spwn/known-tool", "@spwn/known-plugin"},
	}

	issues := ruleToolsExist(in)

	// Expect exactly one error: the bogus plugin.
	var errs []Issue
	for _, is := range issues {
		if is.Level == LevelError {
			errs = append(errs, is)
		}
	}
	if len(errs) != 1 {
		t.Fatalf("want 1 error (bogus plugin), got %d: %+v", len(errs), errs)
	}
	if got := errs[0].Message; got == "" || !contains(got, "@spwn/bogus-plugin") {
		t.Errorf("error message %q should mention @spwn/bogus-plugin", got)
	}
	if got := errs[0].Path; got == "" || !contains(got, "#plugins") {
		t.Errorf("error path %q should mention #plugins", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
