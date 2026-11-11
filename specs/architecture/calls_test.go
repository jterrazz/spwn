package architecture_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/src/core/machine"
	"github.com/jterrazz/studio-station/src/core/section"
	"github.com/jterrazz/studio-station/src/surface/cli/words"
)

type callValue struct {
	f      *file
	name   string
	lit    *ast.CompositeLit
	fields map[string]ast.Expr
}

// value: every package-level Call or Watch value a family's calls package
// declares, with its literal's fields.
func callValues(w *world) []callValue {
	var out []callValue
	for _, f := range w.files {
		if f.test || !isCalls(f.pkg) {
			continue
		}
		for _, d := range f.syntax.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range g.Specs {
				v, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, n := range v.Names {
					if i >= len(v.Values) || !isCallType(f.info.TypeOf(v.Values[i])) {
						continue
					}
					lit, ok := v.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					fields := map[string]ast.Expr{}
					for _, e := range lit.Elts {
						if kv, ok := e.(*ast.KeyValueExpr); ok {
							if k, ok := kv.Key.(*ast.Ident); ok {
								fields[k.Name] = kv.Value
							}
						}
					}
					out = append(out, callValue{f: f, name: n.Name, lit: lit, fields: fields})
				}
			}
		}
	}
	return out
}

func isCallType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == module+"/src/core/call" && (named.Obj().Name() == "Call" || named.Obj().Name() == "Watch")
}

// value: the identifiers a package's All function body names.
func registered(w *world, pkg string) map[string]bool {
	out := map[string]bool{}
	for _, f := range w.files {
		if f.pkg != pkg || f.test {
			continue
		}
		for _, d := range f.syntax.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name != "All" {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok {
					out[id.Name] = true
				}
				return true
			})
		}
	}
	return out
}

// invariant: rule calls-gated (AR-12): every exported Call or Watch of a
// family's calls package is in its All, names itself <family>.<verb> once,
// and sets its Applies.
func ruleCallsGated(w *world) []finding {
	var out []finding
	names := map[string]string{}
	for _, c := range callValues(w) {
		if !ast.IsExported(c.name) {
			continue
		}
		at := c.f.at(c.lit.Pos())
		family := topOf(c.f.pkg)
		if !registered(w, c.f.pkg)[c.name] {
			out = append(out, finding{"calls-gated", at, c.name + " is not in " + c.f.pkg + ".All"})
		}
		if _, ok := c.fields["Applies"]; !ok {
			out = append(out, finding{"calls-gated", at, c.name + " sets no Applies"})
		}
		lit, ok := c.fields["Name"].(*ast.BasicLit)
		if !ok {
			out = append(out, finding{"calls-gated", at, c.name + " names itself by no literal"})
			continue
		}
		name, _ := strconv.Unquote(lit.Value)
		word, verb, dotted := strings.Cut(name, ".")
		switch {
		case !dotted || verb == "" || strings.Contains(verb, "."):
			out = append(out, finding{"calls-gated", at, fmt.Sprintf("%q is not <family>.<verb>", name)})
		case word != family:
			out = append(out, finding{"calls-gated", at, fmt.Sprintf("%q is not named by its family %s", name, family)})
		}
		if other, used := names[name]; used {
			out = append(out, finding{"calls-gated", at, fmt.Sprintf("%q is already %s", name, other)})
		}
		names[name] = at
	}
	return out
}

func requires(fn ast.Node, info *types.Info) bool {
	found := false
	ast.Inspect(fn, func(n ast.Node) bool {
		ce, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := ce.Fun.(*ast.SelectorExpr); ok {
			if obj := info.Uses[sel.Sel]; obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == module+"/src/core/confirm" && (obj.Name() == "Require" || obj.Name() == "Ask") {
				found = true
			}
		}
		return !found
	})
	return found
}

// value: the method obj is, as <package>.<type>.<method>, or "".
func methodOf(obj types.Object) string {
	fn, ok := obj.(*types.Func)
	if !ok || fn.Pkg() == nil {
		return ""
	}
	sig, isSig := fn.Type().(*types.Signature)
	if !isSig || sig.Recv() == nil {
		return ""
	}
	t := sig.Recv().Type()
	if p, isPtr := t.(*types.Pointer); isPtr {
		t = p.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return ""
	}
	return strings.TrimPrefix(named.Obj().Pkg().Path(), module+"/src/") + "." + named.Obj().Name() + "." + fn.Name()
}

// value: the spans of the Ask fields of every kit.Leaf literal of a file:
// the one place a leaf phrases the question its call asked.
func leafAsks(f *file) [][2]token.Pos {
	var out [][2]token.Pos
	ast.Inspect(f.syntax, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		named, ok := f.info.TypeOf(lit).(*types.Named)
		if !ok || named.Obj().Pkg() == nil || named.Obj().Pkg().Path() != module+"/src/surface/cli/kit" || named.Obj().Name() != "Leaf" {
			return true
		}
		for _, e := range lit.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "Ask" {
					out = append(out, [2]token.Pos{kv.Value.Pos(), kv.Value.End()})
				}
			}
		}
		return true
	})
	return out
}

// value: why a selector of a surface decides a confirmation, or "".
func decides(f *file, n *ast.SelectorExpr, asks [][2]token.Pos) string {
	obj := f.info.Uses[n.Sel]
	if obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == module+"/src/core/confirm" && obj.Name() == "Yes" && f.pkg != "surface/cli/kit" {
		return "a surface answers a confirmation before its call asks (confirm.Yes)"
	}
	if methodOf(obj) != "surface/cli/kit.Asker.Confirm" {
		return ""
	}
	inAsk := slices.ContainsFunc(asks, func(span [2]token.Pos) bool { return n.Pos() >= span[0] && n.End() <= span[1] })
	if !inAsk && f.pkg != "surface/cli/kit" {
		return "a leaf puts a question outside its Leaf's Ask: only kit asks, once the call has"
	}
	return ""
}

