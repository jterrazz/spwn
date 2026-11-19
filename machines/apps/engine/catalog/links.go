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
	return c.links(it.Name, it.Requires, it.RequiresOneOf, providerOf(it, method, sys), method)
}

func (c Catalog) InstallLinks(t App, in Install) []Link {
	// value: Links for an app by the install this machine carries: its
	// provider is that install's, its manager that of the install's method.
	var by string
	if p, ok := in.(ProvidedByInstall); ok {
		by = p.Package
	}
	var method string
	if in != nil {
		method = string(in.Method())
	}
	return c.links(t.Name, t.Requires, nil, by, method)
}

func (c Catalog) links(name string, requires, oneOf []string, by, method string) []Link {
	var out []Link
	for _, r := range requires {
		out = append(out, Link{Kind: LinkRequires, Packages: []string{r}})
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

// invariant: the provider is the one of the package's method by that key
// for this system, else of its first method by that key.
func providerOf(it Package, method string, sys System) string {
	by := ""
	for _, m := range it.Methods {
		if m.ProvidedBy == "" || m.Method() != method {
			continue
		}
		if m.Matches(sys) {
			return m.ProvidedBy
		}
		if by == "" {
			by = m.ProvidedBy
		}
	}
	return by
}
