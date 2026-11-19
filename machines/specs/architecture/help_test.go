package architecture_test

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strconv"
	"strings"
	"testing"
)

// value: the fields of a command that its help page prints.
func helpFields() []string { return []string{"Short", "Long", "Example"} }

func helpPackage(pkg string) bool {
	return pkg == "surface/cli" || strings.HasPrefix(pkg, "surface/cli/") || pkg == "wire" || pkg == "ui/help"
}

// value: the words an expression of a help page says: its constant value
// when it has one, else every string literal it holds.
func helpTexts(f *file, e ast.Expr) []string {
	if tv, ok := f.info.Types[e]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
		return []string{constant.StringVal(tv.Value)}
	}
	var out []string
	ast.Inspect(e, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				out = append(out, v)
			}
		}
		return true
	})
	return out
}

func typeIs(f *file, e ast.Expr, name string) bool {
	tv, ok := f.info.Types[e]
	if !ok {
		return false
	}
	t := tv.Type
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	return types.TypeString(t, nil) == name
}

func hidden(lit *ast.CompositeLit) bool {
	for _, el := range lit.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok {
			if k, ok := kv.Key.(*ast.Ident); ok && (k.Name == "Hidden" || k.Name == "Deprecated") {
				return true
			}
		}
	}
	return false
}

// value: what the help pages say, by where it is written: each shown
// command's Short, Long and Example (a cobra command's, a kit leaf's), each group's title, each flag's usage.
func helpSaid(w *world) map[string][]string {
	out := map[string][]string{}
	for _, f := range w.files {
		if f.test || !helpPackage(f.pkg) {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.CompositeLit:
				group := typeIs(f, x, cobra+".Group")
				if hidden(x) {
					return true
				}
				for _, el := range x.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					k, ok := kv.Key.(*ast.Ident)
					if ok && (contains(helpFields(), k.Name) || group && k.Name == "Title") {
						out[f.at(kv.Pos())] = append(out[f.at(kv.Pos())], helpTexts(f, kv.Value)...)
					}
				}
			case *ast.CallExpr:
				sel, ok := x.Fun.(*ast.SelectorExpr)
				if !ok || len(x.Args) < 2 || !typeIs(f, sel.X, pflag+".FlagSet") {
					return true
				}
				usage := x.Args[len(x.Args)-1]
				out[f.at(usage.Pos())] = append(out[f.at(usage.Pos())], helpTexts(f, usage)...)
			}
			return true
		})
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// invariant: rule words, its help half (OU-11): no help page says a word
// SPEC §2.2 retired.
func ruleHelpWords(w *world) []finding {
	var out []finding
	for at, texts := range helpSaid(w) {
		for _, word := range bannedIn(texts) {
			out = append(out, finding{"help-words", at, fmt.Sprintf("a help page says %q, a word SPEC §2.2 retired", word)})
		}
	}
	return out
}

func TestHelpWords(t *testing.T) { judge(t, "HelpWords") }

func TestHelpWordsReadFlags(t *testing.T) { judge(t, "HelpWordsFlag") }

func TestHelpWordsReadConstants(t *testing.T) { judge(t, "HelpWordsConstant") }
