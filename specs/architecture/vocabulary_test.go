package architecture_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// value: the words of SPEC §2.2 that the code and the docs keep for their
// own concept: the Tailnet contract, a file's path, a run of blanks.
func codeWords() []string { return []string{"tailnet", "path", "space"} }

// invariant: the Power contract keeps its name (SPEC §2.2, ARCHITECTURE
// §4.0): it is the IOKit primitive. Its package, its contract test, its
// adapters, the platform doors they drive and the world's slice it reads say
// power; elsewhere a name says it only for a value of that contract. Awake's
// own pages, ids and fields never do.
func powerContract(dir, word string) bool {
	return word == "power" && slices.Contains([]string{
		"src/awake/contract", "src/awake/contract/contracttest", "src/awake/adapter/fake", "src/awake/adapter/macos",
		"src/platform/local", "src/platform/darwin", "src/world",
	}, dir)
}

func holdsPowerContract(obj types.Object) bool {
	t := types.Unalias(obj.Type())
	if sig, ok := t.(*types.Signature); ok && sig.Results().Len() == 1 {
		t = types.Unalias(sig.Results().At(0).Type())
	}
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	named, ok := t.(*types.Named)
	return ok && named.Obj().Name() == "Power" && named.Obj().Pkg() != nil && strings.HasSuffix(named.Obj().Pkg().Path(), "/src/awake/contract")
}

func codeBanned() []banned {
	return slices.DeleteFunc(bannedWords(), func(b banned) bool {
		return slices.Contains(codeWords(), strings.ToLower(b.word))
	})
}

func saysBanned(list []banned, texts []string) []string {
	var found []string
	for _, b := range list {
		if len(bannedInList([]banned{b}, texts)) > 0 {
			found = append(found, b.word)
		}
	}
	return found
}

// value: where a text was read, as a finding names it: its folder,
// repo-relative.
type corpusText struct{ dir, at, text string }

// invariant: a hidden alias says its old word on purpose (SPEC §4.3): the
// name kit.Alias registers, and a function named for the alias it builds,
// its texts with it.
func aliasCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && (sel.Sel.Name == "Alias" || sel.Sel.Name == "AliasArgs")
}

func aliasName(name string) bool {
	return strings.HasSuffix(name, "Alias") || strings.HasSuffix(name, "Aliases")
}

// invariant: another program's own text is not station's vocabulary: a
// file's location (`~/.ssh/config`, `{config}/gh/hosts.yml`), a URL, a
// launchd label or a bundle id, a menu bar story's id (`sheet.lend-account`,
// STORIES.md's name for it), a file's name, an environment variable's
// name, ssh_config's own lines (`Host *`), the JSON keys of another program
// (Tailscale's `HostName`), a code span quoted in a text, and a command line
// station runs: the literal Args, Run and Command of a step, the literal
// arguments of a run…, Script…, Shell… or Command call.
var foreignShape = regexp.MustCompile(`^(?:[~/{][^\n]*|\.[A-Za-z]\S*|[^\s/]+/\S*|\S*://\S*|[a-z]+(?:\.[a-z0-9-]+){2,}\.?|sheet\.[a-z0-9]+-[a-z0-9-]+|[A-Z][A-Z0-9]*(?:_[A-Z0-9]+)+|\s*Host (?:\*(?:\n[^\n]*)*)?|[\w-]+\.(?:toml|json|yml|yaml|env|plist|conf)|json:\x22[A-Z]\w*\x22)$`)

func commandLine(fun ast.Expr) bool {
	var name string
	switch x := fun.(type) {
	case *ast.Ident:
		name = x.Name
	case *ast.SelectorExpr:
		name = x.Sel.Name
	}
	return slices.ContainsFunc([]string{"run", "Script", "Shell", "Command"}, func(p string) bool { return strings.HasPrefix(name, p) })
}

