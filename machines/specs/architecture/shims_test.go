package architecture_test

import (
	"fmt"
	"go/ast"
	"slices"
	"strconv"
	"testing"
)

// invariant: rule process-start-guarded: a program starts in one place,
// platform/local's runner, whose command and Start take the path LookPath
// resolved, and LookPath skips an xcode-select shim when no developer
// directory holds its tool; the guard's own `xcode-select -p`, which only
// prints, is the one other start. So no read of a Mac without developer tools
// opens macOS's install dialog, whatever program a package names.
func ruleProcessStartGuarded(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test {
			continue
		}
		name := execImport(f.syntax)
		if name == "" {
			continue
		}
		for _, d := range f.syntax.Decls {
			fn, ok := d.(*ast.FuncDecl)
			allowed := ok && f.pkg == "platform/local" && slices.Contains([]string{"command", "Start", "shimmed"}, fn.Name.Name)
			ast.Inspect(d, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok || allowed {
					return true
				}
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == name && (sel.Sel.Name == "Command" || sel.Sel.Name == "CommandContext") {
					out = append(out, finding{"process-start-guarded", f.at(sel.Pos()), fmt.Sprintf("%q starts a program itself: a program starts through local.Runner, whose LookPath skips an xcode-select shim with no developer directory", f.pkg)})
				}
				return true
			})
		}
	}
	return out
}

func execImport(f *ast.File) string {
	for _, spec := range f.Imports {
		if path, _ := strconv.Unquote(spec.Path.Value); path == "os/exec" {
			if spec.Name != nil {
				return spec.Name.Name
			}
			return "exec"
		}
	}
	return ""
}

func TestProcessStartGuarded(t *testing.T) { judge(t, "ProcessStartGuarded") }
