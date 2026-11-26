package architecture_test

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/approvals"
)

func approvalChecks() []string {
	// value: what reads or asks a macOS approval: the mDNS group Local
	// Network is read on, the datagram that reads it, and the readers of
	// the other consents macOS gives (notifications, the login items' state,
	// Touch ID).
	return []string{"224.0.0.251", "SendUDP", "SMAppServiceStatus", "UNUserNotificationCenter", "requestAuthorization", "canEvaluatePolicy"}
}

func approvalHolders(rel, word string) bool {
	// value: the approvals family owns every check; platform/local defines the
	// datagram it sends, and the cgo files the readers it will move in.
	if strings.HasPrefix(rel, "apps/engine/approvals/") {
		return true
	}
	if word == "SendUDP" && strings.HasPrefix(rel, "apps/engine/platform/local/") {
		return true
	}
	return slices.Contains([]string{"apps/engine/platform/darwin/agents_darwin.go", "apps/engine/platform/darwin/presence_darwin.go"}, rel)
}

// invariant: rule 24: a macOS approval is read or asked by the approvals
// family alone: outside it, no code under apps/engine/ names the mDNS
// group, its datagram or a consent's reader, so a second check cannot grow beside the
// table.
func ruleApprovalsOwned(w *world) []finding {
	var out []finding
	report := func(rel, where, text string) {
		for _, word := range approvalChecks() {
			if strings.Contains(text, word) && !approvalHolders(rel, word) {
				out = append(out, finding{"approvals", where, word + " is the approvals family's to check"})
			}
		}
	}
	for _, f := range w.files {
		if f.test || !strings.HasPrefix(f.rel, "apps/engine/") {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BasicLit:
				if n.Kind == token.STRING {
					report(f.rel, f.at(n.Pos()), n.Value)
				}
			case *ast.Ident:
				report(f.rel, f.at(n.Pos()), n.Name)
			}
			return true
		})
	}
	for _, rel := range w.cgo {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(w.root, rel))
		if err == nil {
			report(rel, rel, string(data))
		}
	}
	return out
}

func TestApprovalsOwned(t *testing.T) { judge(t, "ApprovalsOwned") }

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