func foreignLiterals(file *ast.File, skip map[ast.Node]bool) {
	argv := func(e ast.Expr) {
		switch e.(type) {
		case *ast.BasicLit, *ast.CompositeLit:
			ast.Inspect(e, func(n ast.Node) bool {
				if lit, ok := n.(*ast.BasicLit); ok {
					skip[lit] = true
				}
				return true
			})
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := x.Key.(*ast.Ident); ok && slices.Contains([]string{"Args", "Run", "Command"}, k.Name) {
				argv(x.Value)
			}
		case *ast.CallExpr:
			if commandLine(x.Fun) {
				for _, a := range x.Args {
					if lit, ok := a.(*ast.BasicLit); ok {
						skip[lit] = true
					}
				}
			}
		}
		return true
	})
}

// invariant: a legacy encoding says its old words on purpose, like a hidden
// alias: a schema SPEC keeps byte for byte, a v2 key read once and written
// in v3's word. The declaration says so in a comment of its own starting
// "value: alias of", and legacyEncodings lists it.
func legacyDecl(file *ast.File, d ast.Decl) string {
	start := d.Pos()
	switch x := d.(type) {
	case *ast.FuncDecl:
		if x.Doc != nil {
			start = x.Doc.Pos()
		}
	case *ast.GenDecl:
		if x.Doc != nil {
			start = x.Doc.Pos()
		}
	}
	for _, c := range file.Comments {
		if c.Pos() < start || c.Pos() > d.End() {
			continue
		}
		for _, line := range strings.Split(c.Text(), "\n") {
			if strings.HasPrefix(line, "value: alias of ") {
				return line
			}
		}
	}
	return ""
}

func legacyDecls(w *world) []string {
	var out []string
	for _, f := range w.files {
		if f.test {
			continue
		}
		for _, d := range f.syntax.Decls {
			if legacyDecl(f.syntax, d) != "" {
				out = append(out, filepath.Dir(f.rel)+" "+declName(d))
			}
		}
	}
	sort.Strings(out)
	return out
}

func declName(d ast.Decl) string {
	switch x := d.(type) {
	case *ast.FuncDecl:
		return x.Name.Name
	case *ast.GenDecl:
		for _, spec := range x.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				return s.Name.Name
			case *ast.ValueSpec:
				return s.Names[0].Name
			}
		}
	}
	return ""
}

func TestVocabularyLegacyEncodings(t *testing.T) {
	clean, _ := worlds(t)
	if got, want := legacyDecls(clean), legacyEncodings(); !slices.Equal(got, want) {
		t.Errorf("the declarations marked alias: are %v, legacyEncodings lists %v", got, want)
	}
}

func splitText(s string) string { return strings.Join(splitWords(s), " ") }

func sourceTexts(w *world) []corpusText {
	var out []corpusText
	for _, f := range w.files {
		if f.test {
			continue
		}
		// invariant: a file cgo rewrote is read from the build cache, in the
		// folder of the package it belongs to.
		dir := "src/" + f.pkg
		base := strings.TrimSuffix(filepath.Base(f.rel), ".go")
		out = append(out, corpusText{dir, f.rel, splitText(base)}, corpusText{dir, f.rel, splitText(f.syntax.Name.Name)})
		skip := map[ast.Node]bool{}
		for _, spec := range f.syntax.Imports {
			skip[spec.Path] = true
		}
		foreignLiterals(f.syntax, skip)
		for _, d := range f.syntax.Decls {
			if fn, ok := d.(*ast.FuncDecl); legacyDecl(f.syntax, d) != "" || ok && aliasName(fn.Name.Name) {
				ast.Inspect(d, func(n ast.Node) bool {
					switch x := n.(type) {
					case *ast.BasicLit:
						skip[x] = true
					case *ast.Ident:
						if x.Name != declName(d) {
							skip[x] = true
						}
					}
					return true
				})
			}
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && aliasCall(call) {
				skip[call.Args[0]] = true
			}
			x, ok := n.(*ast.Ident)
			if !ok || skip[x] || x == f.syntax.Name || f.info.Defs[x] == nil || aliasName(x.Name) || holdsPowerContract(f.info.Defs[x]) {
				return true
			}
			out = append(out, corpusText{dir, f.at(x.Pos()), splitText(x.Name)})
			return true
		})
		literals(f, func(lit *ast.BasicLit, value string) {
			if !skip[lit] && !foreignShape.MatchString(value) {
				out = append(out, corpusText{dir, f.at(lit.Pos()), inlineCodeRe.ReplaceAllString(value, "")})
			}
		})
	}
	return out
}

