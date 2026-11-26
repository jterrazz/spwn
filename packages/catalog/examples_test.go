package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"spwn.sh/packages/dependency/builtin"
	"spwn.sh/packages/project"
	runtimespkg "spwn.sh/packages/runtimes"
)

// value: the examples are the engine's, under its catalog; `spwn up` still builds their spwn: dependencies from this catalog
const examplesDir = "../../machines/catalog/projects"

func filesUnder(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(at string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, at)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(at)
		out[filepath.ToSlash(rel)] = data
		return err
	})
	if err != nil || len(out) == 0 {
		t.Fatalf("read %s: %v, %d files", root, err, len(out))
	}
	return out
}

func TestTheSeveranceFixtureIsTheCatalogsExample(t *testing.T) {
	fixture := filesUnder(t, "../../specs/cli/__fixtures__/severance")
	example := filesUnder(t, filepath.Join(examplesDir, "severance"))
	for rel, data := range example {
		held, ok := fixture[rel]
		switch {
		case !ok:
			t.Errorf("the fixture lacks %s", rel)
		case string(held) != string(data):
			t.Errorf("the fixture's %s differs from the example's", rel)
		}
	}
	for rel := range fixture {
		if _, ok := example[rel]; !ok {
			t.Errorf("the fixture holds %s, which the example does not", rel)
		}
	}
}

func TestEveryExampleDependsOnlyOnToolsOfThisCatalog(t *testing.T) {
	entries, err := os.ReadDir(examplesDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("read %s: %v, %d entries; want the six examples", examplesDir, err, len(entries))
	}
	adapters := runtimespkg.All()
	var builtins, supported []string
	for _, tool := range builtin.BuiltinTools() {
		builtins = append(builtins, tool.Name())
	}
	for _, a := range adapters {
		supported = append(supported, a.Name)
		if a.Tool != nil {
			builtins = append(builtins, a.Tool.Name())
			supported = append(supported, a.Tool.Name())
		}
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			p, err := project.Find(filepath.Join(examplesDir, e.Name()))
			if err != nil || p == nil {
				t.Fatalf("project.Find = %v, %v; want the example's project", p, err)
			}
			var errs []string
			for _, iss := range project.Validate(p, project.ValidateOpts{BuiltinTools: builtins, SupportedRuntimes: supported}) {
				if iss.Level == project.LevelError {
					errs = append(errs, iss.Path+": "+iss.Message)
				}
			}
			if len(errs) > 0 {
				t.Fatalf("validate surfaced %d error(s):\n  %s", len(errs), strings.Join(errs, "\n  "))
			}
		})
	}
}
