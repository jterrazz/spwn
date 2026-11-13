package machinefile

import (
	"cmp"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
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
	Spec catalog.MethodSpec
	// value: the range of versions the Machinefile lets the app move within,
	// and whether the app updates itself, which no range moves.
	Version     catalog.Range
	UpdatesSelf bool
	// value: the version the lock holds the item at, when the step is
	// carried by the method the lock names and that method pins it: the
	// one the install takes; the lock's file when the lock names the
	// method; and Recorded, the version the lock holds where the method
	// cannot pin it, which the install does not take.
	Locked, Recorded, Lock string
	Phase                  catalog.Phase
	// value: the step is baked into the VM image (vm.image.bake, and what
	// a baked item requires): `apply --phase image` runs it, and a bare
	// apply carries it as any other.
	Baked bool
	Needs []catalog.Need
	// value: what has to happen once it is applied for it to take (the
	// Dock restarted, a logout), and the steps of the plan it waits on.
	Restarts []catalog.AfterApply
	Waits    []string
	Why      string
	Values   map[string]catalog.OptionValue
	From     string
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
	Name string
	Lock Lock
	// value: the shared source read, as the lock records it; nil when the
	// machine reads none.
	Source  *catalog.LockSource
	Kind    catalog.MachineRole
	System  catalog.System
	Files   []Origin
	Values  []Value
	Steps   []Step
	Skipped []Skip
	// value: the owner's lines of each phase, in the order image, boot,
	// upgrade, a phase its files say none of left out.
	Scripts []Script
}

// value: the owner's shell lines of one phase (`run.boot`), and the file
// that says them.

type Script struct {
	Phase  RunPhase
	Lines  string
	Origin Origin
}

// value: when a Script runs: image, as its image is built; boot, at a
// VM's first start; upgrade, at each upgrade, after its apps.

type RunPhase string

const (
	RunImage   RunPhase = "image"
	RunBoot    RunPhase = "boot"
	RunUpgrade RunPhase = "upgrade"
)

func (r Resolved) ScriptOf(phase RunPhase) (Script, bool) {
	for _, s := range r.Scripts {
		if s.Phase == phase {
			return s, true
		}
	}
	return Script{}, false
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
	// value: the items baked into the image, and the file that says so.
	bake []listed[struct{}]
	// value: the folders a VM made from it shares with its computer, as
	// the strongest file that says them lists them.
	mounts  []catalog.Mount
	scripts map[RunPhase]Script
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
// one's (an app or a setting whole, by name; accounts and ports whole), a
// file is read once however many extend it, and a cycle of extends is
// refused.
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
	m := merged{values: map[string]Value{}, set: map[string]listed[catalog.SettingEntry]{}, scripts: map[RunPhase]Script{}}
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
			m.drop("accounts")
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
		if r := f.Run; r != nil {
			mergeRun(&m, *r, o)
		}
	}
	return m
}

// invariant: a file that says a phase's lines replaces those of the files
// it extends, whole; a phase it says nothing of keeps theirs.
func mergeRun(m *merged, r catalog.RunSpec, o Origin) {
	for _, s := range []Script{{Phase: RunImage, Lines: r.Image}, {Phase: RunBoot, Lines: r.Boot}, {Phase: RunUpgrade, Lines: r.Upgrade}} {
		if strings.TrimSpace(s.Lines) == "" {
			continue
		}
		s.Origin = o
		m.scripts[s.Phase] = s
		m.put("run."+string(s.Phase), strings.Join(ScriptLines(s.Lines), "; "), o)
	}
}

