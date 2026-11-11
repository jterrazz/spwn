package catalog

import "slices"

// value: how an item installed by one method leans on another: it
// requires it, it requires one of a group (a setting's), its method is
// provided by it, or its method's manager is it.

type LinkKind string

const (
	LinkRequires      LinkKind = "requires"
	LinkRequiresOneOf LinkKind = "requires-one-of"
	LinkProvidedBy    LinkKind = "provided-by"
	LinkManager       LinkKind = "manager"
)

// value: one link of an item: its kind and the items it names, any one of
// which serves; every kind but requires-one-of names one.

type Link struct {
	Kind  LinkKind
	Items []string
}

func (l Link) Names(item string) bool {
	// value: whether the link leans on that item.
	return slices.Contains(l.Items, item)
}

func (c Catalog) Links(it Item, method string, sys System) []Link {
	// invariant: the one list of what an item by a method leans on, read by
	// what brings an item in, what orders it, and what keeps an item
	// another one there still needs; an item never leans on itself.
	var out []Link
	for _, r := range it.Requires {
		out = append(out, Link{Kind: LinkRequires, Items: []string{r}})
	}
	if len(it.RequiresOneOf) > 0 {
		out = append(out, Link{Kind: LinkRequiresOneOf, Items: slices.Clone(it.RequiresOneOf)})
	}
	if by := providerOf(it, method, sys); by != "" && by != it.Name {
		out = append(out, Link{Kind: LinkProvidedBy, Items: []string{by}})
	}
	if owner, ok := c.Manager(method); ok && owner != it.Name {
		out = append(out, Link{Kind: LinkManager, Items: []string{owner}})
	}
	return out
}

func (c Catalog) Manager(method string) (string, bool) {
	// value: the item that manages a method key, when one does.
	for _, p := range c.Items {
		if slices.Contains(p.Item.Manages, ManagerKey(method)) {
			return p.Item.Name, true
		}
	}
	return "", false
}

// invariant: the provider is the one of the item's method by that key
// for this system, else of its first method by that key.
func providerOf(it Item, method string, sys System) string {
	by := ""
	for _, m := range it.Methods {
		if m.ProvidedBy == "" || m.Key() != method {
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
