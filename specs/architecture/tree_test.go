package architecture_test

import (
	"go/ast"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// invariant: ARCHITECTURE 13.1: the top level of src/ names what station does.
func families() []string {
	return []string{"machines", "activity", "apps", "settings", "disk", "ports", "accounts", "background", "awake", "images", "overview", "setup", "engine", "agents", "machinefile"}
}

func plumbing() []string {
	return []string{"core", "remote", "surface", "platform", "world", "wire", "cmd", "catalog", "page", "pages", "ui", "output"}
}

func presentation() []string {
	return []string{"page", "pages", "ui", "output"}
}

func sectioned() []string {
	return []string{"overview", "activity", "apps", "settings", "disk", "ports", "accounts", "background", "awake"}
}

func topOf(pkg string) string {
	top, _, _ := strings.Cut(pkg, "/")
	return top
}

func restOf(pkg string) string {
	_, rest, _ := strings.Cut(pkg, "/")
	return rest
}

func familyInside(family, rest string) bool {
	switch rest {
	case "", "contract", "contract/contracttest", "calls":
		return true
	case "process", "jobs", "feeds":
		return family == "engine"
	}
	tech, ok := strings.CutPrefix(rest, "adapter/")
	return ok && tech != "" && !strings.Contains(tech, "/")
}

func declaresApplies(file *ast.File) bool {
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			if d.Recv == nil && d.Name.Name == "Applies" {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if v, ok := spec.(*ast.ValueSpec); ok && slices.ContainsFunc(v.Names, func(n *ast.Ident) bool { return n.Name == "Applies" }) {
					return true
				}
			}
		}
	}
	return false
}

// invariant: rule tree-screams: every top-level package is a family or named
// plumbing; a family holds only its model, contract/, calls/ and
// adapter/<tech>/ (the engine adds process/, jobs/ and feeds/); a family with
// a section has an applies.go declaring Applies once it holds more than its
// doc.go.
func ruleTree(w *world) []finding {
	var out []finding
	applies, filled := map[string]bool{}, map[string]bool{}
	for _, f := range w.files {
		if f.test {
			continue
		}
		switch filepath.Base(f.path) {
		case "applies.go":
			applies[f.pkg] = declaresApplies(f.syntax)
			filled[topOf(f.pkg)] = true
		case "doc.go":
		default:
			filled[topOf(f.pkg)] = true
		}
	}
	for _, pkg := range sortedKeys(w.packages) {
		top := topOf(pkg)
		switch {
		case slices.Contains(families(), top):
			if !familyInside(top, restOf(pkg)) {
				out = append(out, finding{"tree-screams", "src/" + pkg, "a family holds only its model, contract/, calls/ and adapter/<tech>/"})
			}
		case slices.Contains(plumbing(), top):
		default:
			out = append(out, finding{"tree-screams", "src/" + pkg, "a top-level package that is neither a family nor named plumbing"})
		}
	}
	for _, family := range sectioned() {
		if filled[family] && !applies[family] {
			out = append(out, finding{"tree-screams", "src/" + family, "a family with a section has an applies.go that declares Applies"})
		}
	}
	return out
}

func TestTreeScreams(t *testing.T) { judge(t, "TreeScreams") }
func TestTreeInside(t *testing.T)  { judge(t, "TreeInside") }
func TestTreeApplies(t *testing.T) { judge(t, "TreeApplies") }
