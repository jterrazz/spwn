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
	// the resolution, which brings an item in and orders it, by what keeps
	// an extra another one there still needs, and, through InstallLinks,
	// by apps.install; an item never leans on itself.
	return c.links(it.Name, it.Requires, it.RequiresOneOf, providerOf(it, method, sys), method)
}

func (c Catalog) InstallLinks(t App, in Install) []Link {
	// value: Links for an app by the install this machine carries: its
	// provider is that install's, its manager that of the install's method.
	var by string
	if p, ok := in.(ProvidedByInstall); ok {
		by = p.Item
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
		out = append(out, Link{Kind: LinkRequires, Items: []string{r}})
	}
	if len(oneOf) > 0 {
		out = append(out, Link{Kind: LinkRequiresOneOf, Items: slices.Clone(oneOf)})
	}
	if by != "" && by != name {
		out = append(out, Link{Kind: LinkProvidedBy, Items: []string{by}})
	}
	if owner, ok := c.Manager(method); ok && owner != name {
		out = append(out, Link{Kind: LinkManager, Items: []string{owner}})
	}
	return out
}

func (c Catalog) ManagerKeys() []string {
	// value: every method key a manager item of the catalog installs by,
	// in the catalog's order: the words `--to` and `--manager` take.
	var out []string
	for _, p := range c.Items {
		for _, k := range p.Item.Manages {
			out = append(out, string(k))
		}
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

// outside: a station older than the managers as items (v3.0) names a
// manager by its own word, in `apps update --method`, in a fix's `method`
// and in apps.update's Method: each of today's keys it has a manager of,
// by that word (Homebrew's `cask` is its `brew`); a key absent here is
// one it has no manager of. Kept until the hidden --method goes.
var olderManagers = map[string]string{"brew": "brew", "cask": "brew", "npm": "npm", "bun": "bun", "uv": "uv", "mise": "mise", "skill": "skills"}

func OlderManager(key string) (word string, ok bool) {
	// value: the word a v3.0 station reads for a manager key; false for a
	// key it has no manager of.
	word, ok = olderManagers[key]
	return word, ok
}

func ManagerOfOlder(word string) string {
	// value: today's key for a manager word a v3.0 station says (`skills`
	// is `skill`); any other word as it came.
	if _, today := olderManagers[word]; today {
		return word
	}
	for key, then := range olderManagers {
		if then == word {
			return key
		}
	}
	return word
}
