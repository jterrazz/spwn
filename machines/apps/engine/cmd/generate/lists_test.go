package main

import (
	"cmp"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type sourceFile struct {
	pkg  string
	file *ast.File
}

func sources(t *testing.T, dir string) []sourceFile {
	t.Helper()
	var out []sourceFile
	entries, err := os.ReadDir(filepath.Join(root(t), dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			out = append(out, sources(t, path)...)
			continue
		}
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root(t), path))
		if err != nil {
			t.Fatal(err)
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, body, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sourceFile{pkg: filepath.ToSlash(strings.TrimPrefix(filepath.Dir(path), "apps/engine/")), file: f})
	}
	return out
}

// invariant: a mirror of mirrors() is its real form field for field, both
// ways: the same names, types and tags as the struct its package's
// MarshalJSON writes.
func TestTheMirrorsAreTheFormsTheirTypesWrite(t *testing.T) {
	written := map[reflect.Type]string{
		reflect.TypeFor[catalog.LivesOn](): "wireLivesOn",
		reflect.TypeFor[activity.Series](): "seriesJSON",
	}
	for mirrored, form := range mirrors() {
		name, ok := written[mirrored]
		if !ok {
			t.Errorf("%v is mirrored, and this test names no struct it writes", mirrored)
			continue
		}
		pkg := strings.TrimPrefix(mirrored.PkgPath(), module+"/")
		var want []string
		for _, f := range sources(t, pkg) {
			if f.pkg != strings.TrimPrefix(pkg, "apps/engine/") {
				continue
			}
			ast.Inspect(f.file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok || spec.Name.Name != name {
					return true
				}
				st, ok := spec.Type.(*ast.StructType)
				if !ok {
					return false
				}
				for _, field := range st.Fields.List {
					tag := ""
					if field.Tag != nil {
						tag, _ = strconv.Unquote(field.Tag.Value)
					}
					for _, id := range field.Names {
						want = append(want, id.Name+" "+types.ExprString(field.Type)+" `"+tag+"`")
					}
				}
				return false
			})
		}
		if len(want) == 0 {
			t.Errorf("%s: no struct %s writes %v", pkg, name, mirrored)
			continue
		}
		var got []string
		for f := range form.Fields() {
			got = append(got, f.Name+" "+typeName(f.Type, mirrored)+" `"+string(f.Tag)+"`")
		}
		if !slices.Equal(got, want) {
			t.Errorf("%v mirrors %s.%s as %q, which is %q", form, pkg, name, got, want)
		}
	}
}

// value: a field's type as its source spells it in the package of the
// mirrored type.
func typeName(t, mirrored reflect.Type) string {
	if t == reflect.TypeFor[json.RawMessage]() {
		return "json.RawMessage"
	}
	return strings.ReplaceAll(t.String(), mirrored.PkgPath()[strings.LastIndex(mirrored.PkgPath(), "/")+1:]+".", "")
}

type writtenEvent struct {
	at   string
	name string
	keys []string
}

