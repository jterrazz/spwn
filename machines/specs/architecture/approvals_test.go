package architecture_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
)

func approvalChecks() []string {
	// value: what reads or asks a macOS approval: the mDNS group Local
	// Network is read on (IPv4 and IPv6), the datagram that reads it, and
	// the readers of the other consents macOS gives (notifications, the
	// login items' state, Touch ID).
	return []string{"224.0.0.251", "ff02::fb", "SendUDP", "SMAppServiceStatus", "UNUserNotificationCenter", "requestAuthorization", "canEvaluatePolicy"}
}

func approvalAllowlist() map[string][]string {
	// invariant: the files outside the family that still hold a check's
	// word, each with the words it holds: the datagram's definition, and the
	// readers of the cgo files the family will move in. The list only
	// shrinks: a file that no longer holds its word fails the rule until its
	// entry is removed.
	return map[string][]string{
		"apps/engine/platform/local/sockets.go":          {"SendUDP"},
		"apps/engine/platform/local/system.go":           {"SendUDP"},
		"apps/engine/platform/darwin/agents_darwin.go":   {"SMAppServiceStatus"},
		"apps/engine/platform/darwin/presence_darwin.go": {"canEvaluatePolicy"},
	}
}

func holdsWord(text, word string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(word))
}

func foldedString(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.ParenExpr:
		return foldedString(e.X)
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		x, okx := foldedString(e.X)
		y, oky := foldedString(e.Y)
		return x + y, okx && oky
	}
	return "", false
}

func isMDNSOctets(args []ast.Expr) bool {
	want := []string{"224", "0", "0", "251"}
	if len(args) != len(want) {
		return false
	}
	for i, a := range args {
		lit, ok := a.(*ast.BasicLit)
		if !ok || lit.Kind != token.INT || lit.Value != want[i] {
			return false
		}
	}
	return true
}

func importsC(f *ast.File) bool {
	for _, spec := range f.Imports {
		if spec.Path.Value == `"C"` {
			return true
		}
	}
	return false
}

// invariant: one file's approval words, whatever its build tags: the
// whole text of a file that imports "C" (the readers live in its preamble),
// and for any other file its string literals (folded across a + of
// literals), its identifiers, and the octets 224, 0, 0, 251 of a net.IPv4
// call or a four-element literal.
func approvalHits(rel, text string) ([]finding, bool) {
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, rel, text, parser.SkipObjectResolution)
	if err != nil {
		return []finding{{"approvals", rel, "does not parse: " + err.Error()}}, false
	}
	var out []finding
	hit := func(where, word string) {
		out = append(out, finding{"approvals", where, word + " is the approvals family's to check"})
	}
	at := func(p token.Pos) string { return rel + ":" + strconv.Itoa(fset.Position(p).Line) }
	if importsC(syntax) {
		for _, word := range approvalChecks() {
			if holdsWord(text, word) {
				hit(rel, word)
			}
		}
	}
	strung := func(p token.Pos, s string) {
		for _, word := range approvalChecks() {
			if holdsWord(s, word) {
				hit(at(p), word)
			}
		}
	}
	ast.Inspect(syntax, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if s, ok := foldedString(n); ok {
				strung(n.Pos(), s)
			}
		case *ast.BinaryExpr:
			if s, ok := foldedString(n); ok {
				strung(n.Pos(), s)
			}
		case *ast.Ident:
			for _, word := range approvalChecks() {
				if holdsWord(n.Name, word) {
					hit(at(n.Pos()), word)
				}
			}
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "IPv4" && isMDNSOctets(n.Args) {
				hit(at(n.Pos()), "224.0.0.251")
			}
		case *ast.CompositeLit:
			if isMDNSOctets(n.Elts) {
				hit(at(n.Pos()), "224.0.0.251")
			}
		}
		return true
	})
	return out, true
}

func approvalFindings(w *world, allow map[string][]string) []finding {
	var out []finding
	rels := make([]string, 0, len(w.raw))
	for rel := range w.raw {
		rels = append(rels, rel)
	}
	slices.Sort(rels)
	for _, rel := range rels {
		if strings.HasPrefix(rel, "apps/engine/approvals/") {
			continue
		}
		hits, _ := approvalHits(rel, w.raw[rel])
		for _, h := range hits {
			if !slices.Contains(allow[rel], approvalWordOf(h.what)) {
				out = append(out, h)
			}
		}
	}
	for rel, words := range allow {
		text, ok := w.raw[rel]
		if !ok {
			out = append(out, finding{"approvals", rel, "is allowlisted and no longer exists: remove its entry"})
			continue
		}
		for _, word := range words {
			if !holdsWord(text, word) {
				out = append(out, finding{"approvals", rel, "is allowlisted for " + word + " and no longer holds it: remove its entry"})
			}
		}
	}
	return out
}

func approvalWordOf(what string) string {
	word, _, _ := strings.Cut(what, " ")
	return word
}

// invariant: rule 24: a macOS approval is read or asked by the approvals
// family alone: no Go file under apps/engine/, whatever its build tags or
// the host's GOOS, names the mDNS group (as a string, a string split by +,
// the octets of net.IPv4 or of a four-element literal, or the IPv6 form
// ff02::fb), the datagram that reads it or a consent's reader, so a second
// check cannot grow beside the table. It is a ban on names, not a proof of
// ownership: it does not follow a value through variables, calls or
// another encoding of the group, it does not see the window's Rust or the
// menu bar's Swift (lots A6 and A10), and an allowlisted file must keep
// holding its word.
func ruleApprovalsOwned(w *world) []finding { return approvalFindings(w, approvalAllowlist()) }

func TestApprovalsOwned(t *testing.T)                   { judge(t, "ApprovalsOwned") }
func TestApprovalsOwnedCgoWhateverTheHost(t *testing.T) { judge(t, "ApprovalsOwnedCgo") }
func TestApprovalsOwnedSplitString(t *testing.T)        { judge(t, "ApprovalsOwnedSplit") }
func TestApprovalsOwnedIPv4Call(t *testing.T)           { judge(t, "ApprovalsOwnedIPv4") }
func TestApprovalsOwnedIPv6(t *testing.T)               { judge(t, "ApprovalsOwnedIPv6") }
func TestApprovalsOwnedIdentifier(t *testing.T)         { judge(t, "ApprovalsOwnedIdent") }

// invariant: the allowlist only shrinks: an allowlisted file that no longer
// holds its word, or no longer exists, fails the rule.
func TestApprovalsAllowlistOnlyShrinks(t *testing.T) {
	clean, _ := worlds(t)
	stale := map[string][]string{
		"apps/engine/platform/local/system.go":    {"SMAppServiceStatus"},
		"apps/engine/platform/local/no_longer.go": {"SendUDP"},
	}
	found := map[string]bool{}
	for _, f := range approvalFindings(clean, stale) {
		found[f.where] = true
	}
	for rel := range stale {
		if !found[rel] {
			t.Errorf("the rule did not report the stale allowlist entry %s", rel)
		}
	}
}

// invariant: every approval of the table tells a story of each state it can
// read in, under approvals.list.
func TestEveryApprovalTellsEachState(t *testing.T) {
	root := repoRoot(t)
	for _, id := range approvals.ID("").Enum() {
		for _, state := range approvals.ID(id).States() {
			story := filepath.Join(root, "specs", "usecases", "approvals", "list", id+"-"+string(state)+".spec.yaml")
			if _, err := os.Stat(story); err != nil {
				t.Errorf("approval %s tells no story of %s: %s", id, state, story)
			}
		}
	}
}
