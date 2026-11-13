package architecture_test

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	uitoken "github.com/jterrazz/studio-station/apps/engine/ui/token"
)

func under(pkg string, patterns ...string) bool {
	return slices.ContainsFunc(patterns, func(p string) bool { return matches(p, pkg) })
}

// invariant: rule 13: the drawing libraries are the kit's; fang is help's and
// the entry point's.
func ruleKit(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		for i, ip := range f.imports {
			lib := func(p string) bool { return ip == p || strings.HasPrefix(ip, p+"/") }
			ok := true
			switch {
			case lib(lip), lib(bubble), lib(huh), lib(ansi), lib(cprof), lib(term):
				ok = under(f.pkg, "ui/*")
			case lib(fang):
				ok = under(f.pkg, "ui/help", "wire")
			case lib(tea):
				ok = under(f.pkg, "ui/render/tty")
			}
			if !ok {
				out = append(out, finding{"kit", f.at(f.syntax.Imports[i].Pos()), fmt.Sprintf("%q may not import %q: drawing is the kit's", f.pkg, ip)})
			}
		}
	}
	return out
}

func levels() []string {
	return []string{"ui/token", "ui/atom", "ui/molecule", "ui/organism", "ui/template", "ui/render"}
}

func levelOf(pkg string) int {
	for i, l := range slices.Backward(levels()) {
		if pkg == l || strings.HasPrefix(pkg, l+"/") {
			return i
		}
	}
	return -1
}

// invariant: rule 14: a kit level imports no level above its own, and the kit
// renders page nodes only, never a page function, the catalogue or the platform.
func ruleLevels(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		own := levelOf(f.pkg)
		if own < 0 {
			continue
		}
		for i, ip := range f.imports {
			rel, ok := strings.CutPrefix(ip, module+"/")
			if !ok {
				continue
			}
			switch {
			case under(rel, "pages/*", "catalog", "platform/*"):
				out = append(out, finding{"levels", f.at(f.syntax.Imports[i].Pos()), f.pkg + " renders page nodes only, not " + rel})
			case levelOf(rel) > own && (!f.test || f.pkg != "ui/render"):
				out = append(out, finding{"levels", f.at(f.syntax.Imports[i].Pos()), f.pkg + " imports " + rel + ", a level above its own"})
			}
		}
	}
	return out
}

var widthVerb = regexp.MustCompile(`%[-+ #0]*(\*|[1-9]\d*)(\.\d+)?[a-zA-Z]`)

func callee(f *file, call *ast.CallExpr) string {
	var id *ast.Ident
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		id = fn.Sel
	case *ast.Ident:
		id = fn
	default:
		return ""
	}
	obj := f.info.Uses[id]
	if obj == nil {
		return ""
	}
	key, _ := objectKey(obj)
	return key
}

func paddedFormat(f *file, call *ast.CallExpr, key string) bool {
	at := map[string]int{"fmt.Sprintf": 0, "fmt.Printf": 0, "fmt.Fprintf": 1}
	i, ok := at[key]
	if !ok || len(call.Args) <= i {
		return false
	}
	tv, ok := f.info.Types[call.Args[i]]
	if !ok || tv.Value == nil {
		return false
	}
	s, err := strconv.Unquote(tv.Value.ExactString())
	return err == nil && widthVerb.MatchString(s)
}

// invariant: rule 15: a page function returns meaning (a page, a node, a
// contract value), never text, and lays nothing out.
func ruleSemantic(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if !under(f.pkg, "pages/*") || f.test {
			continue
		}
		for _, decl := range f.syntax.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || !fn.Name.IsExported() || fn.Type.Results == nil {
				continue
			}
			for _, r := range fn.Type.Results.List {
				if t := f.info.TypeOf(r.Type); t != nil && !semantic(t) {
					out = append(out, finding{"semantic", f.at(fn.Pos()), fn.Name.Name + " returns " + t.String() + ": a page function returns page nodes"})
				}
			}
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if key := callee(f, call); key == "strings.Repeat" || paddedFormat(f, call, key) {
				out = append(out, finding{"semantic", f.at(call.Pos()), key + " lays out text in a page function"})
			}
			return true
		})
	}
	return out
}

