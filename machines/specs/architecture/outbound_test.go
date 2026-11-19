package architecture_test

import (
	"fmt"
	"go/ast"
	"go/token"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// value: a host on the internet, bare or in a URL: a name ending in a
// public top-level domain.
var hostLiteral = regexp.MustCompile(`(^|[/@\s:])([a-z0-9-]+\.)+(com|net|org|io|me|dev|sh|co|app|ai|info|so|xyz|ip)(\b|$)`)

// invariant: the packages a read of a machine runs through: the Overview
// and the adapters of the contracts it reads.
func readPath(pkg string) bool {
	switch {
	case topOf(pkg) == "overview", topOf(pkg) == "activity":
		return true
	case strings.HasPrefix(pkg, "settings/adapter/"), strings.HasPrefix(pkg, "background/adapter/"):
		return true
	}
	return slices.Contains([]string{"ports/adapter/lsof", "ports/adapter/procnet", "awake/adapter/macos", "disk/adapter/apfs", "disk/adapter/posix"}, pkg)
}

// value: a literal that names a host and is never reached: a document
// type's name.
func outboundExempt(value string) bool {
	return strings.Contains(value, "www.apple.com/DTDs/")
}

// invariant: rule no-outbound-literals (13.3 rule 16, SC-09): a read of a
// machine calls no third-party service, so no read path spells a host.
func ruleNoOutboundLiterals(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test || !readPath(f.pkg) {
			continue
		}
		imported := map[token.Pos]bool{}
		for _, spec := range f.syntax.Imports {
			imported[spec.Path.Pos()] = true
		}
		literals(f, func(lit *ast.BasicLit, value string) {
			if imported[lit.Pos()] || outboundExempt(value) {
				return
			}
			if host := hostLiteral.FindString(value); host != "" {
				out = append(out, finding{"no-outbound-literals", f.at(lit.Pos()), fmt.Sprintf("%q spells the host %q: a read calls no third-party service", f.pkg, strings.TrimLeft(host, "/@: \t"))})
			}
		})
	}
	return out
}

func TestNoOutboundLiterals(t *testing.T) { judge(t, "NoOutboundLiterals") }