func walk(root, dir string, keep func(rel string) bool, read func(rel, body string)) error {
	var found []string
	err := filepath.WalkDir(filepath.Join(root, dir), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if !d.IsDir() && keep(rel) {
			found = append(found, rel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, rel := range found {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return err
		}
		read(rel, string(body))
	}
	return nil
}

var (
	fenceRe       = regexp.MustCompile("(?s)```.*?```")
	inlineCodeRe  = regexp.MustCompile("`[^`\n]*`")
	descriptionRe = regexp.MustCompile(`(?m)^description:\s*(.*)$`)
	swiftStringRe = regexp.MustCompile(`"(?:[^"\\\n]|\\.)*"`)
	swiftDeclRe   = regexp.MustCompile(`\b(?:struct|class|enum|protocol|func|let|var|case|typealias)\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

// invariant: the docs read as prose: a code block or a code span quotes a
// command or a file as it is, the decision records are history.
func docTexts(root string) ([]corpusText, error) {
	var out []corpusText
	err := walk(root, "docs", func(rel string) bool {
		return strings.HasSuffix(rel, ".md") && !strings.HasPrefix(rel, "docs/decisions/")
	}, func(rel, body string) {
		prose := inlineCodeRe.ReplaceAllString(fenceRe.ReplaceAllString(body, ""), "")
		for i, line := range strings.Split(prose, "\n") {
			out = append(out, corpusText{filepath.Dir(rel), fmt.Sprintf("%s:%d", rel, i+1), line})
		}
	})
	return out, err
}

// invariant: a spec document says its words in its description and in its
// path; specs/usecases/migrate/ proves the hidden aliases by their old words.
func specTexts(root string) ([]corpusText, error) {
	var out []corpusText
	err := walk(root, "specs", func(rel string) bool {
		return strings.HasSuffix(rel, ".spec.yaml") && !strings.HasPrefix(rel, "specs/usecases/migrate/")
	}, func(rel, body string) {
		dir := filepath.Dir(rel)
		out = append(out, corpusText{dir, rel, splitText(strings.TrimSuffix(strings.TrimPrefix(rel, "specs/"), ".spec.yaml"))})
		for _, m := range descriptionRe.FindAllStringSubmatch(body, -1) {
			out = append(out, corpusText{dir, rel, m[1]})
		}
	})
	return out, err
}

// value: the names SwiftUI's protocols require a type to declare, as
// foreign as an Apple API's: a Layout places its subviews.
func swiftRequirements() []string { return []string{"placeSubviews"} }

func swiftTexts(root string) ([]corpusText, error) {
	var out []corpusText
	err := walk(root, "apps/menubar/sources", func(rel string) bool { return strings.HasSuffix(rel, ".swift") }, func(rel, body string) {
		dir := filepath.Dir(rel)
		for i, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			at := fmt.Sprintf("%s:%d", rel, i+1)
			for _, s := range swiftStringRe.FindAllString(line, -1) {
				if text := strings.Trim(s, `"`); !foreignShape.MatchString(text) {
					out = append(out, corpusText{dir, at, text})
				}
			}
			for _, m := range swiftDeclRe.FindAllStringSubmatch(line, -1) {
				if slices.Contains(swiftRequirements(), m[1]) {
					continue
				}
				out = append(out, corpusText{dir, at, splitText(m[1])})
			}
		}
	})
	return out, err
}

// value: the keys of a catalog item whose value is a command line station
// runs, another program's words like the Args of a Go step.
func catalogArgv() []string {
	return []string{"run", "args", "command", "reload", "upgrade", "env", "url", "check", "undo"}
}

// invariant: a catalog item says its words in its values, as its Go literal
// did before it was a file, and in its comments: every scalar but a command
// line and another program's text, and every comment, read as the catalog's
// own folder so the catalog alone may still name OrbStack.
// value: the marks a YAML comment opens its lines with.
var yamlComment = regexp.MustCompile(`(?m)^\s*#\s?`)

