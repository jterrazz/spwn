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