// value: every engine.Event the code of apps/engine/ builds, with the keys it sets
// in its literal and through the variable it is assigned to.
func writtenEvents(t *testing.T) []writtenEvent {
	t.Helper()
	jsonName := map[string]string{}
	for g := range reflect.TypeFor[engine.Event]().Fields() {
		tag, _, _ := strings.Cut(g.Tag.Get("json"), ",")
		jsonName[g.Name] = cmp.Or(tag, g.Name)
	}
	consts := map[string]string{}
	all := sources(t, "apps/engine")
	for _, f := range all {
		if f.pkg != "engine" {
			continue
		}
		ast.Inspect(f.file, func(n ast.Node) bool {
			if v, ok := n.(*ast.ValueSpec); ok {
				for i, id := range v.Names {
					if i < len(v.Values) {
						if lit, ok := v.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							consts[id.Name], _ = strconv.Unquote(lit.Value)
						}
					}
				}
			}
			return true
		})
	}
	isEvent := func(f sourceFile, e ast.Expr) bool {
		switch e := e.(type) {
		case *ast.Ident:
			return f.pkg == "engine" && e.Name == "Event"
		case *ast.SelectorExpr:
			x, ok := e.X.(*ast.Ident)
			return ok && x.Name == "engine" && e.Sel.Name == "Event"
		}
		return false
	}
	nameOf := func(e ast.Expr) string {
		switch e := e.(type) {
		case *ast.BasicLit:
			s, _ := strconv.Unquote(e.Value)
			return s
		case *ast.Ident:
			return consts[e.Name]
		case *ast.SelectorExpr:
			return consts[e.Sel.Name]
		}
		return ""
	}
	build := func(f sourceFile, fn string, lit *ast.CompositeLit) writtenEvent {
		e := writtenEvent{at: f.pkg + "." + fn, keys: literalKeys(lit, jsonName)}
		for _, el := range lit.Elts {
			if kv, ok := el.(*ast.KeyValueExpr); ok && keyName(kv) == "Event" {
				e.name = nameOf(kv.Value)
			}
		}
		return e
	}
	var out []writtenEvent
	for _, f := range all {
		for _, decl := range f.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			held := map[string]int{}
			done := map[*ast.CompositeLit]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.AssignStmt:
					if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
						return true
					}
					if lit, ok := n.Rhs[0].(*ast.CompositeLit); ok && isEvent(f, lit.Type) {
						if x, ok := n.Lhs[0].(*ast.Ident); ok {
							held[x.Name] = len(out)
							out = append(out, build(f, fn.Name.Name, lit))
							done[lit] = true
						}
					}
					if sel, ok := n.Lhs[0].(*ast.SelectorExpr); ok {
						if x, ok := sel.X.(*ast.Ident); ok {
							if i, ok := held[x.Name]; ok && !slices.Contains(out[i].keys, jsonName[sel.Sel.Name]) {
								out[i].keys = append(out[i].keys, jsonName[sel.Sel.Name])
							}
						}
					}
				case *ast.CompositeLit:
					if isEvent(f, n.Type) && !done[n] {
						out = append(out, build(f, fn.Name.Name, n))
					}
				}
				return true
			})
		}
	}
	return out
}

func keyName(kv *ast.KeyValueExpr) string {
	if id, ok := kv.Key.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func literalKeys(lit *ast.CompositeLit, jsonName map[string]string) []string {
	var keys []string
	for _, el := range lit.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok {
			if key := keyName(kv); key != "Event" {
				keys = append(keys, jsonName[key])
			}
		}
	}
	return keys
}

// invariant: event.json's variants are the events the engine writes, both
// ways: each engine.Event built in apps/engine/ is an event the watches of
// enginecalls.Owns name, with every key its variant requires and none it
// does not list; each event they name, and each key a variant lists, is
// written somewhere.
func TestTheEventVariantsAreTheEventsTheEngineWrites(t *testing.T) {
	written := writtenEvents(t)
	if len(written) == 0 {
		t.Fatal("no engine.Event found in apps/engine/: the rule reads nothing")
	}
	seen := map[string]map[string]bool{}
	for _, e := range written {
		if !slices.Contains(events(), e.name) {
			t.Errorf("%s writes the event %q, which no watch of enginecalls.Owns names", e.at, e.name)
			continue
		}
		shape := eventOf(e.name)
		if shape.name == "" {
			t.Errorf("%s writes the event %q, which eventOf gives no variant", e.at, e.name)
			continue
		}
		if seen[e.name] == nil {
			seen[e.name] = map[string]bool{}
		}
		for _, k := range e.keys {
			seen[e.name][k] = true
			if !slices.Contains(shape.required, k) && !slices.Contains(shape.optional, k) {
				t.Errorf("%s writes %q on a %s event, a key its variant %s does not list", e.at, k, e.name, shape.name)
			}
		}
		for _, k := range shape.required {
			if !slices.Contains(e.keys, k) {
				t.Errorf("%s writes a %s event without %q, which its variant %s requires", e.at, e.name, k, shape.name)
			}
		}
	}
	for _, name := range events() {
		if seen[name] == nil {
			t.Errorf("the event %q is named by enginecalls.Owns, and nothing in apps/engine/ writes it", name)
			continue
		}
		for _, k := range eventOf(name).optional {
			if !seen[name][k] {
				t.Errorf("the variant of %q lists %q, which nothing in apps/engine/ writes", name, k)
			}
		}
	}
}