func catalogTexts(root string) ([]corpusText, error) {
	var out []corpusText
	var failed error
	err := walk(root, "src/catalog/items", func(rel string) bool { return strings.HasSuffix(rel, ".yaml") }, func(rel, body string) {
		var doc yaml.Node
		if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
			failed = fmt.Errorf("%s: %w", rel, err)
			return
		}
		said := func(n *yaml.Node) {
			for _, c := range []string{n.HeadComment, n.LineComment, n.FootComment} {
				if c != "" {
					out = append(out, corpusText{"src/catalog", fmt.Sprintf("%s:%d", rel, n.Line), inlineCodeRe.ReplaceAllString(yamlComment.ReplaceAllString(c, ""), "")})
				}
			}
		}
		var read func(n *yaml.Node, argv bool)
		read = func(n *yaml.Node, argv bool) {
			said(n)
			switch n.Kind {
			case yaml.ScalarNode:
				if !argv && !foreignShape.MatchString(n.Value) {
					out = append(out, corpusText{"src/catalog", fmt.Sprintf("%s:%d", rel, n.Line), inlineCodeRe.ReplaceAllString(n.Value, "")})
				}
			case yaml.MappingNode:
				for i := 0; i+1 < len(n.Content); i += 2 {
					said(n.Content[i])
					read(n.Content[i+1], argv || slices.Contains(catalogArgv(), n.Content[i].Value))
				}
			default:
				for _, c := range n.Content {
					read(c, argv)
				}
			}
		}
		read(&doc, false)
	})
	if err == nil {
		err = failed
	}
	return out, err
}

func vocabularyTexts(t *testing.T, w *world) []corpusText {
	t.Helper()
	out := sourceTexts(w)
	for _, read := range []func(string) ([]corpusText, error){docTexts, specTexts, swiftTexts, catalogTexts} {
		texts, err := read(w.root)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, texts...)
	}
	return out
}

// invariant: OrbStack may stay a catalog app (SPEC §2.2): the catalog names
// it, in its Go and its items, and so does the footprint `make generate`
// projects from it; nothing else does.
func catalogueApp(c corpusText, word string) bool {
	return word == "OrbStack" && (c.dir == "src/catalog" || strings.HasPrefix(c.at, "docs/reference/footprint.md:"))
}

func vocabularySaid(texts []corpusText) (said, where map[string][]string) {
	said, where = map[string][]string{}, map[string][]string{}
	for _, c := range texts {
		for _, word := range saysBanned(codeBanned(), []string{c.text}) {
			if catalogueApp(c, word) || powerContract(c.dir, word) {
				continue
			}
			if !slices.Contains(said[c.dir], word) {
				said[c.dir] = append(said[c.dir], word)
			}
			key := c.dir + " " + word
			where[key] = append(where[key], c.at)
		}
	}
	for _, words := range said {
		sort.Strings(words)
	}
	return said, where
}

func ruleVocabulary(texts []corpusText) []finding {
	said, where := vocabularySaid(texts)
	var out []finding
	for _, dir := range slices.Sorted(maps.Keys(said)) {
		for _, word := range said[dir] {
			out = append(out, finding{"vocabulary", where[dir+" "+word][0], fmt.Sprintf("%s says %q, a word SPEC §2.2 retired", dir, word)})
		}
	}
	return out
}

func ruleVocabularySource(w *world) []finding { return ruleVocabulary(sourceTexts(w)) }

func TestVocabularySource(t *testing.T) { judge(t, "Vocabulary") }

func TestVocabularySourceNamesAwake(t *testing.T) { judge(t, "VocabularyAwake") }

