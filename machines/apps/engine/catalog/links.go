package catalog

import "slices"

// value: how a package installed by one method leans on another: it
// requires it, it requires one of a group (a setting's), its method is
// provided by it, or its method's manager is it.

type LinkKind string

const (
	LinkRequires      LinkKind = "requires"
	LinkRequiresOneOf LinkKind = "requires-one-of"
	LinkProvidedBy    LinkKind = "provided-by"
	LinkManager       LinkKind = "manager"
)

// value: one link of a package: its kind and the packages it names, any one of
// which serves; every kind but requires-one-of names one.

type Link struct {
	Kind     LinkKind
	Packages []string
}

func (l Link) Names(pkg string) bool {
	// value: whether the link leans on that package.
	return slices.Contains(l.Packages, pkg)
}

func (c Catalog) Links(it Package, method string, sys System) []Link {
	// invariant: the one list of what a package by a method leans on, read by
	// the resolution, which brings a package in and orders it, by what keeps
	// an extra another one there still needs, and, through InstallLinks,
	// by apps.install; a package never leans on itself.
	m := chosen(it, method, sys)
	return c.links(it.Name, slices.Concat(it.Requires, m.Requires), it.RequiresOneOf, m.ProvidedBy, method)
}

func (c Catalog) InstallLinks(t App, in Install, k MachineKind) []Link {
	// value: Links for an app by the install this kind of machine carries:
	// its provider is that install's, its manager that of the install's
	// method, and its method's requires those of the package's method
	// that install is on that kind.
	var by string
	if p, ok := in.(ProvidedByInstall); ok {
		by = p.Package
	}
	var method string
	if in != nil {
		method = string(in.Method())
	}
	return c.links(t.Name, slices.Concat(t.Requires, c.placedMethod(t.Name, method, k).Requires), nil, by, method)
}

func (c Catalog) links(name string, requires, oneOf []string, by, method string) []Link {
	var out []Link
	var said []string
	for _, r := range requires {
		if r != name && !slices.Contains(said, r) {
			said = append(said, r)
			out = append(out, Link{Kind: LinkRequires, Packages: []string{r}})
		}
	}
	if len(oneOf) > 0 {
		out = append(out, Link{Kind: LinkRequiresOneOf, Packages: slices.Clone(oneOf)})
	}
	if by != "" && by != name {
		out = append(out, Link{Kind: LinkProvidedBy, Packages: []string{by}})
	}
	if owner, ok := c.Manager(method); ok && owner != name {
		out = append(out, Link{Kind: LinkManager, Packages: []string{owner}})
	}
	return out
}

func (c Catalog) ManagerKeys() []string {
	// value: every method key a manager package of the catalog installs by,
	// in the catalog's order: the words `--to` and `--manager` take.
	var out []string
	for _, p := range c.Packages {
		for _, k := range p.Package.Manages {
			out = append(out, string(k))
		}
	}
	return out
}

func (c Catalog) Manager(method string) (string, bool) {
	// value: the package that manages a method key, when one does.
	for _, p := range c.Packages {
		if slices.Contains(p.Package.Manages, ManagerKey(method)) {
			return p.Package.Name, true
		}
	}
	return "", false
}

// invariant: the method a package is carried by is its method by that key
// for this system, else its first method by that key; its provider and
// its own requires are that method's.
func chosen(it Package, method string, sys System) MethodSpec {
	var first *MethodSpec
	for i, m := range it.Methods {
		if m.Method() != method {
			continue
		}
		if m.Matches(sys) {
			return m
		}
		if first == nil {
			first = &it.Methods[i]
		}
	}
	if first == nil {
		return MethodSpec{}
	}
	return *first
}

// invariant: the method an install of the app stands for on a kind of
// machine is the first by its key in the order that kind places the
// package's methods by, the order its installs there come from.
func (c Catalog) placedMethod(name, method string, k MachineKind) MethodSpec {
	p, ok := c.Package(name)
	if !ok {
		return MethodSpec{}
	}
	var on []MethodSpec
	for _, m := range p.Package.Methods {
		if slices.Contains(m.kinds(), k) {
			on = append(on, m)
		}
	}
	for _, m := range c.Ordered(on, k.os(), nil) {
		if m.Method() == method {
			return m
		}
	}
	return MethodSpec{}
}
