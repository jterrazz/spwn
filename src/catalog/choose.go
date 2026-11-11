package catalog

import (
	"cmp"
	"slices"
)

func (m MethodSpec) Method() string {
	// value: the method's word as station runs it, reads it back and
	// records it: an app's install method (a `script:` that runs a line is
	// `shell`, one that only hints is `app`), a setting's key.
	if in := m.install(nil); in != nil {
		return string(in.Method())
	}
	return m.Key()
}

func (m MethodSpec) Runs(k Kind) bool {
	// value: whether station runs this method for an item of that kind: a
	// setting's every method, an app's or a manager's only the ones it has
	// an installer for (github-release waits for its own).
	return k == KindSetting || m.install(nil) != nil
}

// value: a method's standing in the order one system tries an item's
// methods: listed by prefer (its place there), ranked by its manager on
// that system (its rank), any other manager, then a method no manager
// runs (a script, a provider).

type Standing struct {
	Preferred int
	Class     int
	Rank      int
}

const (
	ClassRanked = iota
	ClassManaged
	ClassUnmanaged
)

func (c Catalog) Standing(m MethodSpec, os OS, prefer []ManagerKey) Standing {
	key := m.Key()
	s := Standing{Preferred: len(prefer), Class: ClassUnmanaged}
	if i := slices.Index(prefer, ManagerKey(key)); i >= 0 {
		s.Preferred = i
	}
	if owner, ok := c.Manager(key); ok {
		if it, _ := c.Item(owner); it.Item.Priority != nil {
			if r := it.Item.Priority.on(os); r > 0 {
				s.Class, s.Rank = ClassRanked, r
				return s
			}
		}
	}
	if slices.Contains(ManagerKey("").Enum(), key) {
		s.Class = ClassManaged
	}
	return s
}

func (c Catalog) Ordered(methods []MethodSpec, os OS, prefer []ManagerKey) []MethodSpec {
	// invariant: the one order every reader chooses a method by, the
	// resolution of a Machinefile and the apps a machine carries alike:
	// prefer's keys first, then the system's priority, then any other
	// manager, then the rest; a tie keeps the file's order.
	out := slices.Clone(methods)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && c.Standing(out[j], os, prefer).before(c.Standing(out[j-1], os, prefer)); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func (s Standing) before(o Standing) bool {
	return cmp.Or(cmp.Compare(s.Preferred, o.Preferred), cmp.Compare(s.Class, o.Class), cmp.Compare(s.Rank, o.Rank)) < 0
}

func (p Priority) on(os OS) int {
	switch os {
	case OSMacOS:
		return p.MacOS
	case OSUbuntu:
		return p.Ubuntu
	case OSDebian:
		return p.Debian
	case OSAny:
	}
	return 0
}

// value: the system a kind of machine is, as the apps it carries are
// placed on it: a Mac and a macOS VM run macOS, a Linux VM Ubuntu.
func (k MachineKind) os() OS {
	if k == Linux {
		return OSUbuntu
	}
	return OSMacOS
}

// invariant: an app's installs on each kind of machine are its methods
// that run there, in the order Ordered gives them on that kind's system,
// so the install a machine carries by default is the one a Machinefile's
// resolution chooses there when it asks no manager.
func (c Catalog) placed(it Item) []LivesOn {
	var out []LivesOn
	for _, k := range []MachineKind{Mac, Linux, MacOS} {
		var on []MethodSpec
		for _, m := range it.Methods {
			if m.Runs(it.Kind) && slices.Contains(m.kinds(), k) {
				on = append(on, m)
			}
		}
		for _, m := range c.Ordered(on, k.os(), nil) {
			out = append(out, LivesOn{Kinds: []MachineKind{k}, Install: m.install(it.Provides)})
		}
	}
	return out
}
