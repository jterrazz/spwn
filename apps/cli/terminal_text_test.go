package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// value: what reaches the user's terminal by design unchanged, with why.
var printedRaw = map[string]string{
	"agent/talk.go:talkCmd#0":                             "the agent's own reply, streamed to its terminal (ADR-023, the limit)",
	"authentication/authentication.go:renderDashboardRow": "a row whose glyph every caller styles by a ui function and whose other cells it styles itself",
}

var printFuncs = map[string]bool{"Fprint": true, "Fprintf": true, "Fprintln": true, "Print": true, "Printf": true, "Println": true}

// TestNothingPrintsAProjectStringButThroughTheUIPackage holds that every
// fmt print under apps/cli (outside apps/cli/ui) and every print of
// packages/architect to the terminal gives a %s or %v only a literal, a
// call into the ui package (which passes its text through ui.Clean), or a
// variable assigned only those; any other value is printed by %q or %d.
func TestNothingPrintsAProjectStringButThroughTheUIPackage(t *testing.T) {
	var bad []string
	used := map[string]bool{}
	for _, root := range []string{".", "../../packages/architect"} {
		terminalOnly := root != "."
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && (path == "ui" || path == filepath.Join("cmd", "gen-docs") || d.Name() == "testdata") {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			for name, body := range bodies(f) {
				if _, raw := printedRaw[filepath.ToSlash(path)+":"+name]; raw {
					used[filepath.ToSlash(path)+":"+name] = true
					continue
				}
				ast.Inspect(body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok || !isPkgCall(call, "fmt", printFuncs) {
						return true
					}
					if why := unsafePrint(call, body, terminalOnly); why != "" {
						bad = append(bad, fset.Position(call.Pos()).String()+": "+why)
					}
					return true
				})
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for key := range printedRaw {
		if !used[key] {
			bad = append(bad, key+": allowed to print raw, but names no function")
		}
	}
	if len(bad) > 0 {
		t.Fatalf("a print outside apps/cli/ui gives a value no ui function cleaned (wrap it in ui.Clean, or print it by %%q):\n%s", strings.Join(bad, "\n"))
	}
}

func bodies(f *ast.File) map[string]*ast.BlockStmt {
	out := map[string]*ast.BlockStmt{}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body != nil {
				out[d.Name.Name] = d.Body
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				i := 0
				ast.Inspect(vs, func(n ast.Node) bool {
					lit, ok := n.(*ast.FuncLit)
					if !ok {
						return true
					}
					out[vs.Names[0].Name+"#"+strconv.Itoa(i)] = lit.Body
					i++
					return false
				})
			}
		}
	}
	return out
}

func isPkgCall(call *ast.CallExpr, pkg string, names map[string]bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg && (names == nil || names[sel.Sel.Name])
}

func unsafePrint(call *ast.CallExpr, fn *ast.BlockStmt, terminalOnly bool) string {
	name := call.Fun.(*ast.SelectorExpr).Sel.Name
	args := call.Args
	if strings.HasPrefix(name, "F") {
		if len(args) == 0 {
			return ""
		}
		if _, builder := args[0].(*ast.UnaryExpr); builder {
			return ""
		}
		if terminalOnly && !isPkgCall(&ast.CallExpr{Fun: args[0]}, "os", map[string]bool{"Stdout": true, "Stderr": true}) {
			return ""
		}
		args = args[1:]
	}
	return unsafeArgs(name, args, fn)
}

func unsafeArgs(name string, args []ast.Expr, fn *ast.BlockStmt) string {
	if !strings.HasSuffix(name, "f") {
		for i, a := range args {
			if !safeText(a, fn, 0) {
				return "argument " + strconv.Itoa(i+1) + " of " + name
			}
		}
		return ""
	}
	if len(args) == 0 {
		return ""
	}
	lit, ok := args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "a format that is not a literal"
	}
	format, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "an unreadable format"
	}
	verbs := formatVerbs(format)
	for i, a := range args[1:] {
		if i < len(verbs) && strings.ContainsRune("svc", verbs[i]) && !safeText(a, fn, 0) {
			return "the %" + string(verbs[i]) + " argument " + strconv.Itoa(i+1) + " of " + name
		}
	}
	return ""
}

func formatVerbs(format string) []rune {
	var verbs []rune
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		for i++; i < len(format); i++ {
			c := format[i]
			if c == '*' {
				verbs = append(verbs, 'd')
				continue
			}
			if strings.IndexByte("+-# 0123456789.[]", c) >= 0 {
				continue
			}
			if c != '%' {
				verbs = append(verbs, rune(c))
			}
			break
		}
	}
	return verbs
}

func safeText(e ast.Expr, fn *ast.BlockStmt, depth int) bool {
	if depth > 4 {
		return false
	}
	switch x := e.(type) {
	case *ast.BasicLit:
		return true
	case *ast.ParenExpr:
		return safeText(x.X, fn, depth)
	case *ast.BinaryExpr:
		return safeText(x.X, fn, depth) && safeText(x.Y, fn, depth)
	case *ast.SelectorExpr:
		id, ok := x.X.(*ast.Ident)
		return ok && id.Name == "ui"
	case *ast.CallExpr:
		if isPkgCall(x, "ui", nil) {
			return true
		}
		if isPkgCall(x, "fmt", map[string]bool{"Sprint": true, "Sprintf": true, "Sprintln": true}) {
			return unsafeArgs(x.Fun.(*ast.SelectorExpr).Sel.Name, x.Args, fn) == ""
		}
		if id, ok := x.Fun.(*ast.Ident); ok {
			return safeText(id, fn, depth+1)
		}
		return false
	case *ast.Ident:
		values := assignedValues(fn, x.Name)
		if len(values) == 0 {
			return false
		}
		for _, v := range values {
			if !safeText(v, fn, depth+1) {
				return false
			}
		}
		return true
	}
	return false
}

func assignedValues(fn *ast.BlockStmt, name string) []ast.Expr {
	var values []ast.Expr
	ast.Inspect(fn, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.AssignStmt:
			for i, l := range s.Lhs {
				if id, ok := l.(*ast.Ident); ok && id.Name == name {
					if len(s.Rhs) != len(s.Lhs) {
						values = append(values, &ast.BadExpr{})
						continue
					}
					values = append(values, s.Rhs[i])
				}
			}
		case *ast.ValueSpec:
			for i, id := range s.Names {
				if id.Name == name && i < len(s.Values) {
					values = append(values, s.Values[i])
				}
			}
		case *ast.RangeStmt:
			for _, l := range []ast.Expr{s.Key, s.Value} {
				if id, ok := l.(*ast.Ident); ok && id.Name == name {
					values = append(values, &ast.BadExpr{})
				}
			}
		}
		return true
	})
	return values
}
