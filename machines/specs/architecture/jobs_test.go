package architecture_test

import (
	"go/ast"
	"go/types"
	"strings"
	"testing"
)

// invariant: rule one-background-job (ADR-076 §4): the engine is the one
// job this product installs on a Mac, and it serves the ssh key agent
// itself. So every job definition the code builds (a background/contract
// JobSpec, outside the contract suites) runs the engine, its Program built
// from catalog.EngineArgs; a second job, whatever its label, has nothing
// else to run.
func ruleOneBackgroundJob(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test || strings.HasSuffix(f.pkg, "/contracttest") {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || !isJobSpec(f.info.TypeOf(lit)) {
				return true
			}
			if !runsTheEngine(f, lit) {
				out = append(out, finding{"one-background-job", f.at(lit.Pos()), "a job definition whose Program is not built from catalog.EngineArgs: the engine is the one job, and it serves the key agent itself (ADR-076)"})
			}
			return true
		})
	}
	return out
}

func isJobSpec(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Name() == "JobSpec" && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == module+"/background/contract"
}

func runsTheEngine(f *file, lit *ast.CompositeLit) bool {
	for _, e := range lit.Elts {
		kv, ok := e.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, isIdent := kv.Key.(*ast.Ident); !isIdent || key.Name != "Program" {
			continue
		}
		found := false
		ast.Inspect(kv.Value, func(n ast.Node) bool {
			sel, isSel := n.(*ast.SelectorExpr)
			if !isSel {
				return true
			}
			if v, isVar := f.info.Uses[sel.Sel].(*types.Var); isVar && v.Name() == "EngineArgs" && v.Pkg() != nil && v.Pkg().Path() == module+"/catalog" {
				found = true
			}
			return !found
		})
		return found
	}
	return false
}

func TestOneBackgroundJob(t *testing.T) { judge(t, "OneBackgroundJob") }
