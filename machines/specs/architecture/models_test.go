package architecture_test

import (
	"fmt"
	"go/ast"
	"slices"
	"strings"
	"testing"
)

func modelNames() []string {
	return []string{"MacBook", "Macmini", "Mac mini", "iMac", "Mac Studio", "MacPro", "Mac Pro"}
}

func modelNameAllowed(pkg string) bool {
	return pkg == "catalog" || strings.HasPrefix(pkg, "catalog/") || pkg == "surface/cli/words" || slices.Contains(presentation(), topOf(pkg))
}

// invariant: rule no-model-names (E1): a rule reads the capabilities a Mac
// reports, never its model: a model name is spelled only by the catalogue
// and the presentation.
func ruleNoModelNames(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if modelNameAllowed(f.pkg) {
			continue
		}
		literals(f, func(lit *ast.BasicLit, value string) {
			for _, name := range modelNames() {
				if !strings.Contains(value, name) {
					continue
				}
				out = append(out, finding{"no-model-names", f.at(lit.Pos()), fmt.Sprintf("%q spells the model name %q: a rule reads capabilities", f.pkg, name)})
			}
		})
	}
	return out
}

func TestNoModelNames(t *testing.T) { judge(t, "NoModelNames") }
