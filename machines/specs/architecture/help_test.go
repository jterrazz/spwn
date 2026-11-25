package architecture_test

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	command "github.com/spf13/cobra"
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

// invariant: rule examples (ADR-021 rule 2's suite): every command a user
// types shows how it is typed.
func ruleExamples(root *command.Command) []finding {
	var out []finding
	for _, c := range typedCommands(root) {
		if c.Runnable() && strings.TrimSpace(c.Example) == "" {
			out = append(out, finding{"examples", commandPath(c), "a command help shows with no Example"})
		}
	}
	return out
}

// invariant: rule help-groups (ADR-021 rule 3): every noun and word at the
// top of the tree stands in one of the help's groups.
func ruleHelpGroups(root *command.Command) []finding {
	var out []finding
	for _, c := range typedCommands(root) {
		if top(c) && !slices.ContainsFunc(root.Groups(), func(g *command.Group) bool { return g.ID == c.GroupID }) {
			out = append(out, finding{"help-groups", commandPath(c), "a command at the top of the tree in no help group"})
		}
	}
	return out
}

type example struct {
	at    string
	line  string
	words []string
}

var placeholder = regexp.MustCompile(`<[^>]*>`)

// value: the station runs an example line holds: the line itself, or each
// $(station ...) it composes; a # comment is no word of it.
func examplesOf(root *command.Command) ([]example, []finding) {
	var out []example
	var refused []finding
	for _, c := range typedCommands(root) {
		for _, line := range strings.Split(c.Example, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			code, _, _ := strings.Cut(line, " #")
			runs := []string{code}
			if strings.Contains(code, "$(") {
				runs = nil
				for _, part := range strings.Split(code, "$(")[1:] {
					inner, _, _ := strings.Cut(part, ")")
					runs = append(runs, inner)
				}
			}
			for _, run := range runs {
				words := strings.Fields(run)
				switch {
				case len(words) == 0 || words[0] != root.Name():
					refused = append(refused, finding{"examples-parse", commandPath(c) + ": " + line, "an example that runs no " + root.Name()})
				case placeholder.MatchString(run):
					refused = append(refused, finding{"examples-parse", commandPath(c) + ": " + line, "an example with a placeholder: it names a real value"})
				default:
					out = append(out, example{at: commandPath(c), line: line, words: words[1:]})
				}
			}
		}
	}
	return out, refused
}

// value: why the tree refuses an example's words before its command runs:
// no command under the one it is shown on, a flag, an argument; empty when
// they parse.
func parseRefusal(root *command.Command, e example) string {
	at, _, err := root.Find(strings.Fields(e.at))
	if err != nil {
		return err.Error()
	}
	target, rest, err := root.Find(e.words)
	if err != nil {
		return err.Error()
	}
	for c := target; c != at; c = c.Parent() {
		if !c.HasParent() {
			return "it runs " + commandPath(target) + ", no command under " + e.at
		}
	}
	if target.DisableFlagParsing {
		return ""
	}
	if err := target.ParseFlags(rest); err != nil {
		return err.Error()
	}
	if err := target.ValidateArgs(target.Flags().Args()); err != nil {
		return err.Error()
	}
	if err := target.ValidateRequiredFlags(); err != nil {
		return err.Error()
	}
	if err := target.ValidateFlagGroups(); err != nil {
		return err.Error()
	}
	return ""
}

// invariant: rule examples-parse (audit W10): every example's words are a
// command under the one it is shown on, with its flags and its arguments.
func ruleExamplesParse(fresh func() *command.Command) []finding {
	examples, out := examplesOf(fresh())
	for _, e := range examples {
		if why := parseRefusal(fresh(), e); why != "" {
			out = append(out, finding{"examples-parse", e.at + ": " + e.line, "an example the tree refuses: " + why})
		}
	}
	return out
}

var wordsRefused = regexp.MustCompile(`(?m)^Hint: (run \S+ .*--help|\S+ help)$`)

// value: why a run of an example is no run of it: it crashed, hung, ended
// outside the exit table, or was refused for its words rather than answered;
// empty for a run that answered, whatever its answer.
func runRefusal(r ran) string {
	switch {
	case r.timedOut:
		return "it did not end within a minute"
	case strings.Contains(r.stderr, "panic:") || strings.Contains(r.stderr, "goroutine "):
		return "it crashed: " + clipped([]byte(r.stderr))
	case !slices.Contains([]int{0, 1, 2}, r.exit):
		return "it exited " + strconv.Itoa(r.exit) + ", outside the exit table"
	case wordsRefused.MatchString(r.stderr):
		return "its words were refused: " + strings.TrimSpace(r.stderr)
	}
	return ""
}