func TestVocabularyKeepsPowerForItsContractOnly(t *testing.T) {
	planted := []corpusText{
		{"src/awake/contract", "src/awake/contract/power.go:1", "power"},
		{"src/platform/local", "src/platform/local/power_mac.go:1", "new power"},
		{"src/engine", "src/engine/awake.go:1", "power ended"},
		{"src/pages/menubar", "src/pages/menubar/menubar.go:1", "power-sleep"},
		{"apps/menubar/sources/page", "apps/menubar/sources/page/Page.swift:1", "power"},
	}
	found := ruleVocabulary(planted)
	got := make([]string, 0, len(found))
	for _, f := range found {
		got = append(got, f.where)
	}
	if want := []string{"apps/menubar/sources/page/Page.swift:1", "src/engine/awake.go:1", "src/pages/menubar/menubar.go:1"}; !slices.Equal(got, want) {
		t.Errorf("the rule reports power at %v, want %v: only the Power contract keeps the word", got, want)
	}
}

func TestVocabulary(t *testing.T) {
	clean, _ := worlds(t)
	for _, f := range ruleVocabulary(vocabularyTexts(t, clean)) {
		t.Errorf("%s: %s", f.where, f.what)
	}
}

func TestVocabularyBites(t *testing.T) {
	_, mutant := worlds(t)
	texts := vocabularyTexts(t, mutant)
	planted := []corpusText{
		{"docs/mutant", "docs/mutant/mutant.md:1", "The fleet of hosts"},
		{"specs/usecases/mutant", "specs/usecases/mutant/mutant.spec.yaml", "a port is leased"},
		{"apps/menubar/sources/mutant", "apps/menubar/sources/mutant/Mutant.swift:1", "Lending linux-1 a token"},
	}
	got := ruleVocabulary(append(texts, planted...))
	for _, want := range []string{"src/apps/mutant_vocabulary.go", "docs/mutant/mutant.md", "specs/usecases/mutant/mutant.spec.yaml", "apps/menubar/sources/mutant/Mutant.swift"} {
		if !slices.ContainsFunc(got, func(f finding) bool { return strings.HasPrefix(f.where, want) }) {
			t.Errorf("the vocabulary rule did not report %s, which says a retired word", want)
		}
	}
}

func TestVocabularyReadsTheCatalogsItems(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "catalog", "items", "system-utilities")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	item := "item: station.catalog/v1\nkind: app\nname: orb\ndescription: Restart the daemon\n# outside: the host keeps it\nmethods:\n    - os: macos\n      script: {run: daemon --tools, check: daemon --version}\ndisk:\n    paths: [{path: ~/.orb, level: cache, note: OrbStack's own storage}]\n"
	if err := os.WriteFile(filepath.Join(dir, "orb.yaml"), []byte(item), 0o600); err != nil {
		t.Fatal(err)
	}
	texts, err := catalogTexts(root)
	if err != nil {
		t.Fatal(err)
	}
	found := ruleVocabulary(texts)
	got := make([]string, 0, len(found))
	for _, f := range found {
		got = append(got, f.where+" "+f.what)
	}
	want := []string{
		`src/catalog/items/system-utilities/orb.yaml:4 src/catalog says "daemon", a word SPEC §2.2 retired`,
		`src/catalog/items/system-utilities/orb.yaml:6 src/catalog says "host", a word SPEC §2.2 retired`,
		`src/catalog/items/system-utilities/orb.yaml:10 src/catalog says "storage", a word SPEC §2.2 retired`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("the rule reports %v, want %v: a description, a note and a comment are read, a run line and a check are not, and the catalog may name OrbStack", got, want)
	}
}

func TestVocabularyKeepsItsPhrases(t *testing.T) {
	for _, text := range []string{"a URL or host:port", "after a power cut", "the Tailnet contract", "a file's path", "Off here", "git config --global", "dclean", "Low Power Mode is on", "deleted when it powers off", "waiting for the VM to power off", "no power management here", "Lend an account to linux-1", "Lend {Account} for {For}", "Clean up ephemeral workspaces on This Mac", "Clean up", "It works when it is on power", "Ready to host servers", "Lent to linux-1", "No account lent to review-bot"} {
		if got := saysBanned(codeBanned(), []string{text}); len(got) > 0 {
			t.Errorf("%q says %v, a phrase SPEC §2.2 keeps", text, got)
		}
	}
	for _, text := range []string{"lend list", "the daemon", "a probe", "Storage", "temporary VM", "Power session held", "PowerEnded", "power-awake", "disk clean"} {
		if got := saysBanned(codeBanned(), []string{splitText(text)}); len(got) != 1 {
			t.Errorf("%q says %v, want one retired word", text, got)
		}
	}
}

