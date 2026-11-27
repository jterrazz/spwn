package architecture_test

import (
	"go/ast"
	"go/constant"
	"go/types"
	"testing"
)

// invariant: rule no-wiki-folder: the engine outside the fake world reads
// no wiki folder, so every production call of the wiki's files adapter is
// files.New with the empty root, a Mac that declares none. It holds until
// the entry condition of machines/TODO.md, "Before a Mac may name its wiki
// folder", is met: a hostile-wiki.yaml suite and a 64 MiB adversarial
// corpus suite passing under their stated ceilings. The lot that meets it
// is the one that may lift this rule.
func ruleNoWikiFolder(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			fn, ok := f.info.Uses[sel.Sel].(*types.Func)
			if !ok || fn.Pkg() == nil || fn.Pkg().Path() != module+"/wiki/adapter/files" || fn.Name() != "New" {
				return true
			}
			if len(call.Args) == 1 {
				if tv, ok := f.info.Types[call.Args[0]]; ok && tv.Value != nil && tv.Value.Kind() == constant.String && constant.StringVal(tv.Value) == "" {
					return true
				}
			}
			out = append(out, finding{"no-wiki-folder", f.at(call.Pos()), "the engine is given a wiki folder: it reads none until the entry condition of machines/TODO.md, \"Before a Mac may name its wiki folder\", is met"})
			return true
		})
	}
	return out
}

func TestNoWikiFolder(t *testing.T) { judge(t, "NoWikiFolder") }