func ScriptLines(text string) []string {
	// value: a script's lines as the plan shows them, each line the owner
	// wrote that holds a word.
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, strings.TrimRight(l, " \t"))
		}
	}
	return out
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
		if i.Bake != nil {
			m.bake = nil
			for _, b := range i.Bake {
				m.bake = append(m.bake, listed[struct{}]{name: b, origin: o})
			}
		}
	}
	if v.Mounts != nil {
		m.mounts = v.Mounts
		said := make([]string, 0, len(v.Mounts))
		for _, mt := range v.Mounts {
			w := mt.Computer + " at " + mt.Guest
			if mt.ReadOnly {
				w += " read-only"
			}
			said = append(said, w)
		}
		m.put("vm.mounts", strings.Join(said, ", "), o)
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

func Resolve(lib Library, name string, this Machine, lock Lock) (Resolved, error) {
	// invariant: a Machinefile resolves against the catalog and the system it
	// says (this machine's where it says none): a mixin is never applied, a
	// computer has no vm, each item's method is the first of its matching
	// methods in prefer's order, then the system's priority, then the file's
	// order; the steps run requires first, managers before what they install,
	// `after` between two the plan holds; a cycle is refused. An app the
	// lock holds is carried by the lock's method wherever the system and
	// the entry still allow it, at the lock's version where it meets the
	// one asked.
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
	r := Resolved{Name: name, Lock: lock, Kind: catalog.MachineRole(m.values["machine.kind"].Text)}
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
	p := planner{cat: lib.Catalog, sys: r.System, prefer: m.prefer, lock: lock, at: map[string]int{}}
	for _, b := range m.bake {
		if _, ok := p.lookup(b.name); !ok {
			return Resolved{}, &UnknownItemError{Name: b.name, By: b.origin.Name, At: b.origin.File}
		}
	}
	for _, a := range m.apps {
		if listed := p.app(a.entry, a.origin, "listed"); listed != nil {
			return Resolved{}, listed
		}
	}
	for _, b := range m.bake {
		if _, ok := p.at[b.name]; ok || slices.ContainsFunc(p.skipped, func(s Skip) bool { return s.Item == b.name }) {
			continue
		}
		if baked := p.app(catalog.AppEntry{Name: b.name}, b.origin, "baked"); baked != nil {
			return Resolved{}, baked
		}
	}
	for _, s := range m.setting {
		l := m.set[s]
		if l.entry.From == catalog.FromComputer && r.Kind == catalog.RoleComputer {
			return Resolved{}, &BadValueError{Item: s, Key: "from", Problem: "a computer takes no values from a computer: from is a VM's", At: at(l.origin, l.entry.Line)}
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
	r.Steps, r.Skipped = p.baked(steps, m.bake), p.skipped
	for _, phase := range []RunPhase{RunImage, RunBoot, RunUpgrade} {
		if s, ok := m.scripts[phase]; ok {
			r.Scripts = append(r.Scripts, s)
		}
	}
	r.Values = values(m, r, lib.Catalog)
	return r, nil
}

type planner struct {
	cat     catalog.Catalog
	sys     catalog.System
	prefer  []catalog.ManagerKey
	lock    Lock
	steps   []Step
	at      map[string]int
	skipped []Skip
}

// value: an item by its name, or by a name it was renamed from.
func (p *planner) lookup(name string) (catalog.Placed, bool) {
	it, ok := p.cat.Item(name)
	if !ok {
		for _, c := range p.cat.Items {
			if slices.Contains(c.Item.RenamedFrom, name) {
				it, ok = c, true
			}
		}
	}
	return it, ok
}

// value: where an entry of a file is written, `<file>:<line>`, for a
// refusal to point at; empty for an item another one brings in.
func at(o Origin, line int) string {
	switch {
	case o.File == "":
		return ""
	case line > 0:
		return o.File + ":" + strconv.Itoa(line)
	}
	return o.File
}

func (p *planner) item(name string, by Origin, where string, want catalog.Kind) (catalog.Placed, error) {
	it, ok := p.lookup(name)
	switch {
	case !ok:
		return catalog.Placed{}, &UnknownItemError{Name: name, By: by.Name, At: where}
	case want == catalog.KindSetting && it.Item.Kind != catalog.KindSetting:
		return catalog.Placed{}, &UnknownItemError{Name: name, By: by.Name, Is: "an app: list it under apps", At: where}
	case want != catalog.KindSetting && it.Item.Kind == catalog.KindSetting:
		return catalog.Placed{}, &UnknownItemError{Name: name, By: by.Name, Is: "a setting: list it under settings", At: where}
	}
	return it, nil
}

func itemOrigin(it catalog.Placed) Origin {
	return Origin{Name: it.Item.Name, Layer: Layer(it.Layer), File: it.File}
}

// invariant: what the image bakes is its bake list and, step by step,
// what a baked step requires (its requires, its provider, its manager),
// so `apply --phase image` never runs an item whose requirement it left.
func (p *planner) baked(steps []Step, bake []listed[struct{}]) []Step {
	at := make(map[string]int, len(steps))
	for i, s := range steps {
		at[s.Item] = i
	}
	queue := make([]string, 0, len(bake))
	for _, b := range bake {
		queue = append(queue, b.name)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		i, ok := at[name]
		if !ok || steps[i].Baked {
			continue
		}
		steps[i].Baked = true
		it, _ := p.cat.Item(name)
		for _, l := range p.cat.Links(it.Item, steps[i].Method, p.sys) {
			queue = append(queue, l.Items...)
		}
	}
	return steps
}

func (p *planner) app(e catalog.AppEntry, o Origin, why string) error {
	where := at(o, e.Line)
	it, err := p.item(e.Name, o, where, catalog.KindApp)
	if err != nil {
		return err
	}
	if bad := checkValues(it.Item, e.Options, where); bad != nil {
		return bad
	}
	step, err := p.choose(it, e.Manager, e.Version, why, where)
	if err != nil {
		var none *NoMethodError
		if e.Optional && errors.As(err, &none) {
			p.skipped = append(p.skipped, Skip{Item: it.Item.Name, Why: p.skippedFor(it, e.Manager), Origin: o})
			return nil
		}
		return err
	}
	step.Origin, step.Values = o, e.Options
	p.add(step)
	return nil
}

func (p *planner) setting(name string, e catalog.SettingEntry, o Origin) error {
	where := at(o, e.Line)
	it, err := p.item(name, o, where, catalog.KindSetting)
	if err != nil {
		return err
	}
	if bad := checkValues(it.Item, e.Values, where); bad != nil {
		return bad
	}
	step, err := p.choose(it, "", catalog.Range{}, "listed", where)
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

// value: why an optional item is skipped: no method for the system, or,
// when its entry asks a manager, none by that manager, with the methods
// it has there, so a manager pinned by mistake reads apart from an item
// the system lacks.
func (p *planner) skippedFor(it catalog.Placed, manager catalog.ManagerKey) string {
	if manager == "" {
		return "optional, no method for " + p.system()
	}
	var has []string
	for _, m := range it.Item.Methods {
		if m.Matches(p.sys) && !slices.Contains(has, m.Key()) {
			has = append(has, m.Key())
		}
	}
	if len(has) == 0 {
		return "optional, no method for " + p.system()
	}
	return "optional, no method by " + string(manager) + " for " + p.system() + ", where it has " + strings.Join(has, ", ")
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

func (p *planner) choose(it catalog.Placed, manager catalog.ManagerKey, version catalog.Range, why, where string) (Step, error) {
	var matching []catalog.MethodSpec
	for _, m := range it.Item.Methods {
		if m.Matches(p.sys) && (manager == "" || m.Key() == string(manager)) {
			matching = append(matching, m)
		}
	}
	if len(matching) == 0 {
		return Step{}, &NoMethodError{Item: it.Item.Name, System: p.system(), Manager: string(manager), Why: why, At: where}
	}
	matching = p.cat.Ordered(matching, p.sys.OS, p.prefer)
	chosen, because := matching[0], p.because(matching[0], len(matching))
	locked, ok := p.lock.App(it.Item.Name)
	if ok && it.Item.Kind != catalog.KindSetting {
		at := slices.IndexFunc(matching, func(m catalog.MethodSpec) bool { return m.Method() == locked.Method && m.Fits(version) })
		if at >= 0 {
			chosen, because = matching[at], "locked"
		} else {
			ok = false
		}
	}
	if !chosen.Fits(version) {
		alt := ""
		for _, m := range matching[1:] {
			if m.Fits(version) {
				alt = m.Key()
				break
			}
		}
		return Step{}, &UnpinnableError{Item: it.Item.Name, Version: version.String(), Method: chosen.Key(), Holds: chosen.HoldWords(), Alternative: alt, At: where}
	}
	held, recorded := lockedAt(ok, version, locked.Version), ""
	if !chosen.Pins() {
		held, recorded = "", held
	}
	return Step{
		Item: it.Item.Name, Kind: it.Item.Kind, Method: chosen.Method(), Chosen: because, Spec: chosen,
		Version: version, UpdatesSelf: it.Item.Updates == catalog.UpdatesSelf, Locked: held, Recorded: recorded, Lock: lockFile(ok, p.lock.File), Phase: cmp.Or(it.Item.Phase, catalog.PhaseApply), Needs: slices.Concat(it.Item.Needs, chosen.Needs),
		Restarts: it.Item.AfterApply, Why: why, Defined: itemOrigin(it),
	}, nil
}

func lockedAt(ok bool, asked catalog.Range, locked string) string {
	if ok && asked.Contains(locked) {
		return locked
	}
	return ""
}

func lockFile(ok bool, file string) string {
	if ok {
		return file
	}
	return ""
}

func (p *planner) because(m catalog.MethodSpec, of int) string {
	if of == 1 {
		return "its one method for " + p.system()
	}
	st := p.cat.Standing(m, p.sys.OS, p.prefer)
	switch {
	case st.Preferred < len(p.prefer):
		return "preferred"
	case st.Class == catalog.ClassRanked:
		return "first on " + string(p.sys.OS) + " (rank " + strconv.Itoa(st.Rank) + ")"
	}
	return "first of its methods for " + p.system()
}

// invariant: an item of the plan brings in every item it leans on, each
// resolved like a listed one: what it requires, its method's provider and
// its method's manager; a setting's requires-one-of is met by any one the
// plan holds, else by its first.
func (p *planner) implied() error {
	// invariant: a step brought in is itself judged: the walk goes on while
	// the plan grows.
	for next := 0; next < len(p.steps); {
		s := p.steps[next]
		next++
		it, _ := p.cat.Item(s.Item)
		for _, l := range p.cat.Links(it.Item, s.Method, p.sys) {
			if slices.ContainsFunc(l.Items, p.holds) {
				continue
			}
			dep, err := p.item(l.Items[0], Origin{Name: s.Item}, "", catalog.KindApp)
			if err != nil {
				return err
			}
			step, err := p.choose(dep, "", catalog.Range{}, broughtBy(l.Kind, s.Item), "")
			if err != nil {
				return err
			}
			step.Origin = s.Defined
			p.add(step)
		}
	}
	return nil
}

func broughtBy(k catalog.LinkKind, item string) string {
	switch k {
	case catalog.LinkManager:
		return "the manager of " + item
	case catalog.LinkProvidedBy:
		return "the provider of " + item
	case catalog.LinkRequires, catalog.LinkRequiresOneOf:
	}
	return "required by " + item
}

func (p *planner) holds(name string) bool { _, ok := p.at[name]; return ok }

// invariant: a stable order: among the steps whose dependencies ran, the
// first in the plan's order runs next; what never becomes ready is in a
// cycle, which is printed.
func (p *planner) ordered() ([]Step, error) {
	deps := make([][]int, len(p.steps))
	for i, s := range p.steps {
		it, _ := p.cat.Item(s.Item)
		before := slices.Clone(it.Item.After)
		for _, l := range p.cat.Links(it.Item, s.Method, p.sys) {
			before = append(before, l.Items...)
		}
		for _, b := range before {
			if j, ok := p.at[b]; ok && j != i && !slices.Contains(deps[i], j) {
				deps[i] = append(deps[i], j)
			}
		}
	}
	for i := range p.steps {
		p.steps[i].Waits = nil
		for _, j := range deps[i] {
			p.steps[i].Waits = append(p.steps[i].Waits, p.steps[j].Item)
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
func checkValues(it catalog.Item, values map[string]catalog.OptionValue, where string) error {
	for _, k := range sortedNames(values) {
		v := values[k]
		i := slices.IndexFunc(it.Options, func(o catalog.Option) bool { return o.Name == k })
		if i < 0 {
			return &BadValueError{Item: it.Name, Key: k, Problem: "no option of " + it.Name, At: where}
		}
		o := it.Options[i]
		vault := strings.HasPrefix(string(v), catalog.VaultPrefix)
		switch {
		case o.Secret && !vault:
			return &BadValueError{Item: it.Name, Key: k, Problem: "a secret is a vault: reference (`vault:<name>`), never written out", At: where}
		case vault && o.Type != catalog.OptionString:
			return &BadValueError{Item: it.Name, Key: k, Problem: "a vault: reference is a string option's value", At: where}
		case !vault:
			if problem := o.Check(v); problem != "" {
				return &BadValueError{Item: it.Name, Key: k, Problem: problem, At: where}
			}
		}
	}
	return nil
}

func values(m merged, r Resolved, c catalog.Catalog) []Value {
	// invariant: the values in the Machinefile's reading order: what the
	// machine is and how it runs, then what it carries (the steps in the
	// order they run, then what is skipped), then what it is lent and may
	// open, then the owner's lines per phase.
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
		if !s.Version.IsZero() {
			text = s.Version.String()
		}
		chose := s.Defined
		if s.Lock != "" {
			chose = Origin{Name: r.Name, Layer: Locked, File: s.Lock}
		}
		out = append(out, Value{Key: key, Text: text, Origin: s.Origin}, Value{Key: key + ".method", Text: s.Method, Origin: chose}, Value{Key: key + ".because", Text: s.Chosen, Origin: chose})
		if s.Locked != "" {
			out = append(out, Value{Key: key + ".locked", Text: s.Locked, Origin: chose})
		}
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
	for _, f := range []string{"accounts", "ports", "run"} {
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
