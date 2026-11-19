package architecture_test

import (
	"go/ast"
	"go/types"
	"slices"
	"testing"
)

func recordAccessors() []string { return []string{"recordLocked", "record", "editRecord"} }

const vmState = module + "/engine/process.vmState"

// invariant: rule vm-record-locked: the engine's tick writes a VM's record
// under its lock while that VM's guest session runs beside it, so the record
// is selected only inside its three accessors, which hold the lock or say by
// their name that the caller does; its name, read by the session without
// the lock, is set once, at the VM's construction, never assigned.
func ruleVMRecordLocked(w *world) []finding {
	var out []finding
	for _, f := range w.files {
		if f.pkg != "engine/process" {
			continue
		}
		for _, decl := range f.syntax.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv != nil && slices.Contains(recordAccessors(), fn.Name.Name) {
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.SelectorExpr:
					if vmStateField(f, n) == "rec" {
						out = append(out, finding{"vm-record-locked", f.at(n.Pos()), "a VM's record is reached through record, recordLocked or editRecord"})
					}
				case *ast.AssignStmt:
					for _, lhs := range n.Lhs {
						if sel, ok := lhs.(*ast.SelectorExpr); ok && vmStateField(f, sel) == "name" {
							out = append(out, finding{"vm-record-locked", f.at(n.Pos()), "a VM's name is set once, where it is built"})
						}
					}
				}
				return true
			})
		}
	}
	return out
}

func vmStateField(f *file, sel *ast.SelectorExpr) string {
	s := f.info.Selections[sel]
	if s == nil || s.Kind() != types.FieldVal {
		return ""
	}
	recv := s.Recv()
	if p, ok := recv.(*types.Pointer); ok {
		recv = p.Elem()
	}
	if types.TypeString(recv, nil) != vmState {
		return ""
	}
	return sel.Sel.Name
}

func TestVMRecordLocked(t *testing.T)     { judge(t, "VMRecordLocked") }
func TestVMRecordLockedName(t *testing.T) { judge(t, "VMRecordLockedName") }
