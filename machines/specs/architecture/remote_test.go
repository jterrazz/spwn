package architecture_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strconv"
	"testing"
)

// invariant: the only command lines that start station elsewhere, each the
// plumbing its owner serves or hops by: another Mac's engine API, a VM's
// guest API and its ports reporter, and the owner's own ssh to a VM of
// another Mac. No other code builds `station <words>` to run elsewhere.
func stationLaunchers() map[string]string {
	return map[string]string{
		"remote/adapter/sshstdio":  "station engine attach",
		"remote/adapter/unix":      "station guest serve --stdio",
		"engine":                   "station guest",
		"machines/adapter/openssh": "station machines pipe, the owner's ProxyCommand",
	}
}

var launchesStation = regexp.MustCompile(`exec station\b|/station [a-z]`)

// value: an argv built of separate words that runs station past ssh's `--`
// or by ssh itself: {"ssh", host, "--", "station", "apps"}.
func argvLaunchesStation(words []string) bool {
	return slices.Contains(words, "station") && (slices.Contains(words, "--") || slices.Contains(words, "ssh"))
}

func stringWords(exprs []ast.Expr) []string {
	var out []string
	for _, e := range exprs {
		if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if v, err := strconv.Unquote(lit.Value); err == nil {
				out = append(out, v)
			}
		}
	}
	return out
}

// invariant: rule no-remote-reexec, ARCHITECTURE 13.3 rule 12: --on is a
// Router target, the fleet a fan-out of calls; nothing relays station's
// text from another machine.
func ruleNoRemoteReexec(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test || stationLaunchers()[f.pkg] != "" {
			continue
		}
		found := func(at token.Pos) {
			out = append(out, finding{"no-remote-reexec", f.at(at), fmt.Sprintf("%q starts station elsewhere: a call reaches another machine through the Router", f.pkg)})
		}
		literals(f, func(lit *ast.BasicLit, value string) {
			if launchesStation.MatchString(value) {
				found(lit.Pos())
			}
		})
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CompositeLit:
				if argvLaunchesStation(stringWords(n.Elts)) {
					found(n.Pos())
				}
			case *ast.CallExpr:
				if argvLaunchesStation(stringWords(n.Args)) {
					found(n.Pos())
				}
			}
			return true
		})
	}
	return out
}

func TestNoRemoteReexec(t *testing.T) { judge(t, "NoRemoteReexec") }
