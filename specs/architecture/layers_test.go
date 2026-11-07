package architecture_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"
)

const module = "github.com/jterrazz/studio-station"

const (
	cobra  = "github.com/spf13/cobra"
	pflag  = "github.com/spf13/pflag"
	fang   = "charm.land/fang/v2"
	lip    = "charm.land/lipgloss/v2"
	huh    = "charm.land/huh/v2"
	tea    = "charm.land/bubbletea/v2"
	bubble = "charm.land/bubbles/v2"
	term   = "github.com/charmbracelet/x/term"
	ansi   = "github.com/charmbracelet/x/ansi"
	cprof  = "github.com/charmbracelet/colorprofile"
	std    = "std"
	sshPkg = "golang.org/x/crypto/ssh"
)

// invariant: the import table of docs/concepts/architecture.md, owned HERE. A
// package matches the row with the most specific pattern: an exact path beats
// `x/*`, a longer prefix beats a shorter. `std` is the standard library minus
// forbiddenStd; a forbidden member is re-allowed only by its name in a row.
func importTable() map[string][]string {
	// value: what a surface reaches the use cases by: the kernel's calls,
	// confirmations and machines, each family's model and calls.
	calls := make([]string, 0, 3+2*len(families()))
	calls = append(calls, "core/call", "core/confirm", "core/machine")
	for _, f := range families() {
		calls = append(calls, f, f+"/calls")
	}
	organism := []string{"ui/molecule", "ui/atom", "ui/token", "page", lip, bubble, huh, std}
	template := append(slices.Clone(organism), "ui/organism")
	return map[string][]string{
		"cmd/station":                {"wire", "os"},
		"cmd/generate":               append([]string{"catalog", "disk", "cmd/generate/pages", "page", "ui/render/json", "os", "flag", std}, calls...),
		"cmd/generate/pages":         {"page", "pages", "pages/*", "engine", "machines", "ports", "images", "setup", "overview", "activity", "apps", "settings", "disk", "accounts", "background", "awake", "agents", "core/machine", "catalog", "ui/token", std},
		"wire":                       {"*", fang, cobra, "os", "os/signal", "syscall", "runtime/debug", "net/netip", std},
		"surface/cli":                {"surface/cli/kit", cobra, std},
		"surface/cli/kit":            append([]string{"output", "page", "core/fail", "surface/cli/words", "core/store", "engine", "catalog", "testing", cobra, pflag, std}, calls...),
		"surface/cli/*":              append([]string{"surface/cli/kit", "core/fail", "engine", "page", "pages/*", "catalog", cobra, pflag, std}, calls...),
		"engine":                     {"platform/local", "catalog", "setup", "core/call/lanes", "ports", "ports/contract", "background", "background/contract", "accounts", "accounts/contract", "settings", "settings/contract", "machines", "machines/contract", "images", "setup", "setup/contract", "core/store", "core/adapter/fake", "machines/adapter/fake", std, "activity", "ports/calls", "awake", "awake/contract", "awake/calls", "net/netip"},
		"machines/adapter/ssh":       {"machines/contract", "machines/contract/contracttest", sshPkg, sshPkg + "/knownhosts", "platform/local", "platform/local/localtest", "net/netip", std},
		"machines/adapter/tart":      {"platform/local", "machines", "machines/contract", "machines/contract/contracttest", "images/contract", "core/call", "net/netip", std},
		"platform/local":             {"os", "os/exec", "os/user", "net", "net/http", "syscall", "unsafe", "golang.org/x/sys/unix", std},
		"platform/local/localtest":   {"platform/local", "testing", "testing/fstest", "net", std},
		"catalog":                    {"embed", std},
		"page":                       {std},
		"pages":                      {"page", "catalog", std},
		"pages/*":                    {"page", "pages", "pages/*", "core/call/lanes", "core/finding", "core/machine", "core/section", "engine", "machines", "ports", "images", "setup", "overview", "activity", "apps", "settings", "disk", "accounts", "background", "awake", "agents", "catalog", std},
		"ui/token":                   {lip, term, std},
		"ui/atom":                    {"ui/token", "page", lip, qr, bubble + "/spinner", std},
		"ui/molecule":                {"ui/atom", "ui/token", "page", lip, std},
		"ui/organism":                organism,
		"ui/template":                template,
		"ui/render":                  {"ui/render/*", "page", "ui/token", ansi, std},
		"ui/render/tty":              append(slices.Clone(template), "ui/template", ansi, cprof),
		"ui/render/pipe":             {"page", "ui/token", "ui/atom", std},
		"ui/render/json":             {"page", std},
		"ui/help":                    {"ui/token", fang, lip, std},
		"ui/render/tui":              append(slices.Clone(template), "ui/template", tea),
		"output":                     {"page", "ui/render/*", "core/fail", std},
		"core/*":                     {"core/*", std},
		"core/adapter/*":             {"core/*", "platform/local", "golang.org/x/sys/unix", std},
		"core/adapter/fake":          {"core/*", "world", std},
		"core/contract/contracttest": {"core/*", "testing", std},
		"core/fail":                  {"core/finding", std},
		"surface/tui":                append([]string{"surface/tui/screens/*", "core/section", "page", "pages", "pages/*", "ui/render/tui", "catalog", "core/fail", tea, std}, calls...),
		"surface/tui/*":              append([]string{"surface/tui/screens/*", "core/section", "page", "pages", "pages/*", "ui/render/tui", "catalog", "core/fail", tea, std}, calls...),
	}
}

func forbiddenStd(path string) bool {
	switch path {
	case "os", "os/exec", "os/signal", "os/user", "net", "net/http", "syscall", "io/ioutil",
		"flag", "log", "testing", "testing/fstest", "unsafe", "plugin", "runtime/debug":
		return true
	}
	return strings.HasPrefix(path, "net/") || strings.HasPrefix(path, "testing/")
}

