package architecture_test

import (
	"go/ast"
	"go/types"
	"slices"
	"strings"
	"testing"
)

// value: one exported interface of a contract package, and the methods an
// implementation of it must have.
type contractType struct {
	family, name, where string
	methods             []string
}

// value: one named type of an adapter, and the methods it (or its pointer)
// has.
type adapterType struct {
	family, tech string
	methods      []string
}

func contractOf(pkg string) (string, bool) {
	return topOf(pkg), restOf(pkg) == "contract"
}

func adapterOf(pkg string) (family, tech string, ok bool) {
	tech, ok = strings.CutPrefix(restOf(pkg), "adapter/")
	return topOf(pkg), tech, ok && tech != "" && !strings.Contains(tech, "/")
}

func methodNames(set *types.MethodSet) []string {
	out := make([]string, 0, set.Len())
	for i := range set.Len() {
		out = append(out, set.At(i).Obj().Name())
	}
	return out
}

func typeSpecs(f *file, each func(spec *ast.TypeSpec, obj *types.TypeName)) {
	for _, d := range f.syntax.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range g.Specs {
			spec, ok := s.(*ast.TypeSpec)
			if !ok || spec.TypeParams != nil {
				continue
			}
			if obj, ok := f.info.Defs[spec.Name].(*types.TypeName); ok {
				each(spec, obj)
			}
		}
	}
}

// value: the selector X.Sel of a file whose X names the package at path.
func selectorsOf(f *file, path string, each func(name string)) {
	ast.Inspect(f.syntax, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			if pn, ok := f.info.Uses[id].(*types.PkgName); ok && pn.Imported().Path() == path {
				each(sel.Sel.Name)
			}
		}
		return true
	})
}

// value: what one declaration of a contracttest package names: the contract
// types, and the package's own types it uses.
type suiteDecl struct {
	exported bool
	named    []string
	local    []string
}

func declNames(f *file, n ast.Node, contractPath string) suiteDecl {
	var d suiteDecl
	ast.Inspect(n, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := n.X.(*ast.Ident); ok {
				if pn, ok := f.info.Uses[id].(*types.PkgName); ok && pn.Imported().Path() == contractPath {
					d.named = append(d.named, n.Sel.Name)
				}
			}
		case *ast.Ident:
			if tn, ok := f.info.Uses[n].(*types.TypeName); ok && tn.Pkg() != nil && tn.Pkg().Path() == module+"/"+f.pkg {
				d.local = append(d.local, tn.Name())
			}
		}
		return true
	})
	return d
}

func receiverName(e ast.Expr) string {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	if id, ok := e.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// value: by family, the contract types each exported suite of its
// contracttest names, directly or through the package's own types (a
// Subject struct holding the contract).
func suitesOf(w *world) map[string]map[string][]string {
	decls := map[string]map[string]suiteDecl{}
	for _, f := range w.files {
		if f.test || f.pkg != topOf(f.pkg)+"/contract/contracttest" {
			continue
		}
		fam, path := topOf(f.pkg), module+"/"+topOf(f.pkg)+"/contract"
		if decls[fam] == nil {
			decls[fam] = map[string]suiteDecl{}
		}
		for _, d := range f.syntax.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				sd := declNames(f, d, path)
				if d.Recv == nil {
					sd.exported = d.Name.IsExported()
					decls[fam][d.Name.Name] = sd
					continue
				}
				// value: a method's names are its receiver type's.
				recv := receiverName(d.Recv.List[0].Type)
				held := decls[fam][recv]
				held.named, held.local = append(held.named, sd.named...), append(held.local, sd.local...)
				decls[fam][recv] = held
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						sd, held := declNames(f, ts, path), decls[fam][ts.Name.Name]
						sd.named, sd.local = append(sd.named, held.named...), append(sd.local, held.local...)
						decls[fam][ts.Name.Name] = sd
					}
				}
			}
		}
	}
	out := map[string]map[string][]string{}
	for fam, byName := range decls {
		out[fam] = map[string][]string{}
		for name, d := range byName {
			if !d.exported {
				continue
			}
			seen, named := map[string]bool{name: true}, slices.Clone(d.named)
			for queue := slices.Clone(d.local); len(queue) > 0; queue = queue[1:] {
				if seen[queue[0]] {
					continue
				}
				seen[queue[0]] = true
				named = append(named, byName[queue[0]].named...)
				queue = append(queue, byName[queue[0]].local...)
			}
			for _, c := range named {
				if !slices.Contains(out[fam][c], name) {
					out[fam][c] = append(out[fam][c], name)
				}
			}
		}
	}
	return out
}

