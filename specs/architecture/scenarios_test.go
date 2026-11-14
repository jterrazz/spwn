package architecture_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const scenariosPage = "apps/window/SCENARIOS.md"

var (
	scenarioRow   = regexp.MustCompile(`(?m)^\| ([A-Z]\d+) +\|`)
	scenarioTitle = regexp.MustCompile("\\btest(?:\\.todo)?\\(\\s*['\"`]([A-Z]\\d+) ")
)

// invariant: rule one-test-per-scenario: the window's scenarios have one owner,
// apps/window/SCENARIOS.md, and each is a test of specs/website/
// whose title opens on its id, a test.todo while it waits: an id the page
// holds twice, an id no test declares and a test naming an id the page does
// not hold all fail, so the list and the specs never drift apart.
func ruleOneTestPerScenario(page string, specs map[string]string) []finding {
	var out []finding
	held := map[string]bool{}
	for _, m := range scenarioRow.FindAllStringSubmatch(page, -1) {
		if held[m[1]] {
			out = append(out, finding{"one-test-per-scenario", scenariosPage, m[1] + " is the id of two scenarios: give the second a new one"})
		}
		held[m[1]] = true
	}
	declared := map[string]bool{}
	files := make([]string, 0, len(specs))
	for path := range specs {
		files = append(files, path)
	}
	slices.Sort(files)
	for _, path := range files {
		for _, m := range scenarioTitle.FindAllStringSubmatch(specs[path], -1) {
			declared[m[1]] = true
			if !held[m[1]] {
				out = append(out, finding{"one-test-per-scenario", path, m[1] + " is no scenario of " + scenariosPage})
			}
		}
	}
	ids := make([]string, 0, len(held))
	for id := range held {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		if !declared[id] {
			out = append(out, finding{"one-test-per-scenario", scenariosPage, id + " has no test in specs/website/: declare it, a test.todo saying why it waits"})
		}
	}
	return out
}

func websiteSpecs(t *testing.T) map[string]string {
	t.Helper()
	root := repoRoot(t)
	var found []string
	err := filepath.WalkDir(filepath.Join(root, "specs", "website"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".spec.ts") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		found = append(found, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	specs := make(map[string]string, len(found))
	for _, rel := range found {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		specs[rel] = string(data)
	}
	return specs
}

func TestOneTestPerScenario(t *testing.T) {
	page, err := os.ReadFile(filepath.Join(repoRoot(t), scenariosPage))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range ruleOneTestPerScenario(string(page), websiteSpecs(t)) {
		t.Errorf("%s: %s", f.where, f.what)
	}
}

func TestOneTestPerScenarioBites(t *testing.T) {
	page := "| ID | Scenario |\n| --- | --- |\n| H1 | One |\n| H2 | Two |\n| H2 | Again |\n"
	specs := map[string]string{"specs/website/home/home.spec.ts": "test.todo('H1 lot 2');\ntest(\"Z9 a stray\", async () => {});\n"}
	found := ruleOneTestPerScenario(page, specs)
	got := make([]string, 0, len(found))
	for _, f := range found {
		got = append(got, f.what)
	}
	want := []string{
		"H2 is the id of two scenarios: give the second a new one",
		"Z9 is no scenario of " + scenariosPage,
		"H2 has no test in specs/website/: declare it, a test.todo saying why it waits",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule reports %q, want %q", got, want)
	}
}