// invariant: the standard packages that reach the file system without importing
// os (filepath.Walk, Glob, EvalSymlinks); core, which does no I/O, imports none.
func readsFiles(row, path string) bool {
	return strings.HasPrefix(row, "core") && path == "path/filepath"
}

var testImports = []string{"testing", "testing/fstest", "flag", "os", "path/filepath", "platform/local/localtest"}

type symbolRule struct {
	symbols []string
	owners  []string
	tests   bool
}

// invariant: the forbidden symbols, matched on the resolved object as
// `<package path>.<name>` or `<package path>.<receiver type>.<name>`; a `*`
// symbol stands for every member of its package. The first matching rule
// judges. A symbol is always allowed inside its own package.
func symbolRules() []symbolRule {
	return []symbolRule{
		{symbols: []string{"os.Getenv", "os.LookupEnv", "os.Setenv", "os.UserHomeDir", "os.Hostname", "os.Geteuid"}, owners: []string{"platform/local", "platform/darwin"}},
		{symbols: []string{"os.Environ", "os.Args", "os.Exit", "os.Stdin", "os.Stdout", "os.Stderr"}, owners: []string{"cmd/station"}},
		{symbols: []string{"os.ReadFile", "os.ReadDir", "os.WriteFile", "os.MkdirAll"}, owners: []string{"platform/local", "platform/darwin", "machines/adapter/ssh", "cmd/generate"}, tests: true},
		{symbols: []string{"os.Getpid", "syscall.SIGTERM"}, owners: []string{"wire", "platform/local"}},
		{symbols: []string{"os.File", "os.File.Fd", "os.Interrupt"}, owners: []string{"platform/local", "wire"}},
		{symbols: []string{"os.*", "os/exec.*", "syscall.*", "os/user.*"}, owners: []string{"platform/local", "platform/darwin"}},
		{symbols: []string{"net.*"}, owners: []string{"platform/local", "platform/local/localtest"}, tests: true},
		{symbols: []string{"net/http.*"}, owners: []string{"platform/local"}},
		{symbols: []string{sshPkg + ".*", sshPkg + "/knownhosts.*"}, owners: []string{"machines/adapter/ssh"}},
		{symbols: []string{"fmt.Print", "fmt.Println", "fmt.Printf", "log.*", "print", "println"}},
		{symbols: []string{"fmt.Fprint", "fmt.Fprintln", "fmt.Fprintf", "io.WriteString", "io.Copy", "io.CopyN"}, owners: []string{"output", "ui/render/*", "platform/local", "cmd/generate"}},
		{symbols: cobraWriters(), owners: []string{"surface/cli/kit", "wire"}},
		{symbols: []string{"time.Now", "time.Since", "time.Until", "time.Sleep", "time.After", "time.Tick", "time.NewTimer", "time.NewTicker", "time.AfterFunc"}, owners: []string{"platform/local", "platform/darwin", "machines/adapter/ssh", "ui/render/tty", "surface/tui", "surface/tui/*"}},
		{symbols: []string{"path/filepath.Glob", "path/filepath.Walk", "path/filepath.WalkDir", "path/filepath.EvalSymlinks", "path/filepath.Abs", "io/fs.WalkDir", "io/fs.Glob"}, owners: []string{"platform/local", "platform/local/localtest"}},
		{symbols: paletteSymbols(), owners: []string{"ui/token"}},
		{symbols: []string{lip + ".*"}, owners: []string{"ui/*"}},
		{symbols: []string{bubble + "/key.*"}, owners: []string{"ui/organism", "ui/render/tty", "ui/render/tui"}},
		{symbols: []string{bubble + "/spinner.New", module + "/src/ui/token.Spinner"}, owners: []string{"ui/atom"}},
		{symbols: []string{bubble + "/spinner.*"}, owners: []string{"ui/atom", "ui/render/tty", "ui/render/tui"}},
		{symbols: []string{tea + ".*", bubble + "/*"}, owners: []string{"ui/organism", "ui/render/tui", "surface/tui", "surface/tui/*"}},
		{symbols: []string{huh + ".*"}, owners: []string{"ui/organism", "ui/render/tty", "ui/render/tui"}},
		{symbols: []string{fang + ".*"}, owners: []string{"ui/help", "wire"}},
		{symbols: prefixed(module+"/src/core/fail.", "Silent", "Interruption"), owners: []string{"surface/cli/kit", "wire"}},
		{symbols: append(prefixed(module+"/src/output.", "NewWriter", "Detect", "For"), module+"/src/output.Writer.*"), owners: []string{"surface/cli/kit", "wire"}},
	}
}

func cobraWriters() []string {
	return prefixed(cobra+".Command.", "Print", "Println", "Printf", "PrintErr", "PrintErrln", "PrintErrf",
		"OutOrStdout", "OutOrStderr", "ErrOrStderr", "InOrStdin", "SetOut", "SetErr", "SetIn", "SetOutput")
}

func paletteSymbols() []string {
	return []string{lip + ".Color", lip + ".RGBColor", "image/color.RGBA", "image/color.NRGBA"}
}

func prefixed(prefix string, names ...string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = prefix + n
	}
	return out
}

var hexRe = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)

type file struct {
	pkg     string
	path    string
	rel     string
	test    bool
	syntax  *ast.File
	info    *types.Info
	types   *types.Package
	fset    *token.FileSet
	imports []string
}

type finding struct{ rule, where, what string }

