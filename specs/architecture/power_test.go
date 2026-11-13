package architecture_test

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func cgoFiles() []string {
	return []string{"apps/engine/platform/darwin/power_darwin.go", "apps/engine/platform/darwin/kernel_darwin.go"}
}

func cgoTags() map[string]string {
	tags := map[string]string{}
	for _, rel := range cgoFiles() {
		tags[rel] = "//go:build darwin && cgo"
		tags[strings.TrimSuffix(rel, "_darwin.go")+"_other.go"] = "//go:build !darwin || !cgo"
	}
	return tags
}

// invariant: rule 22: cgo lives in platform/darwin's named files, each built
// only on darwin with cgo, and each one's refusal twin builds everywhere
// else, so a linux binary needs no C.
func ruleCgo(w *world) []finding {
	var out []finding
	for _, rel := range w.cgo {
		if !slices.Contains(cgoFiles(), rel) {
			out = append(out, finding{"cgo", rel, "import \"C\" lives in " + strings.Join(cgoFiles(), ", ") + " only"})
		}
	}
	for rel, tag := range cgoTags() {
		data, err := os.ReadFile(filepath.Join(w.root, rel))
		if err != nil || !strings.HasPrefix(string(data), tag+"\n") {
			out = append(out, finding{"cgo", rel, "must open with " + tag})
		}
	}
	return out
}

func unsafePower() []string {
	return []string{"disablesleep", "kIOPMAssertPreventUserIdleDisplaySleep", "kIOPMAssertionTypeNoDisplaySleep", "PreventUserIdleDisplaySleep", "NoDisplaySleep"}
}

func powerPaths() []string {
	return []string{"apps/engine/awake/", "apps/engine/pages/keepawake/", "apps/engine/surface/cli/awake/", "apps/engine/pages/awake/", "apps/engine/platform/darwin/power_", "apps/engine/wire/awake.go", "apps/engine/wire/keepawake.go", "apps/engine/engine/awake.go", "apps/engine/engine/process/awake.go"}
}

func powerCode(rel string) bool {
	for _, path := range powerPaths() {
		if rel == path || !strings.HasSuffix(path, ".go") && strings.HasPrefix(rel, path) {
			return true
		}
	}
	return false
}

func powerPackages() []string {
	return []string{"awake", "awake/contract", "awake/adapter/macos", "awake/calls"}
}

// invariant: rule 23: Power is safe by construction: no string under apps/engine/
// asks for a display assertion or disablesleep, the power code holds no
// sudo, and the awake family, its macOS adapter and its calls reach no
// file system: no file is written for power.
func rulePowerSafe(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test || !strings.HasPrefix(f.rel, "apps/engine/") {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					return true
				}
				for _, word := range unsafePower() {
					if strings.Contains(n.Value, word) {
						out = append(out, finding{"power", f.at(n.Pos()), "Power never holds " + word})
					}
				}
				if powerCode(f.rel) && strings.Contains(n.Value, "sudo") {
					out = append(out, finding{"power", f.at(n.Pos()), "the power code never runs sudo"})
				}
			case *ast.Ident:
				if !slices.Contains(powerPackages(), f.pkg) {
					return true
				}
				if obj := f.info.Uses[n]; obj != nil && isFS(obj.Type()) {
					out = append(out, finding{"power", f.at(n.Pos()), f.pkg + " reaches no file system"})
				}
			}
			return true
		})
	}
	return out
}

func isFS(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Name() == "FS" && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == module+"/platform/local"
}

func powerRaw(w *world) []finding {
	var out []finding
	var matches []string
	for _, dir := range []string{"local", "darwin"} {
		found, _ := filepath.Glob(filepath.Join(w.root, "apps", "engine", "platform", dir, "power_*.go"))
		matches = append(matches, found...)
	}
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		for _, word := range unsafePower() {
			if strings.Contains(string(data), word) {
				rel, _ := filepath.Rel(w.root, m)
				out = append(out, finding{"power", rel, "Power never holds " + word})
			}
		}
	}
	return out
}

func TestPowerCodeNamesOnlyPathsThatExist(t *testing.T) {
	root := repoRoot(t)
	for _, path := range powerPaths() {
		found, _ := filepath.Glob(filepath.Join(root, path) + "*")
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "/") {
			found, _ = filepath.Glob(filepath.Join(root, path))
		}
		if len(found) == 0 {
			t.Errorf("the no-sudo check names %s, which does not exist: the power code moved and the check lost it", path)
		}
	}
}
