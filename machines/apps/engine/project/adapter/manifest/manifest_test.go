package manifest_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	spwnproject "spwn.sh/packages/project"
	"spwn.sh/packages/runtimes"
	"spwn.sh/packages/transpile"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/project/contract/contracttest"
)

func open(t *testing.T) contract.Projects {
	t.Setenv("SPWN_HOME", t.TempDir())
	return manifest.New()
}

func TestProjects(t *testing.T) {
	contracttest.Projects(t, func(t *testing.T) (contract.Projects, string) { return open(t), t.TempDir() })
}

func TestANameIsTheManifestsRule(t *testing.T) {
	for _, name := range []string{"site", "2fast", "my-site", "", "Site", "-site", "a/b", "..", "site.md"} {
		if project.ValidName(name) != spwnproject.IsValidProjectName(name) {
			t.Errorf("ValidName(%q) = %v; the manifest's rule says %v", name, project.ValidName(name), spwnproject.IsValidProjectName(name))
		}
	}
}

func made(t *testing.T) (contract.Projects, project.Project) {
	t.Helper()
	p := open(t)
	m, err := p.Create(t.Context(), t.TempDir(), "site", catalog.Harnesses()[0], "")
	if err != nil {
		t.Fatal(err)
	}
	return p, m
}

func TestANewProjectHoldsTheMindLayoutAndNoMore(t *testing.T) {
	_, m := made(t)
	for _, rel := range []string{"spwn/agents/neo/playbooks", "spwn/agents/neo/journal", "spwn/knowledge"} {
		if e, err := (local.Beneath{}).Lstat(m.Root, rel); err != nil || !e.Mode.IsDir() {
			t.Errorf("%s = %v; want the folder", rel, err)
		}
	}
	for _, rel := range []string{"spwn/agents/neo/identity", "spwn/agents/neo/skills", "spwn/agents/neo/knowledge", "knowledge", "spwn/worlds", "spwn/hooks.yaml"} {
		if _, err := (local.Beneath{}).Lstat(m.Root, rel); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s = %v; want it absent", rel, err)
		}
	}
}

