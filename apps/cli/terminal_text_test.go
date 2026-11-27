package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// value: what reaches the user's terminal by design unchanged, with why: a
// whole function as `file:function`, or one value it prints as
// `file:function value`, the value written as the code writes it.
var printedRaw = map[string]string{
	"agent/talk.go:talkCmd#0 text":                        "the agent's own reply, printed to its terminal (ADR-023, the limit)",
	"agent/talk.go:talkCmd#0 string(output)":              "the agent's own reply when it is not the runtime's JSON, printed to its terminal (ADR-023, the limit)",
	"agent/talk.go:talkCmd#0 line":                        "the agent's stream-json events, streamed to its terminal (ADR-023, the limit)",
	"authentication/authentication.go:renderDashboardRow": "a row whose glyph every caller styles by a ui function and whose other cells it styles itself",
}

var printFuncs = map[string]bool{"Fprint": true, "Fprintf": true, "Fprintln": true, "Print": true, "Printf": true, "Println": true}

var logFuncs = map[string]bool{"Print": true, "Printf": true, "Println": true, "Fatal": true, "Fatalf": true, "Fatalln": true, "Panic": true, "Panicf": true, "Panicln": true}

func noRaw(ast.Expr) bool { return false }

// TestNothingPrintsAProjectStringButThroughTheUIPackage holds that every
// fmt print under apps/cli (outside apps/cli/ui) and every print of
// packages/architect to the terminal gives a %s or %v only a literal, a
// call into the ui package (which passes its text through ui.Clean), or a
// variable assigned only those; any other value is printed by %q or %d.
// Under apps/cli the same holds for a log print, the source of an io.Copy
// or io.WriteString, and what is written to os.Stdout or os.Stderr.
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
				key := filepath.ToSlash(path) + ":" + name
				if _, raw := printedRaw[key]; raw {
					used[key] = true
					continue
				}
				allow := func(a ast.Expr) bool {
					k := key + " " + types.ExprString(a)
					if _, raw := printedRaw[k]; raw {
						used[k] = true
						return true
					}
					return false
				}
				ast.Inspect(body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if why := unsafeCall(call, body, terminalOnly, allow); why != "" {
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

func unsafeCall(call *ast.CallExpr, fn *ast.BlockStmt, terminalOnly bool, allow func(ast.Expr) bool) string {
	switch {
	case isPkgCall(call, "fmt", printFuncs):
		return unsafePrint(call, fn, terminalOnly, allow)
	case terminalOnly:
		return ""
	case isPkgCall(call, "log", logFuncs):
		return unsafeArgs("log."+call.Fun.(*ast.SelectorExpr).Sel.Name, call.Args, fn, allow)
	case isPkgCall(call, "io", map[string]bool{"Copy": true, "CopyN": true, "CopyBuffer": true, "WriteString": true}):
		if len(call.Args) > 1 && !safeText(call.Args[1], fn, 0) && !allow(call.Args[1]) {
			return "the source of io." + call.Fun.(*ast.SelectorExpr).Sel.Name
		}
	case isStdWrite(call):
		for i, a := range call.Args {
			if !safeText(a, fn, 0) && !allow(a) {
				return "argument " + strconv.Itoa(i+1) + " of a Write to os.Stdout or os.Stderr"
			}
		}
	}
	return ""
}

func isStdWrite(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Write" && sel.Sel.Name != "WriteString") {
		return false
	}
	return isPkgCall(&ast.CallExpr{Fun: sel.X}, "os", map[string]bool{"Stdout": true, "Stderr": true})
}

func unsafePrint(call *ast.CallExpr, fn *ast.BlockStmt, terminalOnly bool, allow func(ast.Expr) bool) string {
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
	return unsafeArgs(name, args, fn, allow)
}

func unsafeArgs(name string, args []ast.Expr, fn *ast.BlockStmt, allow func(ast.Expr) bool) string {
	if !strings.HasSuffix(name, "f") {
		for i, a := range args {
			if !safeText(a, fn, 0) && !allow(a) {
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
		if i < len(verbs) && strings.ContainsRune("svc", verbs[i]) && !safeText(a, fn, 0) && !allow(a) {
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
	case *ast.CompositeLit:
		for _, elt := range x.Elts {
			if _, lit := elt.(*ast.BasicLit); !lit {
				return false
			}
		}
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
			return unsafeArgs(x.Fun.(*ast.SelectorExpr).Sel.Name, x.Args, fn, noRaw) == ""
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
