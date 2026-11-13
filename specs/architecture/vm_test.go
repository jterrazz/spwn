package architecture_test

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// invariant: rule one-way-into-a-vm, ARCHITECTURE 13.3 rule 11: a VM is
// reached through its engine's one Guest connection alone. No code runs ssh
// but the two adapters that reach a Mac, no code runs scp at all, and only the
// engine (its dialer handed by wire's engineGuests) and the contract's own
// adapters and suite dial a Guest.
func sshExecOwners() []string { return []string{"remote/adapter/sshstdio", "machines/adapter/macos"} }

const guestDialer = module + "/machines/contract.GuestDialer"

func dialsAGuest(f *file, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Dial" {
		return false
	}
	t := f.info.TypeOf(sel.X)
	if t == nil {
		return false
	}
	return types.TypeString(t, nil) == guestDialer
}

func mayDialAGuest(f *file) bool {
	switch {
	case f.pkg == "engine" || strings.HasPrefix(f.pkg, "engine/"):
		return true
	case f.rel == "apps/engine/wire/guest_dialer.go":
		return true
	case strings.HasPrefix(f.pkg, "machines/adapter/") || f.pkg == "machines/contract/contracttest":
		return true
	}
	return false
}

// invariant: a literal runs a tool when it names it as a command's Name, as
// an argv's first word (a slice literal's, or the first an append adds), or
// as exec.Command's first argument.
func execNames(f *file, visit func(at, tool string)) {
	tool := func(e ast.Expr) (string, bool) {
		lit, ok := e.(*ast.BasicLit)
		if !ok {
			return "", false
		}
		v, err := strconv.Unquote(lit.Value)
		return v, err == nil && (v == "ssh" || v == "scp")
	}
	ast.Inspect(f.syntax, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.KeyValueExpr:
			if k, ok := x.Key.(*ast.Ident); ok && k.Name == "Name" {
				if v, ok := tool(x.Value); ok {
					visit(f.at(x.Pos()), v)
				}
			}
		case *ast.CompositeLit:
			if len(x.Elts) > 0 {
				if v, ok := tool(x.Elts[0]); ok {
					visit(f.at(x.Pos()), v)
				}
			}
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Command" && len(x.Args) > 0 {
				if v, ok := tool(x.Args[0]); ok {
					visit(f.at(x.Pos()), v)
				}
			}
			if id, ok := x.Fun.(*ast.Ident); ok && id.Name == "append" && len(x.Args) > 1 {
				if v, ok := tool(x.Args[1]); ok {
					visit(f.at(x.Pos()), v)
				}
			}
		}
		return true
	})
}

func ruleOneWayIntoAVM(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		owner := slices.Contains(sshExecOwners(), f.pkg)
		execNames(f, func(at, tool string) {
			switch {
			case tool == "scp":
				out = append(out, finding{"one-way-into-a-vm", at, "scp runs nowhere: a file reaches a VM by Guest.Put"})
			case owner:
			default:
				out = append(out, finding{"one-way-into-a-vm", at, fmt.Sprintf("%q runs ssh: a VM is reached by its engine's Guest, a Mac by remote/adapter/sshstdio", f.pkg)})
			}
		})
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !dialsAGuest(f, call) || mayDialAGuest(f) {
				return true
			}
			out = append(out, finding{"one-way-into-a-vm", f.at(call.Pos()), fmt.Sprintf("%q dials a Guest: only the engine holds a VM's connection", f.pkg)})
			return true
		})
	}
	return out
}

func TestOneWayIntoAVM(t *testing.T)     { judge(t, "OneWayIntoAVM") }
func TestOneWayIntoAVMSCP(t *testing.T)  { judge(t, "OneWayIntoAVMSCP") }
func TestOneWayIntoAVMDial(t *testing.T) { judge(t, "OneWayIntoAVMDial") }