// invariant: rule confirm-in-usecase (SC-07): a call that names a
// confirmation asks it in its own Run; no surface decides one. A surface
// neither requires a confirmation nor answers one before its call asks
// (confirm.Yes is kit's, for --yes); a CLI leaf puts one only in its
// Leaf's Ask, which kit calls once the call has asked.
func ruleConfirmInUsecase(w *world) []finding {
	var out []finding
	for _, c := range callValues(w) {
		if _, asks := c.fields["Confirm"]; asks && !requires(c.fields["Run"], c.f.info) {
			out = append(out, finding{"confirm-in-usecase", c.f.at(c.lit.Pos()), c.name + " names a confirmation its Run never requires"})
		}
	}
	for _, f := range w.files {
		if f.test || topOf(f.pkg) != "surface" {
			continue
		}
		if requires(f.syntax, f.info) {
			out = append(out, finding{"confirm-in-usecase", f.rel, "a surface decides a confirmation (confirm.Require): only a use case does"})
		}
		asks := leafAsks(f)
		ast.Inspect(f.syntax, func(node ast.Node) bool {
			n, isSel := node.(*ast.SelectorExpr)
			if !isSel {
				return true
			}
			if what := decides(f, n, asks); what != "" {
				out = append(out, finding{"confirm-in-usecase", f.at(n.Pos()), what})
			}
			return true
		})
	}
	return out
}

// invariant: rule every-code-worded (ER-07): every fail.Code constant the
// tree declares is worded in surface/cli/words, and raised: named by code
// outside surface/cli/words, whose table and declarations are no raise.
func ruleEveryCodeWorded(w *world) []finding {
	var out []finding
	worded := map[string]bool{}
	for code := range words.Codes() {
		worded[string(code)] = true
	}
	raised := map[*types.Const]bool{}
	for _, f := range w.files {
		if f.test || f.pkg == "surface/cli/words" {
			continue
		}
		for _, obj := range f.info.Uses {
			if c, ok := obj.(*types.Const); ok && isCode(c.Type()) {
				raised[c] = true
			}
		}
	}
	declared := map[string]bool{}
	for _, f := range w.files {
		if f.test {
			continue
		}
		for _, d := range f.syntax.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range g.Specs {
				v, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, n := range v.Names {
					c, ok := f.info.Defs[n].(*types.Const)
					if !ok || !isCode(c.Type()) {
						continue
					}
					value := c.Val().ExactString()
					code, _ := strconv.Unquote(value)
					declared[code] = true
					if !worded[code] {
						out = append(out, finding{"every-code-worded", f.at(n.Pos()), fmt.Sprintf("%q is worded nowhere in surface/cli/words", code)})
					}
					if !raised[c] {
						out = append(out, finding{"every-code-worded", f.at(n.Pos()), fmt.Sprintf("%q is declared but never raised", code)})
					}
				}
			}
		}
	}
	for _, code := range slices.Sorted(maps.Keys(worded)) {
		if !declared[code] {
			out = append(out, finding{"every-code-worded", "src/surface/cli/words/codes.go", fmt.Sprintf("%q is worded but no package declares it", code)})
		}
	}
	return out
}

func isCode(t types.Type) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == module+"/src/core/fail" && named.Obj().Name() == "Code"
}

func TestCallsGated(t *testing.T)       { judge(t, "CallsGated") }
func TestConfirmInUsecase(t *testing.T) { judge(t, "ConfirmInUsecase") }
func TestNoLeafPutsAQuestionOutsideItsLeafAsk(t *testing.T) {
	judge(t, "ConfirmInUsecaseLeafAsks")
}
func TestEveryCodeWorded(t *testing.T)       { judge(t, "EveryCodeWorded") }
func TestEveryCodeWordedRaised(t *testing.T) { judge(t, "EveryCodeWordedRaised") }

// value: the reference profiles of the worlds: a laptop Mac, a desktop
// Mac, a workspace, a server.
func referenceProfiles() []machine.Profile {
	return []machine.Profile{
		{Ref: machine.Ref{Name: "laptop"}, Kind: machine.Computer, System: machine.System{OS: machine.MacOS}, Caps: machine.Capabilities{Battery: true, Lid: true, Jobs: true, Engine: true}},
		{Ref: machine.Ref{Name: "desktop"}, Kind: machine.Computer, System: machine.System{OS: machine.MacOS}, Caps: machine.Capabilities{Jobs: true, Engine: true}},
		{Ref: machine.Ref{Name: "workspace", On: "laptop"}, Kind: machine.Workspace, System: machine.System{OS: machine.Linux}},
		{Ref: machine.Ref{Name: "server", On: "desktop"}, Kind: machine.Server, System: machine.System{OS: machine.Linux}},
	}
}

func TestEveryCallIsRegisteredOnceAndShowsItsSectionWhereverItApplies(t *testing.T) {
	registry := allCalls(t)
	for _, m := range registry.Metas() {
		if m.Applies == nil {
			t.Errorf("%s sets no Applies", m.Name)
			continue
		}
		for _, p := range referenceProfiles() {
			if m.Section != "" && m.Applies(p).OK && !slices.Contains(section.For(p.Kind), m.Section) {
				t.Errorf("%s lets a %s through, which has no %s section", m.Name, p.Kind, m.Section)
			}
		}
	}
}