func covers(a adapterType, c contractType) bool {
	return a.family == c.family && !slices.ContainsFunc(c.methods, func(m string) bool { return !slices.Contains(a.methods, m) })
}

// invariant: rule contracts-are-whole (ARCHITECTURE 13.3, AR-01): every
// exported interface of a contract package has a real adapter and a fake in
// its family's adapter/fake, and a suite of its contracttest that names it
// is run by both.
func ruleContracts(w *world) []finding {
	var contracts []contractType
	var adapters []adapterType
	suites := suitesOf(w)
	ran := map[string]map[bool]bool{}
	for _, f := range w.files {
		family, isContract := contractOf(f.pkg)
		if !f.test && isContract {
			typeSpecs(f, func(spec *ast.TypeSpec, obj *types.TypeName) {
				iface, ok := obj.Type().Underlying().(*types.Interface)
				if !ok || !obj.Exported() || iface.NumMethods() == 0 {
					return
				}
				contracts = append(contracts, contractType{family: family, name: obj.Name(), where: f.at(spec.Pos()), methods: methodNames(types.NewMethodSet(iface))})
			})
		}
		if fam, tech, ok := adapterOf(f.pkg); ok && !f.test {
			typeSpecs(f, func(_ *ast.TypeSpec, obj *types.TypeName) {
				if _, isIface := obj.Type().Underlying().(*types.Interface); isIface {
					return
				}
				adapters = append(adapters, adapterType{family: fam, tech: tech, methods: methodNames(types.NewMethodSet(types.NewPointer(obj.Type())))})
			})
		}
		if fam, tech, ok := adapterOf(f.pkg); ok && f.test {
			selectorsOf(f, module+"/"+fam+"/contract/contracttest", func(name string) {
				key := fam + "/" + name
				if ran[key] == nil {
					ran[key] = map[bool]bool{}
				}
				ran[key][tech == "fake"] = true
			})
		}
	}
	var out []finding
	for _, c := range contracts {
		hasReal := slices.ContainsFunc(adapters, func(a adapterType) bool { return a.tech != "fake" && covers(a, c) })
		fake := slices.ContainsFunc(adapters, func(a adapterType) bool { return a.tech == "fake" && covers(a, c) })
		var byReal, byFake bool
		for _, suite := range suites[c.family][c.name] {
			byReal = byReal || ran[c.family+"/"+suite][false]
			byFake = byFake || ran[c.family+"/"+suite][true]
		}
		switch {
		case !hasReal:
			out = append(out, finding{"contracts-are-whole", c.where, c.name + " has no real adapter in apps/engine/" + c.family + "/adapter/"})
		case !fake:
			out = append(out, finding{"contracts-are-whole", c.where, c.name + " has no fake in apps/engine/" + c.family + "/adapter/fake"})
		case len(suites[c.family][c.name]) == 0:
			out = append(out, finding{"contracts-are-whole", c.where, c.name + " has no suite in apps/engine/" + c.family + "/contract/contracttest"})
		case !byReal || !byFake:
			out = append(out, finding{"contracts-are-whole", c.where, c.name + "'s suite is not run by both a real adapter and the fake"})
		}
	}
	return out
}

func TestContractsAreWhole(t *testing.T) { judge(t, "ContractsAreWhole") }
