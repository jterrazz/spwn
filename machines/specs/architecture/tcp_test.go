package architecture_test

import (
	"go/ast"
	"go/constant"
	"slices"
	"strings"
	"testing"
)

// invariant: rule engine-no-tcp (13.3 rule 16, SC-04): the engine's API is a
// unix socket only: nothing under remote/ or engine/process listens on TCP,
// and every other TCP listener (a lease's forwarder) binds the loopback, so
// no port of station's is reachable from the network.
func ruleEngineNoTCP(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test || f.pkg == "platform/local" || f.pkg == "platform/local/localtest" {
			continue
		}
		api := topOf(f.pkg) == "remote" || f.pkg == "engine/process"
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := c.Fun.(*ast.SelectorExpr)
			if !ok || !slices.Contains([]string{"ListenTCP", "Listen", "ListenPacket"}, sel.Sel.Name) {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "net" {
				out = append(out, finding{"engine-no-tcp", f.at(c.Pos()), "net." + sel.Sel.Name + " outside platform/local: a listener is made there alone"})
				return true
			}
			if sel.Sel.Name != "ListenTCP" {
				return true
			}
			if api {
				out = append(out, finding{"engine-no-tcp", f.at(c.Pos()), "the API server listens on its unix socket only, never on TCP"})
				return true
			}
			if len(c.Args) == 0 {
				return true
			}
			v := f.info.Types[c.Args[0]].Value
			if v == nil || v.Kind() != constant.String || !strings.HasPrefix(constant.StringVal(v), "127.") {
				out = append(out, finding{"engine-no-tcp", f.at(c.Pos()), "a TCP listener binds the loopback only, named by a constant"})
			}
			return true
		})
	}
	return out
}

func TestEngineNoTCP(t *testing.T) { judge(t, "EngineNoTCP") }