func TestVocabularyLetsOnlyTheCatalogueSayOrbStack(t *testing.T) {
	planted := []corpusText{
		{"src/catalog", "src/catalog/apps.go:1", "OrbStack"},
		{"docs/reference", "docs/reference/footprint.md:1", "OrbStack's own home"},
		{"docs/reference", "docs/reference/machines.md:1", "OrbStack runs the VMs"},
		{"src/machines", "src/machines/errors.go:1", "OrbStack"},
	}
	found := ruleVocabulary(planted)
	got := make([]string, 0, len(found))
	for _, f := range found {
		got = append(got, f.where)
	}
	if want := []string{"docs/reference/machines.md:1", "src/machines/errors.go:1"}; !slices.Equal(got, want) {
		t.Errorf("the rule reports OrbStack at %v, want %v: only the catalogue and its footprint name it", got, want)
	}
}

func TestVocabularyReadsPastForeignTextOnly(t *testing.T) {
	for _, text := range []string{"~/.ssh/config", "{config}/gh/hosts.yml", "https://example.com/tools/x.sh", "com.jterrazz.station.daemon", "STATION_PUBLIC_HOST", "config.toml", "Host ", "\nHost *\n  UseKeychain yes\n", `json:"HostName"`, "sheet.lend-account"} {
		if !foreignShape.MatchString(text) {
			t.Errorf("%q is another program's text, read as station's", text)
		}
	}
	for _, text := range []string{"the daemon runs", "Host port 40000 is busy", "Storage", "lend list", `json:"leased"`, "accounts.lend-github"} {
		if foreignShape.MatchString(text) {
			t.Errorf("%q is station's text, read past as another program's", text)
		}
	}
	src := `package p

func f() {
	kit.Command(kit.Leaf[In]{Short: "Restart the daemon"})
	local.Command("git", "config", "--global")
	_ = Step{Args: []string{"mise", "prune", "--tools"}, Hint: "the tools"}
}
`
	file, err := parser.ParseFile(token.NewFileSet(), "p.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	skip := map[ast.Node]bool{}
	foreignLiterals(file, skip)
	var read []string
	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && !skip[lit] {
			read = append(read, lit.Value)
		}
		return true
	})
	if want := []string{`"Restart the daemon"`, `"the tools"`}; !slices.Equal(read, want) {
		t.Errorf("the rule reads %v, want %v: only a command line is another program's", read, want)
	}
}

// invariant: the legacy encodings, each a declaration marked "value: alias
// of": station.status/v1 (the OS connector's, byte for byte one release),
// its rows and its kind word; the v2 keys a store reads once and writes in
// v3's words; v2's hidden commands that are no kit.Alias.
func legacyEncodings() []string {
	return []string{
		"src/accounts/adapter/jsonfile sharedIn",
		"src/accounts/contract GroupV2ShareList",
		"src/catalog StatusV1Kind",
		"src/engine/process hello",
		"src/images UnmarshalJSON",
		"src/machines UnmarshalText",
		"src/machines/adapter/jsonfile file",
		"src/pages/apps AppStatusJSON",
		"src/pages/apps statusV1Category",
		"src/pages/home SnapshotJSON",
		"src/pages/home awakeJSON",
		"src/pages/home stationJSON",
		"src/pages/home workspaceJSON",
		"src/pages/settings ItemJSON",
		"src/ports Reservation",
		"src/ports VMLivesOn",
		"src/ports/adapter/guest Hello",
		"src/ports/adapter/jsonfile ReservationsSchema",
		"src/ports/contract RetiredGroupPorts",
		"src/setup/contract RetiredGroupName",
		"src/surface/cli/kit RetiredEngineCommand",
		"src/surface/cli/settings containerShortcuts",
		"src/world Kept",
	}
}
