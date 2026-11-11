package architecture_test

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"slices"
	"strings"
	"testing"
)

// invariant: rule one-writer-per-store, ARCHITECTURE 12 and 13.3 rule 7: each
// store and each preference group has one owner, the calls package of its
// family; only it writes the store (Put, Delete, Set, Write), only wire
// builds a store's adapter, and each group is written by one contract.
func storeOwners() map[string]string {
	src := module + "/src/"
	return map[string]string{
		src + "core/contract.Preferences":           "core/contract",
		src + "setup/contract.NamePrefs":            "setup/calls",
		src + "setup/contract.IdentityPrefs":        "setup/calls",
		src + "machines/contract.MacsPrefs":         "machines/calls",
		src + "machines/contract.VMUserPrefs":       "machines/calls",
		src + "accounts/contract.ShareListPrefs":    "accounts/calls",
		src + "ports/contract.PortsPrefs":           "ports/calls",
		src + "settings/contract.OffHerePrefs":      "settings/calls",
		src + "machines/contract.MachineStore":      "machines/calls",
		src + "machines/contract.UptimeStore":       "machines/calls",
		src + "images/contract.ImageStore":          "images/calls",
		src + "accounts/contract.ShareStore":        "accounts/calls",
		src + "ports/contract.ReservationStore":     "ports/calls",
		src + "ports/contract.DeclarationStore":     "ports/calls",
		src + "settings/contract.AppliedStore":      "settings/calls",
		src + "agents/contract.Journal":             "agents/calls",
		src + "machinefile/contract.Ledger":         "machinefile/calls",
		src + "engine.Reservations":                 "ports/calls",
		src + "machines/adapter/jsonfile.Store":     "machines/calls",
		src + "machines/adapter/jsonfile.Uptime":    "machines/calls",
		src + "images/adapter/jsonfile.Store":       "images/calls",
		src + "accounts/adapter/jsonfile.Shares":    "accounts/calls",
		src + "ports/adapter/jsonfile.Reservations": "ports/calls",
		src + "ports/adapter/jsonfile.Declarations": "ports/calls",
		src + "settings/adapter/jsonfile.Applied":   "settings/calls",
		src + "agents/adapter/jsonfile.Journal":     "agents/calls",
		src + "machinefile/adapter/jsonfile.Ledger": "machinefile/calls",
		src + "core/adapter/jsonfile.Preferences":   "core/contract",
	}
}

func storeWrite(name string) bool {
	return slices.Contains([]string{"Put", "Delete", "Set", "Write"}, name)
}

func storeOf(f *file, sel *ast.SelectorExpr) (string, bool) {
	t := f.info.TypeOf(sel.X)
	if t == nil {
		return "", false
	}
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	name := types.TypeString(t, nil)
	_, ok := storeOwners()[name]
	return name, ok
}

// invariant: a store's own packages write it freely: its contract (the typed
// groups over Preferences), its adapters and its suite.
func ownStorePackage(pkg, store string) bool {
	owner := strings.TrimPrefix(store, module+"/src/")
	family := topOf(owner)
	return pkg == family+"/contract" || pkg == family+"/contract/contracttest" || strings.HasPrefix(pkg, family+"/adapter/")
}

// invariant: Preferences is written raw only by the typed groups, each in its
// owner's contract package.
func mayWrite(pkg, store string) bool {
	if store == preferences {
		return restOf(pkg) == "contract" || ownStorePackage(pkg, store)
	}
	return pkg == storeOwners()[store] || ownStorePackage(pkg, store)
}

const preferences = module + "/src/core/contract.Preferences"

func ruleOneWriterPerStore(w *world) []finding {
	var out []finding
	groups := map[string]string{}
	for _, f := range w.files {
		if f.test {
			continue
		}
		ast.Inspect(f.syntax, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if obj, isFunc := f.info.Uses[sel.Sel].(*types.Func); isFunc && strings.HasPrefix(sel.Sel.Name, "New") && obj.Pkg() != nil &&
				strings.HasSuffix(obj.Pkg().Path(), "/adapter/jsonfile") && f.pkg != "wire" && !strings.HasSuffix(f.pkg, "/adapter/jsonfile") {
				out = append(out, finding{"one-writer-per-store", f.at(call.Pos()), fmt.Sprintf("%q builds a store: only wire constructs one", f.pkg)})
			}
			if !storeWrite(sel.Sel.Name) {
				return true
			}
			store, ok := storeOf(f, sel)
			if !ok {
				return true
			}
			if store == preferences && sel.Sel.Name == "Write" && restOf(f.pkg) == "contract" && topOf(f.pkg) != "core" && len(call.Args) > 0 {
				if tv, ok := f.info.Types[call.Args[0]]; ok && tv.Value != nil && tv.Value.Kind() == constant.String {
					group := constant.StringVal(tv.Value)
					if other, taken := groups[group]; taken && other != f.pkg {
						out = append(out, finding{"one-writer-per-store", f.at(call.Pos()), fmt.Sprintf("the preference group %q is written by %q and %q: one owner per group", group, other, f.pkg)})
					}
					groups[group] = f.pkg
				}
			}
			if mayWrite(f.pkg, store) {
				return true
			}
			out = append(out, finding{"one-writer-per-store", f.at(call.Pos()), fmt.Sprintf("%q writes %s: only %s does", f.pkg, strings.TrimPrefix(store, module+"/src/"), storeOwners()[store])})
			return true
		})
	}
	return out
}

func TestOneWriterPerStore(t *testing.T)      { judge(t, "OneWriterPerStore") }
func TestOneWriterPerStoreBuilt(t *testing.T) { judge(t, "OneWriterPerStoreBuilt") }
func TestOneWriterPerStoreGroup(t *testing.T) { judge(t, "OneWriterPerStoreGroup") }