// invariant: rule examples-run (audit W10): every example, run by a script
// over the example world, is answered: done, failed, or refused by what the
// world holds or by the terminal it lacks, never crashed nor refused for its
// own words.
func ruleExamplesRun(t *testing.T, root *command.Command) []finding {
	examples, _ := examplesOf(root)
	bin := specBinary(t)
	var mu sync.Mutex
	var out []finding
	t.Run("runs", func(t *testing.T) {
		for _, e := range examples {
			t.Run(strings.Join(e.words, "_"), func(t *testing.T) {
				t.Parallel()
				if why := runRefusal(runOverWorld(t, bin, e.words)); why != "" {
					mu.Lock()
					out = append(out, finding{"examples-run", e.at + ": " + e.line, "an example that does not run: " + why})
					mu.Unlock()
				}
			})
		}
	})
	return out
}

func TestEveryTypedCommandHasAnExample(t *testing.T) {
	t.Parallel()
	t.Run("holds", func(t *testing.T) { excuse(t, ruleExamples(commandTree(t)), nil) })
	t.Run("bites", func(t *testing.T) {
		root := commandTree(t)
		c, _, _ := root.Find([]string{"machines", "start"})
		c.Example = ""
		bites(t, "examples", ruleExamples(root), "machines start")
	})
}

func TestEveryTopCommandStandsInAHelpGroup(t *testing.T) {
	t.Parallel()
	t.Run("holds", func(t *testing.T) { excuse(t, ruleHelpGroups(commandTree(t)), nil) })
	t.Run("bites", func(t *testing.T) {
		root := commandTree(t)
		root.AddCommand(&command.Command{Use: "mutant", Run: func(*command.Command, []string) {}})
		bites(t, "help-groups", ruleHelpGroups(root), "mutant")
	})
}

func TestEveryExampleParses(t *testing.T) {
	t.Parallel()
	t.Run("holds", func(t *testing.T) {
		excuse(t, ruleExamplesParse(func() *command.Command { return commandTree(t) }), nil)
	})
	t.Run("bites", func(t *testing.T) {
		for _, mutant := range []struct{ example, where string }{
			{"  station machines start api --no-such-flag", "machines start: station machines start api --no-such-flag"},
			{"  station machines start", "machines start: station machines start"},
			{"  station machines stop api --yes", "machines start: station machines stop api --yes"},
			{"  station machines start <vm>", "machines start: station machines start <vm>"},
			{"  Machines machines start api", "machines start: Machines machines start api"},
		} {
			fresh := func() *command.Command {
				root := commandTree(t)
				c, _, _ := root.Find([]string{"machines", "start"})
				c.Example = mutant.example
				return root
			}
			bites(t, "examples-parse", ruleExamplesParse(fresh), mutant.where)
		}
	})
}

func TestEveryExampleRuns(t *testing.T) {
	t.Parallel()
	t.Run("holds", func(t *testing.T) { excuse(t, ruleExamplesRun(t, commandTree(t)), nil) })
	t.Run("bites", func(t *testing.T) {
		for _, r := range []ran{
			{exit: 2, stderr: "panic: runtime error: index out of range\n\ngoroutine 1 [running]:\n"},
			{exit: 2, stderr: "Error: unknown flag: --nope\nHint: run station machines start --help\n"},
			{exit: 2, stderr: "Error: unknown command \"nope\"\nHint: station help\n"},
			{exit: 3},
			{timedOut: true, exit: -1},
		} {
			if runRefusal(r) == "" {
				t.Errorf("the examples-run rule took this run for an answer: %+v", r)
			}
		}
		for _, r := range []ran{
			{exit: 0},
			{exit: 1, stderr: "Error: 1 of 10 machines failed\nHint: station machines\n"},
			{exit: 2, stderr: "Error: no image \"linux\"\nHint: station images\n"},
			{exit: 2, stderr: "Error: refusing to remove without --yes\nHint: re-run with --yes\n"},
		} {
			if why := runRefusal(r); why != "" {
				t.Errorf("the examples-run rule refused an answer: %s", why)
			}
		}
	})
}
