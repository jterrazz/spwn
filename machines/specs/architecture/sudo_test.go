package architecture_test

import (
	"fmt"
	"go/ast"
	"slices"
	"testing"
)

// invariant: rule no-sudo-literal (SC-03, decision O3): station runs sudo
// inside a VM it owns alone, as that VM's own admin (its first boot, its
// image build, its poweroff), and on another Mac only for the planned
// restart's fdesetup authrestart, run on the owner's explicit action, its
// password asked by macOS at the terminal and never stored; a plan reads
// whether a root line asks a password by `sudo -k -n true`, which never
// prompts and runs only true; a check never runs sudo, it names the line
// for the owner to type (decision 17). Every
// such line is composed with catalog.AsRoot, every hint's in a home of
// values: the word is spelled nowhere else.
func ruleNoSudoLiteral(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.test || valuesHome(f.pkg) {
			continue
		}
		literals(f, func(lit *ast.BasicLit, value string) {
			if slices.Contains(splitWords(value), "sudo") {
				out = append(out, finding{"no-sudo-literal", f.at(lit.Pos()), fmt.Sprintf("%q spells sudo: a root line is catalog.AsRoot's, a hint surface/cli/words'", f.pkg)})
			}
		})
	}
	return out
}

func TestNoSudoLiteral(t *testing.T) { judge(t, "NoSudoLiteral") }
