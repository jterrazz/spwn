package machinefile

import (
	"cmp"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: where one value of a resolution was read: the Machinefile or the
// item that says it, its layer and its file.

type Origin struct {
	Name  string
	Layer Layer
	File  string
}

// value: one value of a resolved Machinefile, by its key (`vm.network`,
// `apps.jq.method`), as text, with where it was read.

type Value struct {
	Key, Text string
	Origin    Origin
}

// value: one item of the plan: what it is, the method that installs or
// sets it (its manager key) and why it was chosen, the version asked, its
// phase and what it asks of the owner, why it is in the plan, and the
// values it is given.

type Step struct {
	Item   string
	Kind   catalog.Kind
	Method string
	Chosen string
	// value: the method chosen, as its file says it.
	Spec    catalog.MethodSpec
	Version string
	Phase   catalog.Phase
	Needs   []catalog.Need
	Why     string
	Values  map[string]catalog.OptionValue
	From    string
	// value: where the step was asked (a Machinefile, or the item that
	// brings it in), and where its item is defined.
	Origin, Defined Origin
}

// value: an item the Machinefile marks optional that has no method for the system.

type Skip struct {
	Item, Why string
	Origin    Origin
}

// value: a Machinefile resolved: its name, the files read weakest first,
// every value with its origin, the steps in the order they run, and the
// optional items skipped.

type Resolved struct {
	Name    string
	Kind    catalog.MachineRole
	System  catalog.System
	Files   []Origin
	Values  []Value
	Steps   []Step
	Skipped []Skip
}

// value: this machine, for what a Machinefile does not say of its system.

type Machine struct {
	Kind   catalog.MachineRole
	System catalog.SystemSpec
}

type merged struct {
	values  map[string]Value
	order   []string
	machine bool
	vm      *Origin
	prefer  []catalog.ManagerKey
	apps    []listed[catalog.AppEntry]
	set     map[string]listed[catalog.SettingEntry]
	setting []string
}

type listed[T any] struct {
	name   string
	entry  T
	origin Origin
}

func (m *merged) put(key, text string, o Origin) {
	if _, ok := m.values[key]; !ok {
		m.order = append(m.order, key)
	}
	m.values[key] = Value{Key: key, Text: text, Origin: o}
}

func (m *merged) drop(prefix string) {
	m.order = slices.DeleteFunc(m.order, func(k string) bool {
		if k == prefix || strings.HasPrefix(k, prefix+".") {
			delete(m.values, k)
			return true
		}
		return false
	})
}

// invariant: a Machinefile is its extends, in order, then itself, each
// file followed by its overlay; a later file's value replaces an earlier
// one's (an app or a setting whole, by name), a file is read once however
// many extend it, and a cycle of extends is refused.
func (l Library) chain(name string) ([]Placed, error) {
	var out []Placed
	seen := map[string]bool{}
	var walk func(name, by string, stack []string) error
	walk = func(name, by string, stack []string) error {
		if i := slices.Index(stack, name); i >= 0 {
			return &CycleError{Path: append(slices.Clone(stack[i:]), name), Of: "extends"}
		}
		p, ok := l.Machinefiles[name]
		if !ok {
			return &NotFoundError{Name: name, By: by}
		}
		stack = append(stack, name)
		for _, e := range p.File.Extends {
			if err := walk(e, name, stack); err != nil {
				return err
			}
		}
		if seen[name] {
			return nil
		}
		seen[name] = true
		out = append(out, p)
		if o, ok := l.Overlays[name]; ok {
			out = append(out, o)
		}
		return nil
	}
	return out, walk(name, "", nil)
}

func originOf(p Placed) Origin {
	name := p.File.Name
	if name == "" {
		name = p.File.Overrides
	}
	return Origin{Name: name, Layer: p.Layer, File: p.Path}
}

func merge(files []Placed) merged {
	m := merged{values: map[string]Value{}, set: map[string]listed[catalog.SettingEntry]{}}
	for _, p := range files {
		o := originOf(p)
		f := p.File
		if mc := f.Machine; mc != nil {
			m.machine = true
			m.put("machine.kind", string(mc.Kind), o)
			if s := mc.System; s != nil {
				for _, kv := range [][2]string{{"os", string(s.OS)}, {"version", s.Version}, {"arch", string(s.Arch)}} {
					if kv[1] != "" {
						m.put("machine.system."+kv[0], kv[1], o)
					}
				}
			}
		}
		if v := f.VM; v != nil {
			at := o
			m.vm = &at
			mergeVM(&m, *v, o)
		}
		if len(f.Prefer) > 0 {
			m.prefer = f.Prefer
			m.put("prefer", joined(f.Prefer), o)
		}
		if a := f.Apps; a != nil {
			mergeApps(&m, *a, o)
		}
		if s := f.Settings; s != nil {
			mergeSettings(&m, *s, o)
		}
		if a := f.Accounts; a != nil {
			for _, s := range a.Share {
				m.put("accounts.share."+s.Account, cmp.Or(s.For, "shared"), o)
			}
			for _, k := range a.Keep {
				m.put("accounts.keep."+k, "kept", o)
			}
		}
		if p := f.Ports; p != nil {
			m.drop("ports")
			m.put("ports.open", spans(p.Open), o)
			m.put("ports.to", string(p.To), o)
			if len(p.Except) > 0 {
				m.put("ports.except", ints(p.Except), o)
			}
		}
	}
	return m
}

func mergeVM(m *merged, v catalog.VMSpec, o Origin) {
	put := func(key, text string) {
		if text != "" && text != "0" {
			m.put("vm."+key, text, o)
		}
	}
	put("lifetime", string(v.Lifetime))
	if r := v.Resources; r != nil {
		put("resources.cpus", strconv.Itoa(r.CPUs))
		put("resources.memory", string(r.Memory))
		put("resources.disk", string(r.Disk))
	}
	put("network", string(v.Network))
	if i := v.Image; i != nil {
		put("image.from", i.From)
		put("image.bake", strings.Join(i.Bake, ", "))
	}
}

func mergeApps(m *merged, a catalog.AppList, o Origin) {
	for _, r := range a.Remove {
		m.apps = slices.DeleteFunc(m.apps, func(l listed[catalog.AppEntry]) bool { return l.name == r })
		m.drop("apps." + r)
	}
	for _, e := range slices.Concat(a.List, a.Add) {
		if i := slices.IndexFunc(m.apps, func(l listed[catalog.AppEntry]) bool { return l.name == e.Name }); i >= 0 {
			m.apps = slices.Delete(m.apps, i, i+1)
		}
		m.drop("apps." + e.Name)
		m.apps = append(m.apps, listed[catalog.AppEntry]{name: e.Name, entry: e, origin: o})
	}
}

func mergeSettings(m *merged, s catalog.SettingList, o Origin) {
	for _, r := range s.Remove {
		delete(m.set, r)
		m.setting = slices.DeleteFunc(m.setting, func(n string) bool { return n == r })
		m.drop("settings." + r)
	}
	for _, name := range sortedNames(s.Set) {
		if _, ok := m.set[name]; !ok {
			m.setting = append(m.setting, name)
		}
		m.drop("settings." + name)
		m.set[name] = listed[catalog.SettingEntry]{name: name, entry: s.Set[name], origin: o}
	}
}

func Resolve(lib Library, name string, this Machine) (Resolved, error) {
	// invariant: a Machinefile resolves against the catalog and the system it
	// says (this machine's where it says none): a mixin is never applied, a
	// computer has no vm, each item's method is the first of its matching
	// methods in prefer's order, then the system's priority, then the file's
	// order; the steps run requires first, managers before what they install,
	// `after` between two the plan holds; a cycle is refused.
	files, err := lib.chain(name)
	if err != nil {
		return Resolved{}, err
	}
	top := files[len(files)-1]
	if top.Layer == Overlay {
		top = files[len(files)-2]
	}
	if top.File.Machine == nil {
		return Resolved{}, &MixinError{Name: name}
	}
	m := merge(files)
	if said := m.values["machine.system.os"].Text; said == "" || said == string(this.System.OS) {
		mine := Origin{Name: name, Layer: ThisMachine}
		for _, kv := range [][2]string{{"os", string(this.System.OS)}, {"version", this.System.Version}, {"arch", string(this.System.Arch)}} {
			if _, ok := m.values["machine.system."+kv[0]]; !ok && kv[1] != "" {
				m.put("machine.system."+kv[0], kv[1], mine)
			}
		}
	}
	r := Resolved{Name: name, Kind: catalog.MachineRole(m.values["machine.kind"].Text)}
	r.System = catalog.System{
		OS: catalog.OS(m.values["machine.system.os"].Text), Version: m.values["machine.system.version"].Text,
		Arch: catalog.Arch(m.values["machine.system.arch"].Text), Machine: r.Kind,
	}
	for _, f := range files {
		r.Files = append(r.Files, originOf(f))
	}
	if r.Kind == catalog.RoleComputer && m.vm != nil {
		return Resolved{}, &VMOnComputerError{Name: name, File: m.vm.File}
	}
	p := planner{cat: lib.Catalog, sys: r.System, prefer: m.prefer, at: map[string]int{}}
	for _, a := range m.apps {
		if listed := p.app(a.entry, a.origin); listed != nil {
			return Resolved{}, listed
		}
	}
	for _, s := range m.setting {
		l := m.set[s]
		if l.entry.From == catalog.FromComputer && r.Kind == catalog.RoleComputer {
			return Resolved{}, &BadValueError{Item: s, Key: "from", Problem: "a computer takes no values from a computer: from is a VM's"}
		}
		if set := p.setting(s, l.entry, l.origin); set != nil {
			return Resolved{}, set
		}
	}
	if brought := p.implied(); brought != nil {
		return Resolved{}, brought
	}
	steps, err := p.ordered()
	if err != nil {
		return Resolved{}, err
	}
	r.Steps, r.Skipped = steps, p.skipped
	r.Values = values(m, r, lib.Catalog)
	return r, nil
}

type planner struct {
	cat     catalog.Catalog
	sys     catalog.System
	prefer  []catalog.ManagerKey
	steps   []Step
	at      map[string]int
	skipped []Skip
}

func (p *planner) item(name string, by Origin, want catalog.Kind) (catalog.Placed, error) {
	it, ok := p.cat.Item(name)
	if !ok {
		for _, c := range p.cat.Items {
			if slices.Contains(c.Item.RenamedFrom, name) {
				it, ok = c, true
			}
		}
	}
	switch {
	case !ok:
		return catalog.Placed{}, &UnknownItemError{Name: name, By: by.Name}
	case want == catalog.KindSetting && it.Item.Kind != catalog.KindSetting:
		return catalog.Placed{}, &UnknownItemError{Name: name, By: by.Name, Is: "an app: list it under apps"}
	case want != catalog.KindSetting && it.Item.Kind == catalog.KindSetting:
		return catalog.Placed{}, &UnknownItemError{Name: name, By: by.Name, Is: "a setting: list it under settings"}
	}
	return it, nil
}

func itemOrigin(it catalog.Placed) Origin {
	return Origin{Name: it.Item.Name, Layer: Layer(it.Layer), File: it.File}
}

func (p *planner) app(e catalog.AppEntry, o Origin) error {
	it, err := p.item(e.Name, o, catalog.KindApp)
	if err != nil {
		return err
	}
	if bad := checkValues(it.Item, e.Options); bad != nil {
		return bad
	}
	step, err := p.choose(it, e.Manager, e.Version, "listed")
	if err != nil {
		var none *NoMethodError
		if e.Optional && errors.As(err, &none) {
			p.skipped = append(p.skipped, Skip{Item: it.Item.Name, Why: "optional, no method for " + p.system(), Origin: o})
			return nil
		}
		return err
	}
	step.Origin, step.Values = o, e.Options
	p.add(step)
	return nil
}

func (p *planner) setting(name string, e catalog.SettingEntry, o Origin) error {
	it, err := p.item(name, o, catalog.KindSetting)
	if err != nil {
		return err
	}
	if bad := checkValues(it.Item, e.Values); bad != nil {
		return bad
	}
	step, err := p.choose(it, "", "", "listed")
	if err != nil {
		var none *NoMethodError
		if e.Optional && errors.As(err, &none) {
			p.skipped = append(p.skipped, Skip{Item: name, Why: "optional, no method for " + p.system(), Origin: o})
			return nil
		}
		return err
	}
	step.Origin, step.Values, step.From = o, e.Values, e.From
	p.add(step)
	return nil
}

func (p *planner) add(s Step) {
	p.at[s.Item] = len(p.steps)
	p.steps = append(p.steps, s)
}

func (p *planner) system() string {
	s := string(p.sys.OS)
	if p.sys.Version != "" {
		s += " " + p.sys.Version
	}
	return s
}

// value: a method's place in the order a system tries them: prefer's
// listed keys first, then the managers this system ranks, by rank, then
// the other managers, then a script; a tie keeps the file's order.
func (p *planner) rank(m catalog.MethodSpec) (preferred, class, rank int) {
	key := m.Key()
	preferred = len(p.prefer)
	if i := slices.Index(p.prefer, catalog.ManagerKey(key)); i >= 0 {
		preferred = i
	}
	if owner, ok := p.manager(key); ok && owner.Item.Priority != nil {
		if r := priorityOn(*owner.Item.Priority, p.sys.OS); r > 0 {
			return preferred, 0, r
		}
	}
	if slices.Contains(catalog.ManagerKey("").Enum(), key) {
		return preferred, 1, 0
	}
	return preferred, 2, 0
}

func priorityOn(p catalog.Priority, os catalog.OS) int {
	switch os {
	case catalog.OSMacOS:
		return p.MacOS
	case catalog.OSUbuntu:
		return p.Ubuntu
	case catalog.OSDebian:
		return p.Debian
	case catalog.OSAny:
	}
	return 0
}

// value: the manager item that installs by a method key (homebrew for
// brew and cask).
func (p *planner) manager(key string) (catalog.Placed, bool) {
	for _, it := range p.cat.Items {
		if slices.Contains(it.Item.Manages, catalog.ManagerKey(key)) {
			return it, true
		}
	}
	return catalog.Placed{}, false
}

func (p *planner) choose(it catalog.Placed, manager catalog.ManagerKey, version, why string) (Step, error) {
	var matching []catalog.MethodSpec
	for _, m := range it.Item.Methods {
		if m.Matches(p.sys) && (manager == "" || m.Key() == string(manager)) {
			matching = append(matching, m)
		}
	}
	if len(matching) == 0 {
		return Step{}, &NoMethodError{Item: it.Item.Name, System: p.system(), Manager: string(manager), Why: why}
	}
	slices.SortStableFunc(matching, func(a, b catalog.MethodSpec) int {
		ap, ac, ar := p.rank(a)
		bp, bc, br := p.rank(b)
		return cmp.Or(cmp.Compare(ap, bp), cmp.Compare(ac, bc), cmp.Compare(ar, br))
	})
	chosen := matching[0]
	if version != "" && !chosen.Pins() {
		alt := ""
		for _, m := range matching[1:] {
			if m.Pins() {
				alt = m.Key()
				break
			}
		}
		return Step{}, &UnpinnableError{Item: it.Item.Name, Version: version, Method: chosen.Key(), Alternative: alt}
	}
	return Step{
		Item: it.Item.Name, Kind: it.Item.Kind, Method: chosen.Key(), Chosen: p.because(chosen, len(matching)), Spec: chosen,
		Version: version, Phase: cmp.Or(it.Item.Phase, catalog.PhaseApply), Needs: slices.Concat(it.Item.Needs, chosen.Needs),
		Why: why, Defined: itemOrigin(it),
	}, nil
}

func (p *planner) because(m catalog.MethodSpec, of int) string {
	if of == 1 {
		return "its one method for " + p.system()
	}
	preferred, class, rank := p.rank(m)
	switch {
	case preferred < len(p.prefer):
		return "preferred"
	case class == 0:
		return "first on " + string(p.sys.OS) + " (rank " + strconv.Itoa(rank) + ")"
	}
	return "first of its methods for " + p.system()
}

// invariant: an item of the plan brings in what it requires and the
// manager its method installs by, each resolved like a listed one; a
// setting's requires-one-of is met by any one the plan holds, else by its
// first.
func (p *planner) implied() error {
	// invariant: a step brought in is itself judged: the walk goes on while
	// the plan grows.
	for next := 0; next < len(p.steps); {
		s := p.steps[next]
		next++
		it, _ := p.cat.Item(s.Item)
		needs := slices.Clone(it.Item.Requires)
		if len(it.Item.RequiresOneOf) > 0 && !slices.ContainsFunc(it.Item.RequiresOneOf, p.holds) {
			needs = append(needs, it.Item.RequiresOneOf[0])
		}
		if owner, ok := p.manager(s.Method); ok && owner.Item.Name != s.Item {
			needs = append(needs, owner.Item.Name)
		}
		for _, n := range needs {
			if p.holds(n) {
				continue
			}
			why := "required by " + s.Item
			if owner, ok := p.manager(s.Method); ok && owner.Item.Name == n {
				why = "the manager of " + s.Item
			}
			dep, err := p.item(n, Origin{Name: s.Item}, catalog.KindApp)
			if err != nil {
				return err
			}
			step, err := p.choose(dep, "", "", why)
			if err != nil {
				return err
			}
			step.Origin = s.Defined
			p.add(step)
		}
	}
	return nil
}

func (p *planner) holds(name string) bool { _, ok := p.at[name]; return ok }

// invariant: a stable order: among the steps whose dependencies ran, the
// first in the plan's order runs next; what never becomes ready is in a
// cycle, which is printed.
func (p *planner) ordered() ([]Step, error) {
	deps := make([][]int, len(p.steps))
	for i, s := range p.steps {
		it, _ := p.cat.Item(s.Item)
		before := slices.Concat(it.Item.Requires, it.Item.RequiresOneOf, it.Item.After)
		if owner, ok := p.manager(s.Method); ok {
			before = append(before, owner.Item.Name)
		}
		for _, b := range before {
			if j, ok := p.at[b]; ok && j != i && !slices.Contains(deps[i], j) {
				deps[i] = append(deps[i], j)
			}
		}
	}
	done := make([]bool, len(p.steps))
	var out []Step
	for len(out) < len(p.steps) {
		next := -1
		for i := range p.steps {
			if !done[i] && !slices.ContainsFunc(deps[i], func(j int) bool { return !done[j] }) {
				next = i
				break
			}
		}
		if next < 0 {
			return nil, &CycleError{Path: p.cycle(deps, done), Of: "the items' requires and after"}
		}
		done[next] = true
		out = append(out, p.steps[next])
	}
	return out, nil
}

func (p *planner) cycle(deps [][]int, done []bool) []string {
	// invariant: a step left undone waits on another left undone, so the
	// walk along them comes back to a step it passed: the cycle.
	at := slices.Index(done, false)
	var path []int
	for !slices.Contains(path, at) {
		path = append(path, at)
		for _, j := range deps[at] {
			if !done[j] {
				at = j
				break
			}
		}
	}
	loop := path[slices.Index(path, at):]
	names := make([]string, 0, len(loop)+1)
	for _, i := range loop {
		names = append(names, p.steps[i].Item)
	}
	return append(names, names[0])
}

// invariant: every value names an option of its item and fits its type; a
// secret's is a `vault:` reference, never the secret written out.
func checkValues(it catalog.Item, values map[string]catalog.OptionValue) error {
	for _, k := range sortedNames(values) {
		v := values[k]
		i := slices.IndexFunc(it.Options, func(o catalog.Option) bool { return o.Name == k })
		if i < 0 {
			return &BadValueError{Item: it.Name, Key: k, Problem: "no option of " + it.Name}
		}
		o := it.Options[i]
		vault := strings.HasPrefix(string(v), catalog.VaultPrefix)
		switch {
		case o.Secret && !vault:
			return &BadValueError{Item: it.Name, Key: k, Problem: "a secret is a vault: reference (`vault:<name>`), never written out"}
		case vault && o.Type != catalog.OptionString:
			return &BadValueError{Item: it.Name, Key: k, Problem: "a vault: reference is a string option's value"}
		case !vault:
			if problem := o.Check(v); problem != "" {
				return &BadValueError{Item: it.Name, Key: k, Problem: problem}
			}
		}
	}
	return nil
}

func values(m merged, r Resolved, c catalog.Catalog) []Value {
	// invariant: the values in the Machinefile's reading order: what the
	// machine is and how it runs, then what it carries (the steps in the
	// order they run, then what is skipped), then what it is lent and may
	// open.
	out := make([]Value, 0, len(m.order)+3*len(r.Steps))
	family := func(k string) string { f, _, _ := strings.Cut(k, "."); return f }
	for _, f := range []string{"machine", "vm", "prefer"} {
		for _, k := range m.order {
			if family(k) == f {
				out = append(out, m.values[k])
			}
		}
	}
	for _, s := range r.Steps {
		group := "apps."
		if s.Kind == catalog.KindSetting {
			group = "settings."
		}
		key := group + s.Item
		text := s.Why
		if s.Version != "" {
			text = s.Version
		}
		out = append(out, Value{Key: key, Text: text, Origin: s.Origin}, Value{Key: key + ".method", Text: s.Method, Origin: s.Defined}, Value{Key: key + ".because", Text: s.Chosen, Origin: s.Defined})
		for _, k := range sortedNames(s.Values) {
			out = append(out, Value{Key: key + "." + k, Text: string(s.Values[k]), Origin: s.Origin})
		}
		if s.From != "" {
			out = append(out, Value{Key: key + ".from", Text: s.From, Origin: s.Origin})
		}
	}
	for _, s := range r.Skipped {
		group := "apps."
		if it, ok := c.Item(s.Item); ok && it.Item.Kind == catalog.KindSetting {
			group = "settings."
		}
		out = append(out, Value{Key: group + s.Item, Text: s.Why, Origin: s.Origin})
	}
	for _, f := range []string{"accounts", "ports"} {
		for _, k := range m.order {
			if family(k) == f {
				out = append(out, m.values[k])
			}
		}
	}
	return out
}

func Explain(r Resolved, key string) ([]Value, error) {
	// invariant: the values of one key: the key itself and every key under
	// it; a key the resolution holds no value of is refused.
	if key == "" {
		return r.Values, nil
	}
	var out []Value
	for _, v := range r.Values {
		if v.Key == key || strings.HasPrefix(v.Key, key+".") {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil, &UnknownKeyError{Key: key, Name: r.Name}
	}
	return out, nil
}

func joined[T ~string](list []T) string {
	words := make([]string, len(list))
	for i, w := range list {
		words[i] = string(w)
	}
	return strings.Join(words, ", ")
}

func spans(set catalog.PortSet) string {
	words := make([]string, len(set))
	for i, s := range set {
		words[i] = s.String()
	}
	return strings.Join(words, ", ")
}

func ints(list []int) string {
	words := make([]string, len(list))
	for i, n := range list {
		words[i] = strconv.Itoa(n)
	}
	return strings.Join(words, ", ")
}

func sortedNames[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