func semantic(t types.Type) bool {
	for {
		switch u := t.(type) {
		case *types.Slice:
			t = u.Elem()
			continue
		case *types.Pointer:
			t = u.Elem()
			continue
		case *types.Array:
			t = u.Elem()
			continue
		}
		break
	}
	named, ok := t.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	path := named.Obj().Pkg().Path()
	if path == module+"/page" {
		return true
	}
	return strings.HasPrefix(path, module+"/pages/") && strings.HasSuffix(named.Obj().Name(), "JSON")
}

func bannedRunes() []rune {
	var out []rune
	for _, g := range append(uitoken.Glyphs(), "✓✅⚠❌ℹ？▶▾▸") {
		for _, r := range g {
			if r > 127 && !slices.Contains(out, r) {
				out = append(out, r)
			}
		}
	}
	return out
}

// invariant: rule 16: every glyph station prints is a token; the set is read
// from token.Glyphs itself, so a new glyph is banned the day it is declared.
func ruleGlyphs(w *world) []finding {
	var out []finding
	banned := bannedRunes()
	for _, f := range w.files {
		if f.pkg == "ui/token" || (f.test && under(f.pkg, "ui/*", "ui")) {
			continue
		}
		literals(f, func(lit *ast.BasicLit, value string) {
			for _, r := range value {
				if slices.Contains(banned, r) {
					out = append(out, finding{"glyphs", f.at(lit.Pos()), fmt.Sprintf("the glyph %q outside apps/engine/ui/token", r)})
					return
				}
			}
		})
	}
	return out
}

// invariant: rule 17: the page language carries meaning only; a field or type
// named for its presentation would let a page pick its own look.
func rulePresentation(w *world) []finding {
	var out []finding
	banned := []string{"Tone", "Color", "Style", "Width"}
	for _, f := range w.files {
		if f.pkg != "page" || f.test {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				if slices.Contains(banned, x.Name.Name) {
					out = append(out, finding{"presentation", f.at(x.Pos()), "the type " + x.Name.Name + " in the page language"})
				}
			case *ast.Field:
				for _, name := range x.Names {
					if slices.Contains(banned, name.Name) {
						out = append(out, finding{"presentation", f.at(name.Pos()), "the field " + name.Name + " in the page language"})
					}
				}
			}
			return true
		})
	}
	return out
}

// invariant: rule 20: only page functions build pages; a tab or a leaf that
// assembles its own bypasses the one place a page is judged.
func ruleBuilders(w *world) []finding {
	var out []finding
	nodes := []string{"Page", "Section", "Row", "Group", "Item"}
	for _, f := range w.files {
		if f.test || under(f.pkg, "page", "pages", "pages/*", "cmd/generate", "cmd/generate/*") {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				if named, ok := f.info.TypeOf(x).(*types.Named); ok && len(x.Elts) > 0 && named.Obj().Pkg() != nil &&
					named.Obj().Pkg().Path() == module+"/page" && slices.Contains(nodes, named.Obj().Name()) {
					out = append(out, finding{"builders", f.at(x.Pos()), "a page." + named.Obj().Name() + " built outside a page function"})
				}
			case *ast.CallExpr:
				if callee(f, x) == module+"/page.Seal" {
					out = append(out, finding{"builders", f.at(x.Pos()), "page.Seal called outside a page function"})
				}
			}
			return true
		})
	}
	return out
}