func TestABrokenAgentPlansInvalid(t *testing.T) {
	p, m := made(t)
	if err := os.WriteFile(filepath.Join(m.Root, "spwn", "agents", project.Starter, "agent.yaml"), []byte("name: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := p.Plan(t.Context(), m.Root)
	if err != nil || plan.Valid || plan.Summary.Errors == 0 || plan.Issues[0].Level != project.Error {
		t.Fatalf("Plan = %+v, %v; want an error first", plan, err)
	}
}

func TestAMissingPromptIsAnIssueOfTheRender(t *testing.T) {
	for _, c := range []struct {
		name, agent string
		content     *string
	}{
		{"empty", project.Starter, new(string)},
		{"blank", project.Starter, ptr("   \n\t\n")},
		{"no file", project.Starter, nil},
		{"the second of two agents", "trinity", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, m := made(t)
			if c.agent != project.Starter {
				secondAgent(t, m.Root, c.agent)
			}
			content := c.content
			prompt := filepath.Join(m.Root, "spwn", "agents", c.agent, "AGENTS.md")
			if content == nil {
				if err := local.New(nil, nil, nil, nil).FS.Remove(prompt); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(prompt, []byte(*content), 0o600); err != nil {
				t.Fatal(err)
			}
			plan, err := p.Plan(t.Context(), m.Root)
			missing := func(i project.Issue) bool { return strings.Contains(i.Message, "agent prompt is missing or empty") }
			if err != nil || plan.Valid || !slices.ContainsFunc(plan.Issues, func(i project.Issue) bool {
				return i.Source == project.SourceCompile && i.Level == project.Error && missing(i) && strings.Contains(i.Hint, "spwn/agents/"+c.agent+"/AGENTS.md")
			}) {
				t.Fatalf("Plan = %+v, %v; want a render error naming the prompt to write", plan, err)
			}
			for _, i := range plan.Issues {
				if missing(i) && !strings.Contains(i.Hint, "spwn/agents/"+c.agent+"/") {
					t.Errorf("issue %+v names another agent than %s", i, c.agent)
				}
			}
		})
	}
}

func secondAgent(t *testing.T, root, name string) {
	t.Helper()
	from, to := filepath.Join(root, "spwn", "agents", project.Starter), filepath.Join(root, "spwn", "agents", name)
	if err := os.MkdirAll(to, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{"agent.yaml", "SOUL.md", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(from, file))
		if err != nil {
			t.Fatal(err)
		}
		if file == "agent.yaml" {
			data = []byte(strings.Replace(string(data), "\nname: "+project.Starter+"\n", "\nname: "+name+"\n", 1))
		}
		if err = os.WriteFile(filepath.Join(to, file), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	edited(t, filepath.Join(root, project.ManifestName), appended("  "+name+":\n    agents: ["+name+"]\n    workspaces: [.]\n    knowledge: ./spwn/knowledge\n"))
}

func ptr(s string) *string { return &s }

func TestAnAgentIsRenderedByTheHarnessItDeclaresAndByNoOther(t *testing.T) {
	owner := map[string]string{}
	for _, harness := range catalog.Harnesses() {
		p := open(t)
		m, err := p.Create(t.Context(), t.TempDir(), "site", harness, "")
		if err != nil {
			t.Fatal(err)
		}
		files, err := p.Render(t.Context(), m.Root, project.Starter)
		if err != nil || len(files) == 0 {
			t.Fatalf("Render under %s = %d files, %v; want the harness's tree", harness, len(files), err)
		}
		for _, f := range files {
			if other, taken := owner[f.Path]; taken {
				t.Errorf("%s renders %s, which %s renders too; a harness's tree is its own", harness, f.Path, other)
			}
			owner[f.Path] = harness
		}
	}
}

func TestAnUnsupportedBackendPlansInvalidNamingTheSupportedOnes(t *testing.T) {
	p, m := made(t)
	agent := "name: " + project.Starter + "\nruntime:\n  backend: \"spwn:fictional-runtime\"\n"
	if err := os.WriteFile(filepath.Join(m.Root, "spwn", "agents", project.Starter, "agent.yaml"), []byte(agent), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := p.Plan(t.Context(), m.Root)
	if err != nil || plan.Valid {
		t.Fatalf("Plan = %+v, %v; want invalid", plan, err)
	}
	i := slices.IndexFunc(plan.Issues, func(i project.Issue) bool { return strings.Contains(i.Message, "spwn:fictional-runtime") })
	if i < 0 {
		t.Fatalf("Plan = %+v; want an issue naming spwn:fictional-runtime", plan)
	}
	if got := plan.Issues[i]; got.Level != project.Error || !strings.Contains(got.Hint, "claude-code") {
		t.Fatalf("issue = %+v; want an error listing the supported backends", got)
	}
}

func TestEveryHarnessTheManifestAcceptsHasARenderer(t *testing.T) {
	names := slices.Sorted(slices.Values(runtimes.Names()))
	if !slices.Equal(names, transpile.RegisteredRuntimes()) {
		t.Fatalf("runtimes %v, renderers %v; a harness the plan accepts must render", names, transpile.RegisteredRuntimes())
	}
}

func TestAReferenceTheLockfileDoesNotRecordIsDriftNotPlan(t *testing.T) {
	p, m := made(t)
	if err := os.WriteFile(filepath.Join(m.Root, "spwn.lock"), []byte("version: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	drift, err := p.Drift(t.Context(), m.Root)
	if err != nil || len(drift.Rows) == 0 || drift.Rows[0].Source != project.SourceLockfile || !strings.Contains(drift.Rows[0].Message, "spwn.lock") {
		t.Fatalf("Drift = %+v, %v; want a lockfile row", drift, err)
	}
	plan, err := p.Plan(t.Context(), m.Root)
	if err != nil || !plan.Valid {
		t.Fatalf("Plan = %+v, %v; want valid: drift refuses nothing", plan, err)
	}
}

func edited(t *testing.T, path string, edit func(string) string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(edit(string(data))), 0o600); err != nil {
		t.Fatal(err)
	}
}

func without(prefix string) func(string) string {
	return func(s string) string {
		var kept []string
		for _, line := range strings.Split(s, "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), prefix) {
				kept = append(kept, line)
			}
		}
		return strings.Join(kept, "\n")
	}
}

func appended(tail string) func(string) string {
	return func(s string) string { return s + tail }
}

func depending(ref string) func(string) string { return appended("\n  - " + ref + "\n") }

func agentFile(root string) string {
	return filepath.Join(root, "spwn", "agents", project.Starter, "agent.yaml")
}

func skillFile(t *testing.T, root, name string) {
	t.Helper()
	body := "---\nname: " + name + "\ndescription: One-line description of what this skill does.\n---\n\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(root, "spwn", "skills", name+".md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEachRuleOfTheDeclarationIsAnIssueOfThePlan(t *testing.T) {
	manifestFile := func(root string) string { return filepath.Join(root, project.ManifestName) }
	for _, c := range []struct {
		rule          string
		breaks        func(t *testing.T, root string)
		level         project.Level
		message, hint string
	}{
		{"an agent with no description", func(t *testing.T, root string) { edited(t, agentFile(root), without("description:")) },
			project.Error, "is missing a description", ""},
		{"an agent without its SOUL.md", func(t *testing.T, root string) {
			if err := local.New(nil, nil, nil, nil).FS.Remove(filepath.Join(root, "spwn", "agents", project.Starter, "SOUL.md")); err != nil {
				t.Fatal(err)
			}
		}, project.Error, "missing SOUL.md", ""},
		{"one agent in two groupings", func(t *testing.T, root string) {
			edited(t, manifestFile(root), appended("  backup:\n    agents: ["+project.Starter+"]\n    workspaces: [.]\n"))
		}, project.Error, "already deployed", ""},
		{"a grouping with no knowledge path", func(t *testing.T, root string) { edited(t, manifestFile(root), without("knowledge:")) },
			project.Info, "has no knowledge path", "knowledge: ./knowledge"},
		{"a grouping named for a subcommand", func(t *testing.T, root string) {
			edited(t, manifestFile(root), func(s string) string { return strings.Replace(s, "\n  "+project.Starter+":\n", "\n  knowledge:\n", 1) })
		}, project.Error, `world name "knowledge" collides`, ""},
		{"a local skill with no file", func(t *testing.T, root string) { edited(t, agentFile(root), depending("skill/unknown-local")) },
			project.Error, `dependency "skill/unknown-local" does not exist`, "spwn/skills/unknown-local.md"},
		{"a bare reference", func(t *testing.T, root string) {
			skillFile(t, root, "code-review")
			edited(t, agentFile(root), depending("code-review"))
		}, project.Error, `dependency "code-review" is invalid`, "skill/code-review"},
		{"a reference in the retired @spwn/ form", func(t *testing.T, root string) { edited(t, agentFile(root), depending(`"@spwn/python"`)) },
			project.Error, `dependency "@spwn/python" is invalid`, ""},
		{"a reference in the retired local: form", func(t *testing.T, root string) {
			skillFile(t, root, "foo")
			edited(t, agentFile(root), depending(`"local:foo"`))
		}, project.Error, `dependency "local:foo" is invalid`, "skill/foo"},
		{"a reference to a scope of no registry", func(t *testing.T, root string) { edited(t, agentFile(root), depending(`"@acme/foo"`)) },
			project.Error, `dependency "@acme/foo" is invalid`, ""},
		{"a retired tools block", func(t *testing.T, root string) { edited(t, agentFile(root), appended("\ntools:\n  - \"spwn:unix\"\n")) },
			project.Error, `legacy top-level "tools" block`, "flat `dependencies:` list"},
		{"a retired skills block", func(t *testing.T, root string) {
			edited(t, agentFile(root), appended("\nskills:\n  - \"skill/focus\"\n"))
		},
			project.Error, `legacy top-level "skills" block`, "flat `dependencies:` list"},
		{"a retired hooks block", func(t *testing.T, root string) {
			edited(t, agentFile(root), appended("\nhooks:\n  - \"hook/session-banner\"\n"))
		},
			project.Error, `legacy top-level "hooks" block`, "flat `dependencies:` list"},
	} {
		t.Run(c.rule, func(t *testing.T) {
			p, m := made(t)
			c.breaks(t, m.Root)
			plan, err := p.Plan(t.Context(), m.Root)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(plan.Issues, func(i project.Issue) bool {
				return i.Level == c.level && i.Source == project.SourceManifest && strings.Contains(i.Message, c.message) && strings.Contains(i.Hint, c.hint)
			}) {
				t.Fatalf("Plan issues = %+v; want a %s saying %q with a hint naming %q", plan.Issues, c.level, c.message, c.hint)
			}
			if c.level == project.Error && plan.Valid {
				t.Errorf("Plan = %+v; an error must make it invalid", plan)
			}
		})
	}
}

func TestAReferenceThatResolvesPlansValid(t *testing.T) {
	for ref, file := range map[string]string{"skill/reading": "reading", "spwn:mempalace": ""} {
		t.Run(ref, func(t *testing.T) {
			p, m := made(t)
			if file != "" {
				skillFile(t, m.Root, file)
			}
			edited(t, agentFile(m.Root), depending(ref))
			plan, err := p.Plan(t.Context(), m.Root)
			if err != nil || !plan.Valid {
				t.Fatalf("Plan = %+v, %v; want valid", plan, err)
			}
		})
	}
}

func TestReadingAProjectWritesNoHome(t *testing.T) {
	p, m := made(t)
	home, spwnHome := t.TempDir(), filepath.Join(t.TempDir(), "spwn-home")
	t.Setenv("HOME", home)
	t.Setenv("SPWN_HOME", spwnHome)
	if _, err := p.Find(t.Context(), t.TempDir()); !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("Find outside a project = %v; want ErrNotFound", err)
	}
	if _, err := p.Show(t.Context(), m.Root); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Plan(t.Context(), m.Root); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Drift(t.Context(), m.Root); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{spwnHome, filepath.Join(home, ".spwn")} {
		if _, err := local.New(nil, nil, nil, nil).FS.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s after reads = %v; want it absent", path, err)
		}
	}
}

func TestAProjectIsFoundFromAFolderInsideIt(t *testing.T) {
	p, m := made(t)
	if root, err := p.Find(t.Context(), filepath.Join(m.Root, "spwn", "agents")); err != nil || root != m.Root {
		t.Fatalf("Find = %q, %v; want %q", root, err, m.Root)
	}
}