type world struct {
	root     string
	files    []*file
	packages map[string]bool
	// value: the source files cgo rewrote before compiling (repo-relative):
	// their syntax reaches the rules only as cgo's output.
	cgo []string
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func load(t *testing.T, overlay map[string]string) *world {
	t.Helper()
	root := repoRoot(t)
	cfg := &packages.Config{
		Dir:     root,
		Tests:   true,
		Mode:    packages.NeedName | packages.NeedImports | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles | packages.NeedCompiledGoFiles,
		Overlay: map[string][]byte{},
	}
	for name, src := range overlay {
		cfg.Overlay[filepath.Join(root, name)] = []byte(src)
	}
	pkgs, err := packages.Load(cfg, "./src/...")
	if err != nil {
		t.Fatal(err)
	}
	w := &world{root: root, packages: map[string]bool{}}
	seen := map[string]bool{}
	for _, p := range pkgs {
		if strings.HasSuffix(p.ID, ".test") {
			continue
		}
		for _, e := range p.Errors {
			t.Errorf("%s: %v", p.ID, e)
		}
		rel, ok := strings.CutPrefix(strings.TrimSuffix(p.PkgPath, "_test"), module+"/src/")
		if !ok {
			continue
		}
		w.packages[rel] = true
		for _, g := range p.GoFiles {
			if !slices.Contains(p.CompiledGoFiles, g) && !seen["cgo:"+g] {
				seen["cgo:"+g] = true
				r, _ := filepath.Rel(root, g)
				w.cgo = append(w.cgo, r)
			}
		}
		for i, syntax := range p.Syntax {
			path := p.CompiledGoFiles[i]
			if seen[path] {
				continue
			}
			seen[path] = true
			f := &file{pkg: rel, path: path, test: strings.HasSuffix(path, "_test.go"), syntax: syntax, info: p.TypesInfo, types: p.Types, fset: p.Fset}
			f.rel, _ = filepath.Rel(root, path)
			for _, spec := range syntax.Imports {
				ip, _ := strconv.Unquote(spec.Path.Value)
				f.imports = append(f.imports, ip)
			}
			w.files = append(w.files, f)
		}
	}
	return w
}

func matches(pattern, pkg string) bool {
	if pattern == "*" {
		return true
	}
	if base, ok := strings.CutSuffix(pattern, "/*"); ok {
		return strings.HasPrefix(pkg, base+"/")
	}
	return pattern == pkg
}

func rowOf(pkg string) (string, bool) {
	best, score := "", -1
	for pattern := range importTable() {
		if !matches(pattern, pkg) {
			continue
		}
		s := len(pattern) * 2
		if !strings.HasSuffix(pattern, "/*") {
			s = 1 << 20
		}
		if s > score {
			best, score = pattern, s
		}
	}
	return best, score >= 0
}

func (f *file) at(pos token.Pos) string {
	p := f.fset.Position(pos)
	return fmt.Sprintf("%s:%d", f.rel, p.Line)
}

func allowedImport(row, pkg, ip string, test bool) bool {
	allowed := importTable()[row]
	if test {
		allowed = append(slices.Clone(allowed), testImports...)
		if ip == module+"/src/"+pkg {
			return true
		}
	}
	if rel, ok := strings.CutPrefix(ip, module+"/src/"); ok {
		for _, a := range allowed {
			if matches(a, rel) {
				return true
			}
		}
		return false
	}
	first, _, _ := strings.Cut(ip, "/")
	if !strings.Contains(first, ".") {
		if slices.Contains(allowed, ip) {
			return true
		}
		return slices.Contains(allowed, std) && !forbiddenStd(ip) && !readsFiles(row, ip)
	}
	for _, a := range allowed {
		if ip == a || strings.HasPrefix(ip, a+"/") {
			return true
		}
	}
	return false
}

func ruleRows(w *world) []finding {
	var out []finding
	for _, pkg := range sortedKeys(w.packages) {
		if p, ruled := placeOf(pkg); ruled && p.row != rowPresentation {
			continue
		}
		if _, ok := rowOf(pkg); !ok {
			out = append(out, finding{"rows", "src/" + pkg, "no row of the import table names this package"})
		}
	}
	return out
}

func ruleImports(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		row, ok := rowOf(f.pkg)
		if !ok {
			continue
		}
		for i, ip := range f.imports {
			if !allowedImport(row, f.pkg, ip, f.test) {
				out = append(out, finding{"imports", f.at(f.syntax.Imports[i].Pos()), fmt.Sprintf("row %q may not import %q", row, ip)})
			}
		}
	}
	return out
}

func objectKey(obj types.Object) (key, pkg string) {
	if b, ok := obj.(*types.Builtin); ok {
		return b.Name(), ""
	}
	if obj.Pkg() == nil {
		return "", ""
	}
	pkg = obj.Pkg().Path()
	if fn, ok := obj.(*types.Func); ok {
		if recv := fn.Signature().Recv(); recv != nil {
			t := recv.Type()
			if p, ok := t.(*types.Pointer); ok {
				t = p.Elem()
			}
			if named, ok := t.(*types.Named); ok {
				return pkg + "." + named.Obj().Name() + "." + obj.Name(), pkg
			}
			return "", pkg
		}
	}
	if v, ok := obj.(*types.Var); ok && v.IsField() {
		return "", pkg
	}
	if obj.Parent() != obj.Pkg().Scope() {
		return "", pkg
	}
	return pkg + "." + obj.Name(), pkg
}

func symbolMatches(symbol, key, pkg string) bool {
	if base, ok := strings.CutSuffix(symbol, ".*"); ok {
		return pkg == base
	}
	if base, ok := strings.CutSuffix(symbol, "/*"); ok {
		return pkg == base || strings.HasPrefix(pkg, base+"/")
	}
	return symbol == key
}