// invariant: rule 21: layout lives in the kit: outside apps/engine/ui nothing pads,
// measures or repeats a line, and only the program wraps a view.
func ruleLayout(w *world) []finding {
	var out []finding
	measures := []string{"strings.Repeat", lip + ".Width", ansi + ".StringWidth"}
	for _, f := range w.files {
		// value: cmd/generate writes markdown for docs/, never station's output.
		if under(f.pkg, "ui/*", "ui", "cmd/generate", "cmd/generate/*") || f.test {
			continue
		}
		for i, ip := range f.imports {
			if ip == "text/tabwriter" {
				out = append(out, finding{"layout", f.at(f.syntax.Imports[i].Pos()), "text/tabwriter outside the kit"})
			}
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			key := callee(f, call)
			switch {
			case slices.Contains(measures, key), paddedFormat(f, call, key):
				out = append(out, finding{"layout", f.at(call.Pos()), key + " lays out text outside the kit"})
			}
			return true
		})
	}
	return out
}

// invariant: rule 18: the renderers every fixture is proved by, the golden
// names apps/engine/ui/render/render_test.go writes.
func goldensOf(fixture string) []string {
	out := []string{"tty100", "tty80", "pipe", "json"}
	if strings.HasSuffix(fixture, ".full") {
		out = append(out, "ttylight100")
	}
	if strings.HasPrefix(fixture, "kitchen-sink.") {
		out = append(out, "tty40")
	}
	return out
}

func missingGoldens(fixtures []string, goldens map[string]bool) []finding {
	var out []finding
	for _, fx := range fixtures {
		for _, r := range goldensOf(fx) {
			if !goldens[fx+"."+r] {
				out = append(out, finding{"goldens", "apps/engine/ui/testdata/__expected__/" + fx + "." + r, "a fixture without its golden"})
			}
		}
	}
	return out
}

