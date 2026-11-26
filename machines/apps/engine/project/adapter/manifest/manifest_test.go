package manifest_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	spwnproject "spwn.sh/packages/project"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
	"github.com/jterrazz/studio-station/apps/engine/project"
	"github.com/jterrazz/studio-station/apps/engine/project/adapter/manifest"
	"github.com/jterrazz/studio-station/apps/engine/project/contract"
	"github.com/jterrazz/studio-station/apps/engine/project/contract/contracttest"
)

func open(t *testing.T) contract.Projects {
	t.Setenv("SPWN_HOME", t.TempDir())
	return manifest.New(local.New(nil, nil, nil, nil).FS)
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
	m, err := p.Create(t.Context(), t.TempDir(), "site", catalog.Harnesses()[0])
	if err != nil {
		t.Fatal(err)
	}
	return p, m
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

func TestAnEmptyPromptIsAnIssueOfTheRender(t *testing.T) {
	p, m := made(t)
	if err := os.WriteFile(filepath.Join(m.Root, "spwn", "agents", project.Starter, "AGENTS.md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := p.Plan(t.Context(), m.Root)
	if err != nil || !slices.ContainsFunc(plan.Issues, func(i project.Issue) bool { return i.Source == project.SourceCompile && i.Level == project.Error }) {
		t.Fatalf("Plan = %+v, %v; want a render error", plan, err)
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

func TestAProjectIsFoundFromAFolderInsideIt(t *testing.T) {
	p, m := made(t)
	if root, err := p.Find(t.Context(), filepath.Join(m.Root, "spwn", "agents")); err != nil || root != m.Root {
		t.Fatalf("Find = %q, %v; want %q", root, err, m.Root)
	}
}