func ruleSymbols(w *world) []finding {
	var out []finding
	rules := symbolRules()
	for _, f := range w.files {
		self := module + "/src/" + f.pkg
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := f.info.Uses[id]
			if obj == nil {
				return true
			}
			key, pkg := objectKey(obj)
			if key == "" || pkg == self {
				return true
			}
			for _, r := range rules {
				if !slices.ContainsFunc(r.symbols, func(s string) bool { return symbolMatches(s, key, pkg) }) {
					continue
				}
				if (r.tests && f.test) || slices.ContainsFunc(r.owners, func(o string) bool { return matches(o, f.pkg) }) {
					return true
				}
				out = append(out, finding{"symbols", f.at(id.Pos()), fmt.Sprintf("%s is not allowed in %q", key, f.pkg)})
				return true
			}
			return true
		})
	}
	return out
}

func ruleSystemtest(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		for i, ip := range f.imports {
			if ip == module+"/src/platform/local/localtest" && !f.test {
				out = append(out, finding{"systemtest", f.at(f.syntax.Imports[i].Pos()), "platform/local/localtest is imported outside a _test.go file"})
			}
		}
	}
	return out
}

func literals(f *file, visit func(lit *ast.BasicLit, value string)) {
	ast.Inspect(f.syntax, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || (lit.Kind != token.STRING && lit.Kind != token.CHAR) {
			return true
		}
		value := lit.Value
		if lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				value = v
			}
		} else if r, _, _, err := strconv.UnquoteChar(strings.Trim(lit.Value, "'"), '\''); err == nil {
			value = string(r)
		}
		visit(lit, value)
		return true
	})
}

func ruleANSI(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		literals(f, func(lit *ast.BasicLit, value string) {
			if strings.Contains(value, "\x1b[") || strings.Contains(lit.Value, `\x1b[`) || strings.Contains(lit.Value, `\033[`) {
				out = append(out, finding{"ansi", f.at(lit.Pos()), "an ANSI escape literal: styles come from theme, erasures from x/ansi"})
			}
		})
	}
	return out
}

func rulePalette(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.pkg == "ui/token" {
			continue
		}
		literals(f, func(lit *ast.BasicLit, value string) {
			if hexRe.MatchString(value) {
				out = append(out, finding{"palette", f.at(lit.Pos()), "a hex color outside ui/token"})
			}
		})
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			if id, ok := n.(*ast.Ident); ok {
				if obj := f.info.Uses[id]; obj != nil {
					key, _ := objectKey(obj)
					if slices.Contains(paletteSymbols(), key) {
						out = append(out, finding{"palette", f.at(id.Pos()), key + " outside ui/token"})
					}
				}
			}
			return true
		})
	}
	return out
}

func isFunc(t types.Type) bool {
	_, ok := t.Underlying().(*types.Signature)
	return ok
}

func ruleCatalog(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.pkg != "catalog" || f.test {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.FuncLit:
				out = append(out, finding{"catalog", f.at(v.Pos()), "a func literal in the catalogue"})
			case *ast.StructType:
				for _, field := range v.Fields.List {
					if t := f.info.TypeOf(field.Type); t != nil && isFunc(t) {
						out = append(out, finding{"catalog", f.at(field.Pos()), "a func-typed field in the catalogue"})
					}
				}
			}
			return true
		})
		for _, decl := range f.syntax.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					if obj := f.info.Defs[name]; obj != nil && isFunc(obj.Type()) {
						out = append(out, finding{"catalog", f.at(name.Pos()), "a func-typed package-level value in the catalogue"})
					}
				}
			}
		}
	}
	return out
}

// invariant: the models whose refusals reach the kit as errors, worded by
// its failures_*.go; the other families word theirs as codes in their calls.
func wordedByKit() []string { return []string{"machines", "setup", "ports", "awake"} }

func ruleWorded(w *world) []finding {
	worded := map[string]bool{}
	for _, f := range w.files {
		if f.pkg != "surface/cli/kit" || !strings.HasPrefix(filepath.Base(f.path), "failures_") {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "As" {
				return true
			}
			if obj := f.info.Uses[sel.Sel]; obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != "errors" {
				return true
			}
			t := f.info.TypeOf(call.Args[1])
			for {
				p, ok := t.(*types.Pointer)
				if !ok {
					break
				}
				t = p.Elem()
			}
			if named, ok := t.(*types.Named); ok && named.Obj().Pkg() != nil {
				worded[named.Obj().Pkg().Path()+"."+named.Obj().Name()] = true
			}
			return true
		})
	}
	var out []finding
	for _, f := range w.files {
		if !slices.Contains(wordedByKit(), f.pkg) || f.test {
			continue
		}
		for _, decl := range f.syntax.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if !ts.Name.IsExported() || !strings.HasSuffix(ts.Name.Name, "Error") {
					continue
				}
				if !worded[f.types.Path()+"."+ts.Name.Name] {
					out = append(out, finding{"worded", f.at(ts.Pos()), ts.Name.Name + " is no errors.As target of src/surface/cli/kit/failures_*.go"})
				}
			}
		}
	}
	return out
}

var kinds = []string{"invariant:", "hazard:", "outside:", "value:"}

func commentFindings(fset *token.FileSet, rel string, syntax *ast.File) []finding {
	if ast.IsGenerated(syntax) {
		return nil
	}
	var out []finding
	for _, group := range syntax.Comments {
		first := group.List[0].Text
		if strings.HasPrefix(first, "//go:") || strings.HasPrefix(first, "//nolint") {
			continue
		}
		text := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(first, "//"), "/*"))
		if !slices.ContainsFunc(kinds, func(k string) bool { return strings.HasPrefix(text, k) }) {
			out = append(out, finding{"comments", fmt.Sprintf("%s:%d", rel, fset.Position(group.Pos()).Line), "a comment that does not open with invariant:, hazard:, outside: or value:"})
		}
	}
	return out
}