func TestEveryFixtureHasItsGoldens(t *testing.T) {
	root := repoRoot(t)
	entries, err := os.ReadDir(filepath.Join(root, "apps", "engine", "ui", "testdata", "pages"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures := make([]string, 0, len(entries))
	for _, e := range entries {
		fixtures = append(fixtures, strings.TrimSuffix(e.Name(), ".json"))
	}
	goldens := map[string]bool{}
	files, err := os.ReadDir(filepath.Join(root, "apps", "engine", "ui", "testdata", "__expected__"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range files {
		goldens[e.Name()] = true
	}
	t.Run("holds", func(t *testing.T) {
		if len(fixtures) == 0 {
			t.Fatal("no page fixture under apps/engine/ui/testdata/pages")
		}
		for _, f := range missingGoldens(fixtures, goldens) {
			t.Errorf("%s: %s", f.where, f.what)
		}
	})
	t.Run("bites", func(t *testing.T) {
		without := map[string]bool{}
		for k := range goldens {
			without[k] = k != "apps.full.tty80"
		}
		if !slices.ContainsFunc(missingGoldens(fixtures, without), func(f finding) bool { return strings.HasSuffix(f.where, "apps.full.tty80") }) {
			t.Error("the goldens rule did not report a fixture without its tty80 golden")
		}
	})
}

func pageStates() []string { return []string{"loading", "empty", "error", "full"} }

// value: UI-SPEC §17's fixture families, each proved in its four states; a
// family is checked because the spec lists it, not because a file exists.
func pageFamilies() []string {
	return []string{
		"home", "home-all", "apps", "settings", "settings-matrix", "background", "disk", "all-machines", "machines", "images", "ports",
		"outcome-apps-install", "outcome-settings-apply", "outcome-disk-free", "setup", "run", "kitchen-sink", "accounts", "outcome-stop", "outcome-start", "engine", "keep-awake", "outcome-keep-awake", "awake", "activity", "menubar", "ssh", "overview", "agents", "machinefile",
	}
}

// value: the fixtures named by the world they draw rather than by a state:
// the menu bar's proofs of USE-CASES §23 cite them by those names.
func worldFamilies() map[string][]string {
	return map[string][]string{
		"menubar.home":     {"one-mac", "two-macs", "infra-down", "engine-stopped", "engine-outdated"},
		"menubar.vm":       {"linux-1", "infra"},
		"menubar.mac":      {"mini", "mini-offline"},
		"menubar.ports":    {"mini", "linux-1"},
		"menubar.awake":    {"two-macs"},
		"menubar.activity": {"mac"},
	}
}

func missingPages(packages, families, fixtures []string) []finding {
	have := map[string]bool{}
	present := map[string]bool{}
	for _, fx := range fixtures {
		have[fx] = true
		if cut := strings.LastIndexByte(fx, '.'); cut > 0 {
			present[fx[:cut]] = true
		}
	}
	var out []finding
	for _, pkg := range packages {
		if !slices.ContainsFunc(families, func(f string) bool {
			return f == pkg || strings.ReplaceAll(f, "-", "") == pkg || strings.HasPrefix(f, pkg+"-")
		}) {
			out = append(out, finding{"fixtures", "apps/engine/pages/" + pkg, "a page package with no fixture family in UI-SPEC §17"})
		}
	}
	for _, f := range families {
		for _, state := range pageStates() {
			if !have[f+"."+state] {
				out = append(out, finding{"fixtures", "apps/engine/ui/testdata/pages/" + f + "." + state + ".json", "a §17 fixture family without its " + state + " state"})
			}
		}
	}
	for f, worlds := range worldFamilies() {
		for _, w := range worlds {
			if !have[f+"."+w] {
				out = append(out, finding{"fixtures", "apps/engine/ui/testdata/pages/" + f + "." + w + ".json", "a world fixture the menu bar's proofs cite is missing"})
			}
		}
	}
	for f := range present {
		if _, world := worldFamilies()[f]; !world && !slices.Contains(families, f) {
			out = append(out, finding{"fixtures", "apps/engine/ui/testdata/pages/" + f + ".*.json", "a fixture family UI-SPEC §17 does not list"})
		}
	}
	return out
}

func TestEveryPageHasItsFourFixtures(t *testing.T) {
	root := repoRoot(t)
	dirs, err := os.ReadDir(filepath.Join(root, "apps", "engine", "pages"))
	if err != nil {
		t.Fatal(err)
	}
	var packages []string
	for _, d := range dirs {
		if d.IsDir() {
			packages = append(packages, d.Name())
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, "apps", "engine", "ui", "testdata", "pages"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures := make([]string, 0, len(entries))
	for _, e := range entries {
		fixtures = append(fixtures, strings.TrimSuffix(e.Name(), ".json"))
	}
	t.Run("holds", func(t *testing.T) {
		for _, f := range missingPages(packages, pageFamilies(), fixtures) {
			t.Errorf("%s: %s", f.where, f.what)
		}
	})
	t.Run("bites", func(t *testing.T) {
		gone := func(f string) bool {
			return f == "ports.error" || strings.HasPrefix(f, "run.") || f == "menubar.home.infra-down"
		}
		got := missingPages(append(slices.Clone(packages), "mutant"), pageFamilies(), append(slices.DeleteFunc(slices.Clone(fixtures), gone), "outcome-tools.full"))
		if !slices.ContainsFunc(got, func(f finding) bool { return f.where == "apps/engine/pages/mutant" }) {
			t.Error("the fixtures rule did not report a page package with no fixture")
		}
		if !slices.ContainsFunc(got, func(f finding) bool { return strings.HasSuffix(f.where, "ports.error.json") }) {
			t.Error("the fixtures rule did not report a family missing its error state")
		}
		if !slices.ContainsFunc(got, func(f finding) bool { return strings.HasSuffix(f.where, "menubar.home.infra-down.json") }) {
			t.Error("the fixtures rule did not report a missing world fixture")
		}
		if !slices.ContainsFunc(got, func(f finding) bool { return strings.HasSuffix(f.where, "run.loading.json") }) {
			t.Error("the fixtures rule did not report a listed family with no fixture at all")
		}
		if !slices.ContainsFunc(got, func(f finding) bool { return strings.Contains(f.where, "/outcome-tools.*") }) {
			t.Error("the fixtures rule did not report a family the spec does not list")
		}
	})
}
