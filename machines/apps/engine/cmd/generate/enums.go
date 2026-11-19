package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// invariant: a named string type is an enum on the wire when the code gives
// it only its typed constants: every `const X T = "..."` of apps/engine/ is one of its
// values, and a conversion `T(x)` of anything but a string literal, or a
// typed constant whose value is no literal, makes it open (a plain string).
// An untyped literal assigned to a field of the type is not seen; the
// recorded fixtures' test reads what the code really sends.
func enumsOf(root string) map[string][]string {
	fset := token.NewFileSet()
	values := map[string][]string{}
	open := map[string]bool{}
	types := map[string]bool{}
	type parsed struct {
		pkg  string
		file *ast.File
	}
	var files []parsed
	var walk func(dir, pkg string)
	walk = func(dir, pkg string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			panic(err)
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() {
				walk(path, strings.TrimPrefix(pkg+"."+e.Name(), "."))
				continue
			}
			if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			body, err := os.ReadFile(path)
			if err != nil {
				panic(err)
			}
			f, err := parser.ParseFile(fset, path, body, parser.SkipObjectResolution)
			if err != nil {
				panic(err)
			}
			files = append(files, parsed{pkg, f})
		}
	}
	walk(filepath.Join(root, "apps", "engine"), "")
	for _, p := range files {
		for _, decl := range p.file.Decls {
			if g, ok := decl.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, s := range g.Specs {
					if ts, ok := s.(*ast.TypeSpec); ok && ts.Assign == 0 {
						if id, ok := ts.Type.(*ast.Ident); ok && id.Name == "string" {
							types[p.pkg+"."+ts.Name.Name] = true
						}
					}
				}
			}
		}
	}
	for _, p := range files {
		imports := map[string]string{}
		for _, spec := range p.file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			rel, ok := strings.CutPrefix(path, module+"/apps/engine/")
			if !ok {
				continue
			}
			name := rel[strings.LastIndex(rel, "/")+1:]
			if spec.Name != nil {
				name = spec.Name.Name
			}
			imports[name] = strings.ReplaceAll(rel, "/", ".")
		}
		typeOf := func(e ast.Expr) string {
			switch e := e.(type) {
			case *ast.Ident:
				return p.pkg + "." + e.Name
			case *ast.SelectorExpr:
				if x, ok := e.X.(*ast.Ident); ok && imports[x.Name] != "" {
					return imports[x.Name] + "." + e.Sel.Name
				}
			}
			return ""
		}
		ast.Inspect(p.file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ValueSpec:
				t := typeOf(n.Type)
				if n.Type == nil || !types[t] {
					return true
				}
				for _, v := range n.Values {
					lit, ok := v.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						open[t] = true
						continue
					}
					s, _ := strconv.Unquote(lit.Value)
					values[t] = append(values[t], s)
				}
			case *ast.CallExpr:
				t := typeOf(n.Fun)
				if !types[t] || len(n.Args) != 1 {
					return true
				}
				if lit, ok := n.Args[0].(*ast.BasicLit); !ok || lit.Kind != token.STRING {
					open[t] = true
				}
			}
			return true
		})
	}
	out := map[string][]string{}
	for t, v := range values {
		if !open[t] {
			slices.Sort(v)
			out[t] = slices.Compact(v)
		}
	}
	return out
}