func ruleComments(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		out = append(out, commentFindings(f.fset, f.rel, f.syntax)...)
	}
	dir := filepath.Join(w.root, "specs", "architecture")
	entries, _ := os.ReadDir(dir)
	fset := token.NewFileSet()
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		syntax, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ParseComments)
		if err != nil {
			out = append(out, finding{"comments", e.Name(), err.Error()})
			continue
		}
		out = append(out, commentFindings(fset, filepath.Join("specs", "architecture", e.Name()), syntax)...)
	}
	return out
}

func ruleRegistries(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		for _, decl := range f.syntax.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.Name == "init" {
					out = append(out, finding{"registries", f.at(d.Pos()), "an init() function"})
				}
			case *ast.GenDecl:
				if d.Tok != token.VAR || f.pkg == "catalog" {
					continue
				}
				for _, spec := range d.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range vs.Names {
						obj := f.info.Defs[name]
						if obj == nil {
							continue
						}
						switch obj.Type().Underlying().(type) {
						case *types.Slice, *types.Map:
							out = append(out, finding{"registries", f.at(name.Pos()), "a package-level slice or map: a mutable registry"})
						}
					}
				}
			}
		}
	}
	return out
}

// invariant: rule 12: the raw byte streams reach only a leaf declared
// Plumbing, in the same file as its kit.Leaf literal.
func ruleStreams(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.pkg == "surface/cli/kit" || f.test {
			continue
		}
		plumbing, calls := false, []token.Pos{}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.KeyValueExpr:
				if k, ok := x.Key.(*ast.Ident); ok && k.Name == "Plumbing" {
					if v, ok := x.Value.(*ast.Ident); ok && v.Name == "true" {
						plumbing = true
					}
				}
			case *ast.SelectorExpr:
				if x.Sel.Name != "Streams" {
					return true
				}
				if fn, ok := f.info.Uses[x.Sel].(*types.Func); ok && fn.Pkg() != nil && fn.Pkg().Path() == module+"/src/surface/cli/kit" {
					calls = append(calls, x.Pos())
				}
			}
			return true
		})
		if !plumbing {
			for _, at := range calls {
				out = append(out, finding{"streams", f.at(at), "kit.IO.Streams outside a leaf declared Plumbing"})
			}
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

type rule struct {
	name    string
	check   func(*world) []finding
	mutant  string
	source  string
	expects string
}

func rules() []rule {
	return []rule{
		{"Rows", ruleRows, "src/mutant/mutant.go", "package mutant\n", "src/mutant"},
		{"RowsPresentation", ruleRows, "src/ui/mutantleak/leak.go", "package mutantleak\n\nimport _ \"os/exec\"\n", "src/ui/mutantleak"},
		{"Imports", ruleImports, "src/catalog/mutant_imports.go", "package catalog\n\nimport _ \"os/exec\"\n", "src/catalog/mutant_imports.go"},
		{"ImportsCore", ruleImports, "src/core/machine/mutant_imports.go", "package machine\n\nimport _ \"os\"\n", "src/core/machine/mutant_imports.go"},
		{"ImportsCoreFiles", ruleImports, "src/core/section/mutant_imports.go", "package section\n\nimport _ \"path/filepath\"\n", "src/core/section/mutant_imports.go"},
		{"Symbols", ruleSymbols, "src/pages/apps/mutant_symbols.go", "package apps\n\nimport \"time\"\n\nfunc mutantNow() time.Time { return time.Now() }\n", "src/pages/apps/mutant_symbols.go"},
		{"Systemtest", ruleSystemtest, "src/ui/token/mutant_systemtest.go", "package token\n\nimport _ \"" + module + "/src/platform/local/localtest\"\n", "src/ui/token/mutant_systemtest.go"},
		{"ANSI", ruleANSI, "src/pages/apps/mutant_ansi.go", "package apps\n\nconst mutantBold = \"\\x1b[1m\"\n", "src/pages/apps/mutant_ansi.go"},
		{"Palette", rulePalette, "src/pages/apps/mutant_palette.go", "package apps\n\nconst mutantRed = \"#ff0000\"\n", "src/pages/apps/mutant_palette.go"},
		{"Catalog", ruleCatalog, "src/catalog/mutant_catalog.go", "package catalog\n\ntype mutantEntry struct{ Check func() bool }\n\nvar mutantDefault = mutantEntry{Check: func() bool { return true }}\n", "src/catalog/mutant_catalog.go"},
		{"Worded", ruleWorded, "src/ports/mutant_worded.go", "package ports\n\ntype UnwordedError struct{}\n\nfunc (UnwordedError) Error() string { return \"unworded\" }\n", "src/ports/mutant_worded.go"},
		{"Comments", ruleComments, "src/pages/apps/mutant_comments.go", "package apps\n\n// a comment that states no kind\nconst mutantComment = 1\n", "src/pages/apps/mutant_comments.go"},
		{"Streams", ruleStreams, "src/surface/cli/mutant/mutant_streams.go", "package mutant\n\nimport \"" + module + "/src/surface/cli/kit\"\n\nfunc mutantStreams(o kit.IO) { _, _ = o.Streams() }\n", "src/surface/cli/mutant"},
		{"Kit", ruleKit, "src/pages/apps/mutant_kit.go", "package apps\n\nimport _ \"" + lip + "\"\n", "src/pages/apps/mutant_kit.go"},
		{"Levels", ruleLevels, "src/ui/atom/mutant_levels.go", "package atom\n\nimport _ \"" + module + "/src/ui/render/json\"\n", "src/ui/atom/mutant_levels.go"},
		{"LevelsCatalog", ruleLevels, "src/ui/organism/mutant_levels.go", "package organism\n\nimport _ \"" + module + "/src/catalog\"\n", "src/ui/organism/mutant_levels.go"},
		{"Semantic", ruleSemantic, "src/pages/apps/mutant_semantic.go", "package apps\n\nfunc Line() string { return \"\" }\n", "src/pages/apps/mutant_semantic.go"},
		{"TUILoadingLaw", ruleTUILoading, "src/surface/tui/mutant_loading.go", "package tui\n\nimport \"context\"\n\nfunc (s shell) mutantView() string {\n\tp, _ := s.deps.Pages(context.Background(), nil)\n\treturn p.Command\n}\n", "src/surface/tui/mutant_loading.go"},
		{"TUILoadingLawCalls", ruleTUILoading, "src/surface/tui/mutant_loading_calls.go", "package tui\n\nimport (\n\t\"context\"\n\n\t\"" + module + "/src/core/call\"\n\t\"" + module + "/src/core/machine\"\n\tenginecalls \"" + module + "/src/engine/calls\"\n)\n\nfunc (s shell) mutantCallView() bool {\n\t_, err := call.Invoke(context.Background(), nil, machine.Ref{}, enginecalls.Read, enginecalls.ReadIn{}, nil)\n\treturn err == nil\n}\n", "src/surface/tui/mutant_loading_calls.go"},
		{"Glyphs", ruleGlyphs, "src/surface/cli/apps/mutant_glyphs.go", "package appscmd\n\nconst mutantMark = \"\u25cf\"\n", "src/surface/cli/apps/mutant_glyphs.go"},
		{"Presentation", rulePresentation, "src/page/mutant_presentation.go", "package page\n\ntype mutantStyled struct{ Tone int }\n", "src/page/mutant_presentation.go"},
		{"Keys", ruleKeys, "src/surface/tui/screens/tools/mutant_keys.go", "package toolsscreen\n\nimport tea \"" + tea + "\"\n\nfunc mutantKey(k tea.KeyPressMsg) bool {\n\tswitch k.String() {\n\tcase \"z\":\n\t\treturn true\n\t}\n\treturn false\n}\n", "src/surface/tui/screens/tools/mutant_keys.go"},
		{"Builders", ruleBuilders, "src/surface/tui/screens/tools/mutant_builders.go", "package toolsscreen\n\nimport \"" + module + "/src/page\"\n\nvar mutantSection = page.Section{Title: \"x\"}\n", "src/surface/tui/screens/tools/mutant_builders.go"},
		{"Layout", ruleLayout, "src/surface/cli/apps/mutant_layout.go", "package appscmd\n\nimport \"fmt\"\n\nfunc mutantPad(x string) string { return fmt.Sprintf(\"%-20s\", x) }\n", "src/surface/cli/apps/mutant_layout.go"},
		{"LayoutRepeat", ruleLayout, "src/surface/tui/screens/tools/mutant_layout.go", "package toolsscreen\n\nimport \"strings\"\n\nfunc mutantRule(n int) string { return strings.Repeat(\"-\", n) }\n", "src/surface/tui/screens/tools/mutant_layout.go"},
		{"Cgo", ruleCgo, "src/awake/adapter/macos/mutant_cgo.go", "package macos\n\nimport \"C\"\n", "src/awake/adapter/macos/mutant_cgo.go"},
		{"PowerSafe", func(w *world) []finding { return append(rulePowerSafe(w), powerRaw(w)...) }, "src/awake/mutant_safe.go", "package awake\n\nconst mutantSafe = \"pmset -a disablesleep 1\"\n", "src/awake/mutant_safe.go"},
		{"PowerFiles", rulePowerSafe, "src/awake/adapter/macos/mutant_files.go", "package macos\n\nimport \"github.com/jterrazz/studio-station/src/platform/local\"\n\nfunc mutantWrite(files local.FS) error { return files.WriteFile(\"x\", nil, 0o600) }\n", "src/awake/adapter/macos/mutant_files.go"},
		{"HelpWords", ruleHelpWords, "src/surface/cli/use/mutant_help.go", "package usecmd\n\nimport \"" + cobra + "\"\n\nvar mutantHelp = &cobra.Command{Use: \"share\", Short: \"Share a port on the tailnet\"}\n", "src/surface/cli/use/mutant_help.go"},
		{"HelpWordsFlag", ruleHelpWords, "src/surface/cli/use/mutant_flag.go", "package usecmd\n\nimport \"" + cobra + "\"\n\nfunc mutantFlag(c *cobra.Command) { c.Flags().Bool(\"here\", false, \"this host only\") }\n", "src/surface/cli/use/mutant_flag.go"},
		{"HelpWordsConstant", ruleHelpWords, "src/surface/cli/use/mutant_constant.go", "package usecmd\n\nimport \"" + cobra + "\"\n\nconst mutantWord = \"its own config\"\n\nvar mutantMove = &cobra.Command{Use: \"move\", Short: \"Move it where \" + mutantWord + \" says\"}\n", "src/surface/cli/use/mutant_constant.go"},
		{"TreeScreams", ruleTree, "src/utils/mutant.go", "package utils\n", "src/utils"},
		{"ContractsAreWhole", ruleContracts, "src/disk/contract/mutant_contract.go", "package contract\n\ntype MutantProbe interface{ Probe() }\n", "src/disk/contract/mutant_contract.go"},
		{"TreeInside", ruleTree, "src/apps/helpers/mutant.go", "package helpers\n", "src/apps/helpers"},
		{"TreeApplies", ruleTree, "src/awake/applies.go", "package awake\n", "src/awake"},
		{"TreeAppliesElsewhere", ruleTree, "src/awake/mutant_gate.go", "package awake\n\nimport \"" + module + "/src/core/machine\"\n\nvar Applies = machine.Anywhere\n", "src/awake"},
		{"FamilyImports", ruleFamilyImports, "src/apps/mutant_imports.go", "package apps\n\nimport _ \"os/exec\"\n", "src/apps/mutant_imports.go"},
		{"FamilyImportsPlatform", ruleFamilyImports, "src/disk/mutant_imports_platform.go", "package disk\n\nimport _ \"" + module + "/src/platform/local\"\n", "src/disk/mutant_imports_platform.go"},
		{"FamilyImportsSurfaceEngine", ruleFamilyImports, "src/surface/cli/start/mutant_engine.go", "package startcmd\n\nimport _ \"" + module + "/src/engine\"\n", "src/surface/cli/start/mutant_engine.go"},
		{"TechnologyWords", ruleTechnologyWords, "src/apps/mutant_words.go", "package apps\n\nconst mutantMethod = \"brew\"\n", "src/apps/mutant_words.go"},
		{"TechnologyWordsSSH", ruleTechnologyWords, "src/apps/mutant_ssh.go", "package apps\n\nvar mutantArgv = []string{\"ssh\", \"-o\", \"BatchMode=yes\", \"vm\", \"true\"}\n", "src/apps/mutant_ssh.go"},
		{"TechnologyWordsSSHPath", ruleTechnologyWords, "src/apps/mutant_ssh_path.go", "package apps\n\nconst mutantBinary = \"nohup /usr/bin/ssh vm\"\n", "src/apps/mutant_ssh_path.go"},
		{"TechnologyWordsSSHLine", ruleTechnologyWords, "src/apps/mutant_ssh_line.go", "package apps\n\nconst mutantLine = \"exec ssh -o BatchMode=yes vm\"\n", "src/apps/mutant_ssh_line.go"},
		{"OneWayIntoAVM", ruleOneWayIntoAVM, "src/apps/mutant_vm_ssh.go", "package apps\n\ntype mutantCmd struct{ Name string }\n\nvar mutantInto = mutantCmd{Name: \"ssh\"}\n", "src/apps/mutant_vm_ssh.go"},
		{"OneWayIntoAVMSCP", ruleOneWayIntoAVM, "src/apps/mutant_vm_scp.go", "package apps\n\nvar mutantCopy = []string{\"scp\", \"a\", \"vm:/tmp\"}\n", "src/apps/mutant_vm_scp.go"},
		{"OneWayIntoAVMDial", ruleOneWayIntoAVM, "src/apps/mutant_vm_dial.go", "package apps\n\nimport (\n\t\"context\"\n\n\t\"" + module + "/src/machines/contract\"\n)\n\nfunc mutantDial(ctx context.Context, d contract.GuestDialer) { _, _ = d.Dial(ctx, contract.GuestTarget{}) }\n", "src/apps/mutant_vm_dial.go"},
		{"OneWriterPerStore", ruleOneWriterPerStore, "src/apps/mutant_store.go", "package apps\n\nimport (\n\t\"context\"\n\n\t\"" + module + "/src/machines/contract\"\n)\n\nfunc mutantForget(ctx context.Context, s contract.MachineStore) { _ = s.Delete(ctx, \"api\") }\n", "src/apps/mutant_store.go"},
		{"OneWriterPerStoreBuilt", ruleOneWriterPerStore, "src/apps/mutant_store_built.go", "package apps\n\nimport \"" + module + "/src/machines/adapter/jsonfile\"\n\nvar mutantStore = jsonfile.New(nil, \"vms\")\n", "src/apps/mutant_store_built.go"},
		{"OneWriterPerStoreGroup", ruleOneWriterPerStore, "src/apps/contract/mutant_group.go", "package contract\n\nimport core \"" + module + "/src/core/contract\"\n\nfunc mutantName(p core.Preferences) error { return p.Write(\"name\", \"mutant\") }\n", "src/apps/contract/mutant_group.go"},
		{"NoOutboundLiterals", ruleNoOutboundLiterals, "src/overview/mutant_outbound.go", "package overview\n\nconst mutantLookup = \"curl -s https://ifconfig.me\"\n", "src/overview/mutant_outbound.go"},
		{"EngineNoTCP", ruleEngineNoTCP, "src/remote/serve/mutant_tcp.go", "package serve\n\ntype mutantSockets interface{ ListenTCP(ip string, port int) (any, error) }\n\nfunc mutantListen(s mutantSockets) { _, _ = s.ListenTCP(\"127.0.0.1\", 8080) }\n", "src/remote/serve/mutant_tcp.go"},
		{"NoRemoteReexec", ruleNoRemoteReexec, "src/apps/mutant_reexec.go", "package apps\n\nfunc mutantRelay(host string) []string { return []string{host, \"--\", \"station\", \"apps\", \"--json\"} }\n", "src/apps/mutant_reexec.go"},
		{"NoSudoLiteral", ruleNoSudoLiteral, "src/apps/mutant_sudo.go", "package apps\n\nvar mutantRoot = []string{\"sudo\", \"brew\", \"install\"}\n", "src/apps/mutant_sudo.go"},
		{"NoModelNames", ruleNoModelNames, "src/awake/mutant_models.go", "package awake\n\nfunc mutantLaptop(model string) bool { return model == \"MacBook\" }\n", "src/awake/mutant_models.go"},
		{"CallsGated", ruleCallsGated, "src/apps/calls/mutant_calls.go", "package calls\n\nimport \"" + module + "/src/core/call\"\n\nvar Mutant = call.Call[Deps, struct{}, struct{}]{Name: \"tools.mutant\"}\n", "src/apps/calls/mutant_calls.go"},
		{"ConfirmInUsecase", ruleConfirmInUsecase, "src/disk/calls/mutant_confirm.go", "package calls\n\nimport (\n\t\"context\"\n\n\t\"" + module + "/src/core/call\"\n\t\"" + module + "/src/core/confirm\"\n\t\"" + module + "/src/core/machine\"\n)\n\nvar mutantErase = call.Call[Deps, struct{}, struct{}]{Name: \"storage.mutant\", Applies: machine.Anywhere, Confirm: confirm.Erase,\n\tRun: func(context.Context, Deps, struct{}, call.Progress) (struct{}, error) { return struct{}{}, nil }}\n", "src/disk/calls/mutant_confirm.go"},
		{"ConfirmInUsecaseAnswered", ruleConfirmInUsecase, "src/surface/tui/screens/settings/mutant_answer.go", "package settingsscreen\n\nimport \"" + module + "/src/core/confirm\"\n\nvar mutantAnswer = confirm.Yes\n", "src/surface/tui/screens/settings/mutant_answer.go"},
		{"ConfirmInUsecaseLeafAsks", ruleConfirmInUsecase, "src/surface/cli/stop/mutant_ask.go", "package stopcmd\n\nimport (\n\t\"" + module + "/src/page\"\n\t\"" + module + "/src/surface/cli/kit\"\n)\n\nfunc mutantAsk(a kit.Asker) { _, _ = a.Confirm(page.Confirm{}) }\n", "src/surface/cli/stop/mutant_ask.go"},
		{"EveryCodeWorded", ruleEveryCodeWorded, "src/apps/mutant_code.go", "package apps\n\nimport \"" + module + "/src/core/fail\"\n\nconst mutantCode fail.Code = \"apps.mutant\"\n", "src/apps/mutant_code.go"},
		{"EveryCodeWordedRaised", ruleEveryCodeWorded, "src/surface/cli/words/mutant_unraised.go", "package words\n\nimport \"" + module + "/src/core/fail\"\n\nconst mutantUnraised fail.Code = \"usage.needs-terminal\"\n", "src/surface/cli/words/mutant_unraised.go"},
		{"NoSubprocessInSample", ruleNoSubprocessInSample, "src/activity/adapter/macos/mutant_exec.go", "package macos\n\nimport _ \"os/exec\"\n", "src/activity/adapter/macos/mutant_exec.go"},
		{"Vocabulary", ruleVocabularySource, "src/apps/mutant_vocabulary.go", "package apps\n\ntype mutantLease struct{ Fleet string }\n", "src/apps/mutant_vocabulary.go"},
		{"VocabularyAwake", ruleVocabularySource, "src/ports/mutant_awake.go", "package ports\n\ntype mutantStrip struct {\n\tHeld bool `json:\"power\"`\n}\n\nconst mutantHeld = \"Power session held\"\n", "src/ports/mutant_awake.go"},
		{"Registries", ruleRegistries, "src/pages/apps/mutant_registries.go", "package apps\n\nvar mutantRegistry = map[string]int{}\n\nfunc init() { mutantRegistry[\"x\"] = 1 }\n", "src/pages/apps/mutant_registries.go"},
	}
}

var (
	loadOnce                sync.Once
	cleanWorld, mutantWorld *world
)

func worlds(t *testing.T) (clean, mutant *world) {
	t.Helper()
	loadOnce.Do(func() {
		overlay := map[string]string{}
		for _, r := range rules() {
			overlay[r.mutant] = r.source
		}
		cleanWorld, mutantWorld = load(t, nil), load(t, overlay)
	})
	if cleanWorld == nil || mutantWorld == nil {
		t.Fatal("the packages under src/ did not load")
	}
	return cleanWorld, mutantWorld
}

func judge(t *testing.T, name string) {
	clean, mutant := worlds(t)
	i := slices.IndexFunc(rules(), func(r rule) bool { return r.name == name })
	r := rules()[i]
	t.Run("holds", func(t *testing.T) {
		for _, f := range r.check(clean) {
			t.Errorf("%s: %s", f.where, f.what)
		}
	})
	t.Run("bites", func(t *testing.T) {
		for _, f := range r.check(mutant) {
			if strings.HasPrefix(f.where, r.expects) {
				return
			}
		}
		t.Errorf("the %s rule did not report the violating file %s", r.name, r.mutant)
	})
}

func TestRows(t *testing.T)             { judge(t, "Rows") }
func TestRowsPresentation(t *testing.T) { judge(t, "RowsPresentation") }

func TestImports(t *testing.T)    { judge(t, "Imports") }
func TestSymbols(t *testing.T)    { judge(t, "Symbols") }
func TestSystemtest(t *testing.T) { judge(t, "Systemtest") }
func TestANSI(t *testing.T)       { judge(t, "ANSI") }
func TestPalette(t *testing.T)    { judge(t, "Palette") }
func TestCatalog(t *testing.T)    { judge(t, "Catalog") }
func TestWorded(t *testing.T)     { judge(t, "Worded") }
func TestComments(t *testing.T)   { judge(t, "Comments") }
func TestRegistries(t *testing.T) { judge(t, "Registries") }
func TestCgo(t *testing.T)        { judge(t, "Cgo") }
func TestPowerSafe(t *testing.T)  { judge(t, "PowerSafe") }
func TestPowerFiles(t *testing.T) { judge(t, "PowerFiles") }
func TestStreams(t *testing.T)    { judge(t, "Streams") }

func TestImportsCore(t *testing.T)      { judge(t, "ImportsCore") }
func TestImportsCoreFiles(t *testing.T) { judge(t, "ImportsCoreFiles") }

func TestKit(t *testing.T)           { judge(t, "Kit") }
func TestLevels(t *testing.T)        { judge(t, "Levels") }
func TestLevelsCatalog(t *testing.T) { judge(t, "LevelsCatalog") }
func TestSemantic(t *testing.T)      { judge(t, "Semantic") }
func TestGlyphs(t *testing.T)        { judge(t, "Glyphs") }
func TestPresentation(t *testing.T)  { judge(t, "Presentation") }
func TestKeys(t *testing.T)          { judge(t, "Keys") }
func TestBuilders(t *testing.T)      { judge(t, "Builders") }
func TestLayout(t *testing.T)        { judge(t, "Layout") }
func TestLayoutRepeat(t *testing.T)  { judge(t, "LayoutRepeat") }
