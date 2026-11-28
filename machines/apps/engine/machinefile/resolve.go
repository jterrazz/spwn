package machinefile

import (
	"cmp"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

// value: one value of a resolved Machinefile, by its key (`vm.network`,
// `apps.jq.method`), as text, with where it was read.

type Value struct {
	Key, Text string
	Origin    Origin
	// value: the text held a secret: an option's value is withheld whole,
	// any other text word by word.
	Withheld bool
	// value: an option's value beside the package's own default for it ("" when
	// the package gives none, and never a secret's), and the file of the
	// package that defines it; empty and nil on every other value.
	Default string
	Defined *Origin
}

// value: one package of the plan: what it is, the method that installs or
// sets it (its manager key) and why it was chosen, the version asked, its
// phase and what it asks of the owner, why it is in the plan, and the
// values it is given.

type Step struct {
	Package string
	Kind    catalog.Kind
	Method  string
	Chosen  string
	// value: the method chosen, as its file says it.
	Spec catalog.MethodSpec
	// value: the range of versions the Machinefile lets the app move within,
	// and whether the app updates itself, which no range moves.
	Version     catalog.Range
	UpdatesSelf bool
	// value: the version the lock holds the package at, when the step is
	// carried by the method the lock names and that method pins it: the
	// one the install takes; the lock's file when the lock names the
	// method; and Recorded, the version the lock holds where the method
	// cannot pin it, which the install does not take.
	Locked, Recorded, Lock string
	Phase                  catalog.Phase
	// value: the step is baked into the VM image (vm.image.bake, each
	// step of the image phase, and what a baked package requires): `apply
	// --phase image` runs it, and a bare apply carries it as any other but
	// a step of the image phase, which it leaves.
	Baked bool
	Needs []catalog.Need
	// value: what has to happen once it is applied for it to take (the
	// Dock restarted, a logout), and the steps of the plan it waits on.
	Restarts []catalog.AfterApply
	Waits    []string
	Why      string
	Values   map[string]catalog.OptionValue
	From     string
	// value: the Machinefile marks the package optional: it is planned
	// where a method exists, and skipped where none does.
	Optional bool
	// value: where the step was asked (a Machinefile, or the package that
	// brings it in), and where its package is defined.
	Origin, Defined Origin
	// value: the Machinefile that lists the step, or lists the package
	// that brings it in, followed to the file; the chapter of a drift it
	// counts in; and, for an app, how far its version lets it move in the
	// owner's words ("stays on 24, takes minors and fixes"), empty for a
	// setting.
	File    Origin
	Chapter Chapter
	Rule    string
}

// value: a package the Machinefile marks optional that has no method for the system.

type Skip struct {
	Package, Why string
	Origin       Origin
}

// value: a Machinefile resolved: its name, the files read weakest first,
// every value with its origin, the steps in the order they run, and the
// optional packages skipped.

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
	// upgrade, each file's of a phase its own Script in the chain's order,
	// a phase its files say none of left out.
	Scripts []Script
}

// value: the owner's shell lines of one phase (`run.boot`) one file says,
// and that file.

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

func ScriptsOf(scripts []Script, phase RunPhase) []Script {
	var out []Script
	for _, s := range scripts {
		if s.Phase == phase {
			out = append(out, s)
		}
	}
	return out
}

// value: this machine, for what a Machinefile does not say of its system.

type Machine struct {
	// value: the machine's name, whose own changes join the chain of the
	// Machinefile it follows.
	Name   string
	Kind   catalog.MachineRole
	System catalog.SystemSpec
}

type merged struct {
	values  map[string]Value
	order   []string
	machine bool
	vm      *Origin
	// value: the packages baked into the image, and the file that says so.
	bake []listed[struct{}]
	// value: the folders a VM made from it shares with its computer, as
	// the strongest file that says them lists them.
	mounts []catalog.Mount
	// value: each file's lines of each phase, in the chain's order.
	scripts []Script
	prefer  []catalog.ManagerKey
	// value: the ports of the strongest file that says them, whole, an
	// overlay's changes made key by key on them.
	ports *catalog.PortRights
	// value: what the files kept say of each program's account, the key
	// each lists it under, in the chain's order.
	accounts map[catalog.ProgramName][]account
	// value: ports merged into what cannot be read, by the overlay that
	// made them so.
	refused error
	apps    []listed[catalog.AppEntry]
	set     map[string]listed[catalog.SettingEntry]
	setting []string
	// value: each Machinefile's name and the names of every file it
	// extends, near or far.
	ancestors map[string][]string
	// value: what each file of the chain says of each app and setting
	// that no file extending it says otherwise.
	appHeads map[string][]claim[catalog.AppEntry]
	setHeads map[string][]claim[catalog.SettingEntry]
	// value: the last Machinefile of the chain, the one resolved.
	top string
}

// value: what one file says of an app or a setting: its entry, or that it
// removes it; unit is the Machinefile it belongs to, an overlay's the
// file it changes.

type claim[T any] struct {
	unit    string
	origin  Origin
	entry   T
	removed bool
}

type account struct {
	unit, key string
	origin    Origin
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

// value: how far a chain may reach: 8 Machinefiles deep, itself counted,
// and 32 in all, its overlays not counted.
const (
	ChainDepth = 8
	ChainFiles = 32
)

// invariant: a Machinefile is its extends, in order and depth first, then
// itself, each file followed by its overlay, then, when the machine it is
// resolved for names none of them, that machine's own changes: its
// overlay that overrides the Machinefile asked, which no other machine
// reads. How the files merge is merge's.
func (l Library) chain(name, machine string) ([]Placed, error) {
	var out []Placed
	files, err := walkChain(name, func(name string) ([]string, bool, error) {
		p, ok := l.Machinefiles[name]
		return p.File.Extends, ok, nil
	})
	for _, n := range files {
		out = append(out, l.Machinefiles[n])
		if o, ok := l.Overlays[n]; ok && o.File.Overrides == n {
			out = append(out, o)
		}
	}
	if err != nil {
		return out, err
	}
	if o, ok := l.Overlays[machine]; ok && !slices.Contains(files, machine) && o.File.Overrides == name {
		out = append(out, o)
	}
	return out, nil
}

// invariant: the files of a chain, each once, in the order they merge,
// read by extends (whether the file is there, and what it extends). A
// cycle of extends is refused first, printed around its loop whatever its
// length; then a chain more than ChainDepth deep, printed along its
// longest path, and one of more than ChainFiles files, naming the file
// past the bound. A file is read once however many extend it, and its own
// depth kept, so a file reached again deeper is judged at that depth: the
// work is one visit per file and one per extends entry.
func walkChain(top string, extends func(name string) ([]string, bool, error)) ([]string, error) {
	type read struct {
		extends []string
		ok      bool
		err     error
	}
	cache := map[string]read{}
	look := func(name string) read {
		r, ok := cache[name]
		if !ok {
			r.extends, r.ok, r.err = extends(name)
			cache[name] = r
		}
		return r
	}
	done := map[string]bool{}
	var acyclic func(name string, stack []string) error
	acyclic = func(name string, stack []string) error {
		if i := slices.Index(stack, name); i >= 0 {
			return &CycleError{Path: append(slices.Clone(stack[i:]), name), Of: "extends"}
		}
		r := look(name)
		if r.err != nil {
			return r.err
		}
		if !r.ok || done[name] {
			return nil
		}
		stack = append(stack, name)
		for _, e := range r.extends {
			if err := acyclic(e, stack); err != nil {
				return err
			}
		}
		done[name] = true
		return nil
	}
	if err := acyclic(top, nil); err != nil {
		return nil, err
	}
	var out []string
	below := map[string][]string{}
	var walk func(name, by string, stack []string) error
	walk = func(name, by string, stack []string) error {
		if deepest, seen := below[name]; seen {
			if len(stack)+len(deepest) > ChainDepth {
				return &ChainTooLongError{Path: slices.Concat(stack, deepest), Deep: true}
			}
			return nil
		}
		r := look(name)
		switch {
		case r.err != nil:
			return r.err
		case !r.ok:
			return &NotFoundError{Name: name, By: by}
		}
		stack = append(stack, name)
		if len(stack) > ChainDepth {
			return &ChainTooLongError{Path: slices.Clone(stack), Deep: true}
		}
		deepest := []string{name}
		for _, e := range r.extends {
			if err := walk(e, name, stack); err != nil {
				return err
			}
			if d := below[e]; len(d) >= len(deepest) {
				deepest = append([]string{name}, d...)
			}
		}
		if len(out) == ChainFiles {
			return &ChainTooLongError{Path: []string{stack[0]}, Past: name}
		}
		below[name] = deepest
		out = append(out, name)
		return nil
	}
	return out, walk(top, "", nil)
}

func originOf(p Placed) Origin {
	name := p.File.Name
	if name == "" {
		name = p.File.Overrides
	}
	return Origin{Name: name, Kind: p.Kind, File: p.Path}
}

// invariant: the files merge in the chain's order, a later file's value
// replacing an earlier one's: `machine` and `vm` key by key, `vm.mounts`,
// `vm.image.bake`, `prefer` and `ports` whole (an overlay's `ports` key by
// key), whatever extends what; an app or a setting by name over what the
// files it extends said, beside what a file it does not extend said
// (claimed); accounts by program (mergeAccounts); each file's `run` a
// Script of its own (mergeRun). What two files beside each other give is
// judged once on the whole (merged.judged).
func merge(files []Placed) merged {
	m := merged{values: map[string]Value{}, set: map[string]listed[catalog.SettingEntry]{}, ancestors: map[string][]string{}, appHeads: map[string][]claim[catalog.AppEntry]{}, setHeads: map[string][]claim[catalog.SettingEntry]{}, accounts: map[catalog.ProgramName][]account{}}
	for _, p := range files {
		o := originOf(p)
		f := p.File
		u := unitOf(p)
		if p.Kind != Overlay {
			var above []string
			for _, e := range f.Extends {
				above = append(append(above, e), m.ancestors[e]...)
			}
			m.ancestors[u], m.top = above, u
		}
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
			mergeApps(&m, *a, o, u)
		}
		if s := f.Settings; s != nil {
			mergeSettings(&m, *s, o, u)
		}
		if a := f.Accounts; a != nil {
			mergeAccounts(&m, *a, p)
		}
		if ports := f.Ports; ports != nil {
			mergePorts(&m, *ports, p)
		}
		if r := f.Run; r != nil {
			mergeRun(&m, *r, o)
		}
	}
	return m
}

// invariant: the system a chain is judged and resolved for, one for every
// surface that reads it: the chain's own word, and on a machine (this
// named) what that machine is where the chain says nothing of it, put
// among the values as this machine's, unless the chain says another OS.
// A template's, a mixin's and a VM's to come are the chain's alone.
func (m *merged) systemFor(name string, this Machine) catalog.System {
	if said := m.values["machine.system.os"].Text; said == "" || said == string(this.System.OS) {
		mine := Origin{Name: name, Kind: ThisMachine}
		for _, kv := range [][2]string{{"os", string(this.System.OS)}, {"version", this.System.Version}, {"arch", string(this.System.Arch)}} {
			if _, ok := m.values["machine.system."+kv[0]]; !ok && kv[1] != "" {
				m.put("machine.system."+kv[0], kv[1], mine)
			}
		}
	}
	return m.said()
}

// value: the system the chain itself says, empty where it says none.
func (m merged) said() catalog.System {
	kind := catalog.MachineRole(m.values["machine.kind"].Text)
	return catalog.System{OS: catalog.OS(m.values["machine.system.os"].Text), Version: m.values["machine.system.version"].Text, Arch: catalog.Arch(m.values["machine.system.arch"].Text), Machine: kind}
}

// value: the Machinefile a file belongs to: its own name, or for an
// overlay the file it changes.
func unitOf(p Placed) string {
	if p.Kind == Overlay {
		return p.File.Overrides
	}
	return p.File.Name
}

// invariant: a file's word on a name stands over what the files it
// extends said of it, and beside what a file it does not extend said:
// the claims kept are those no later file of their line overrode.
func claimed[T any](m merged, heads map[string][]claim[T], name string, c claim[T]) {
	heads[name] = append(slices.DeleteFunc(heads[name], func(h claim[T]) bool {
		return h.unit == c.unit || slices.Contains(m.ancestors[c.unit], h.unit)
	}), c)
}

// invariant: the merge judged once whole, against the catalog and the
// system it resolves for: what it merged into that cannot be read, then
// two files of which neither extends the other that give one app or
// setting different entries, or one program's account under each key,
// refused naming both and the name. Entries
// compare by what they mean, for the system the chain is judged for
// (merged.systemFor): a range by how far it lets the app move, a
// manager the package takes by default there as none (with no system
// known, a manager named beside none is refused as unjudged), and never by
// optional: a name one of them requires is required. The same entry twice
// is one, and a file that extends both settles it by saying it. The apps
// are judged first, then the settings, then the accounts, each by name.
func (m *merged) judged(cat catalog.Catalog, sys catalog.System) error {
	if m.refused != nil {
		return m.refused
	}
	for _, name := range slices.Sorted(maps.Keys(m.appHeads)) {
		heads := m.appHeads[name]
		same := func(bare bool) func(x, y catalog.AppEntry) bool {
			return func(x, y catalog.AppEntry) bool {
				x, y = plainApp(cat, sys, m.prefer, x), plainApp(cat, sys, m.prefer, y)
				if !x.Version.Same(y.Version) {
					return false
				}
				x.Version, y.Version = catalog.Range{}, catalog.Range{}
				if bare && (x.Manager == "") != (y.Manager == "") {
					x.Manager, y.Manager = "", ""
				}
				return reflect.DeepEqual(x, y)
			}
		}
		line := func(e catalog.AppEntry) int { return e.Line }
		if a, b, ok := differ(heads, same(false), line); ok {
			_, _, beyond := differ(heads, same(true), line)
			return &SiblingConflictError{Kind: "app", Name: name, First: a, Second: b, Top: m.top, Unjudged: sys.OS == "" && !beyond}
		}
		if i := slices.IndexFunc(m.apps, func(l listed[catalog.AppEntry]) bool { return l.name == name }); i >= 0 {
			if best, ok := specific(heads); ok {
				m.apps[i].entry, m.apps[i].origin = best.entry, best.origin
			}
			if requiredBy(heads, func(e catalog.AppEntry) bool { return e.Optional }) {
				m.apps[i].entry.Optional = false
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(m.setHeads)) {
		heads := m.setHeads[name]
		a, b, ok := differ(heads, func(x, y catalog.SettingEntry) bool {
			return reflect.DeepEqual(plainSetting(x), plainSetting(y))
		}, func(e catalog.SettingEntry) int { return e.Line })
		if ok {
			return &SiblingConflictError{Kind: "setting", Name: name, First: a, Second: b, Top: m.top}
		}
		if l, held := m.set[name]; held && requiredBy(heads, func(e catalog.SettingEntry) bool { return e.Optional }) {
			l.entry.Optional = false
			m.set[name] = l
		}
	}
	return m.accountsJudged()
}

// value: an app entry as it compares: no line, no optional, options empty
// as none, and the manager the package takes by default for this system as
// none.
func plainApp(cat catalog.Catalog, sys catalog.System, prefer []catalog.ManagerKey, e catalog.AppEntry) catalog.AppEntry {
	e.Line, e.Optional = 0, false
	if len(e.Options) == 0 {
		e.Options = nil
	}
	if e.Manager != "" && sys.OS != "" {
		if it, ok := cat.Package(e.Name); ok {
			var matching []catalog.MethodSpec
			for _, m := range it.Package.Methods {
				if m.Matches(sys) {
					matching = append(matching, m)
				}
			}
			if len(matching) > 0 && cat.Ordered(matching, sys.OS, prefer)[0].Key() == string(e.Manager) {
				e.Manager = ""
			}
		}
	}
	return e
}

func plainSetting(e catalog.SettingEntry) catalog.SettingEntry {
	e.Line, e.Optional = 0, false
	if len(e.Values) == 0 {
		e.Values = nil
	}
	return e
}

// invariant: of siblings that mean one entry, the merge keeps the most
// specific whatever their order: one that names its manager over one
// that leaves it, then a required one over an optional one; between two
// equal, the one whose version reads first, then whose file's name does.
// No claim kept when the name is removed.
func specific(heads []claim[catalog.AppEntry]) (claim[catalog.AppEntry], bool) {
	if len(heads) == 0 || heads[0].removed {
		return claim[catalog.AppEntry]{}, false
	}
	score := func(e catalog.AppEntry) int {
		s := 0
		if e.Manager != "" {
			s += 2
		}
		if !e.Optional {
			s++
		}
		return s
	}
	return slices.MinFunc(heads, func(x, y claim[catalog.AppEntry]) int {
		return cmp.Or(cmp.Compare(score(y.entry), score(x.entry)), cmp.Compare(x.entry.Version.String(), y.entry.Version.String()), cmp.Compare(x.origin.Name, y.origin.Name))
	}), true
}

// value: one of the claims kept lists the name and does not mark it
// optional.
func requiredBy[T any](heads []claim[T], optional func(T) bool) bool {
	return slices.ContainsFunc(heads, func(h claim[T]) bool { return !h.removed && !optional(h.entry) })
}

func differ[T any](heads []claim[T], same func(T, T) bool, line func(T) int) (first, second Said, differs bool) {
	if len(heads) < 2 {
		return first, second, false
	}
	for _, h := range heads[1:] {
		if h.removed != heads[0].removed || !h.removed && !same(h.entry, heads[0].entry) {
			return Said{Name: heads[0].origin.Name, File: heads[0].origin.File, Line: line(heads[0].entry)}, Said{Name: h.origin.Name, File: h.origin.File, Line: line(h.entry)}, true
		}
	}
	return first, second, false
}

// invariant: a Machinefile's accounts join those of the files before it
// in the chain, each program said by the file that first lists it under
// its key. A key a file gives as an empty list (`signins: []`), and every
// key an overlay gives, clears what the files before it gave under that
// key, an overlay's then giving its own. A program a file lists under the
// other key than a file it extends, or itself, moves there: the most
// specific file's word stands. What two files of which neither extends the
// other give it under each key stands beside, judged once on the whole
// merge (merged.judged).
func mergeAccounts(m *merged, a catalog.AccountsSpec, p Placed) {
	o, u := originOf(p), unitOf(p)
	for _, list := range []struct {
		key, other, text string
		names            []catalog.ProgramName
	}{{signInKey, grantKey, "signs in", a.SignIns}, {grantKey, signInKey, "granted", a.Grants}} {
		if list.names == nil {
			continue
		}
		if p.Kind == Overlay || len(list.names) == 0 {
			m.drop(strings.TrimSuffix(list.key, "."))
			for n, heads := range m.accounts {
				m.accounts[n] = slices.DeleteFunc(heads, func(h account) bool { return h.key == list.key })
			}
		}
		for _, n := range list.names {
			heads := slices.DeleteFunc(m.accounts[n], func(h account) bool {
				return h.key == list.other && (h.unit == u || slices.Contains(m.ancestors[u], h.unit))
			})
			if !slices.ContainsFunc(heads, func(h account) bool { return h.key == list.other }) {
				m.drop(list.other + string(n))
			}
			if !slices.ContainsFunc(heads, func(h account) bool { return h.key == list.key }) {
				heads = append(heads, account{unit: u, key: list.key, origin: o})
				m.put(list.key+string(n), list.text, o)
			}
			m.accounts[n] = heads
		}
	}
}

// invariant: a program two files of which neither extends the other give
// under each key, one signing in to it and the other granting it, is
// refused naming both files; the file that extends both settles it by
// listing it under one, or its overlay by either.
func (m merged) accountsJudged() error {
	for _, n := range slices.Sorted(maps.Keys(m.accounts)) {
		heads := m.accounts[n]
		if len(heads) == 0 {
			continue
		}
		if i := slices.IndexFunc(heads, func(h account) bool { return h.key != heads[0].key }); i > 0 {
			return &SiblingConflictError{Kind: "account", Name: string(n), First: Said{Name: heads[0].origin.Name, File: heads[0].origin.File}, Second: Said{Name: heads[i].origin.Name, File: heads[i].origin.File}, Top: m.top}
		}
	}
	return nil
}

// invariant: a Machinefile's ports replace those of the files it extends
// whole; an overlay's change them key by key (PortRights.Over), and a
// merge that cannot be read is refused at the overlay.
func mergePorts(m *merged, ports catalog.PortRights, p Placed) {
	o := originOf(p)
	said := ports
	if p.Kind == Overlay && m.ports != nil {
		said = m.ports.Over(ports)
	}
	m.ports = &said
	whole := p.Kind != Overlay
	if whole {
		m.drop("ports")
	}
	if whole || ports.Open != nil {
		m.drop("ports.open")
		if len(ports.Open) > 0 {
			m.put("ports.open", spans(ports.Open), o)
		}
	}
	if whole || ports.To != "" {
		m.drop("ports.to")
		if ports.To != "" {
			m.put("ports.to", string(ports.To), o)
		}
	}
	if whole || ports.Except != nil {
		m.drop("ports.except")
		if len(ports.Except) > 0 {
			m.put("ports.except", ints(ports.Except), o)
		}
	}
	for _, port := range ports.Audienced() {
		m.put("ports.audiences."+strconv.Itoa(port), string(ports.Audiences[port]), o)
	}
	if problem := said.Problem(); p.Kind == Overlay && problem != "" && m.refused == nil {
		m.refused = mergeRefused(p, "ports", problem)
	}
}

func mergeRefused(p Placed, key, problem string) error {
	before := p.File.Overrides + "'s chain"
	if p.Kind != Overlay {
		before = "the files before it in the chain"
	}
	return refusedFile("machinefile", &catalog.FileError{File: p.Path, Problem: "its " + key + " with those of " + before + ": " + problem})
}

// invariant: each file's lines of a phase are a Script of their own, after
// those of the files before it in the chain: no file's lines replace
// another's. A phase a file says empty (`boot: ""`), which an overlay may
// say too, clears what every file before it in the chain gave that phase,
// a sibling's before it included, as an empty accounts list does. The
// clearing stays in the merge as a Script with no lines, which explain
// says (`cleared by <file>`) and no run runs.
func mergeRun(m *merged, r catalog.RunSpec, o Origin) {
	for _, p := range r.Phases() {
		phase := map[string]RunPhase{"image": RunImage, "boot": RunBoot, "upgrade": RunUpgrade}[p[0]]
		if strings.TrimSpace(p[1]) != "" {
			m.scripts = append(m.scripts, Script{Phase: phase, Lines: p[1], Origin: o})
			continue
		}
		m.scripts = append(slices.DeleteFunc(m.scripts, func(s Script) bool { return s.Phase == phase }), Script{Phase: phase, Origin: o})
	}
}

// value: the scripts in the order they run: the image's, the boot's, the
// upgrade's, each phase's in the chain's order, a clearing none.
func (m merged) phased() []Script {
	return slices.DeleteFunc(m.inOrder(), func(s Script) bool { return s.Lines == "" })
}

// value: the scripts in that order, the clearings among them.
func (m merged) inOrder() []Script {
	return slices.Concat(ScriptsOf(m.scripts, RunImage), ScriptsOf(m.scripts, RunBoot), ScriptsOf(m.scripts, RunUpgrade))
}

// value: the values of the owner's lines, one per Script, each its lines
// joined, said by its file; a phase a file cleared reads `cleared by
// <file>`.
func (m merged) runValues() []Value {
	scripts := m.inOrder()
	out := make([]Value, 0, len(scripts))
	for _, s := range scripts {
		if s.Lines == "" {
			out = append(out, Value{Key: "run." + string(s.Phase), Text: "cleared by " + s.Origin.Name, Origin: s.Origin})
			continue
		}
		text, withheld := secret.Screen(strings.Join(ScriptLines(s.Lines), "; "))
		out = append(out, Value{Key: "run." + string(s.Phase), Text: text, Origin: s.Origin, Withheld: withheld})
	}
	return out
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

func mergeApps(m *merged, a catalog.AppList, o Origin, unit string) {
	for _, r := range a.Remove {
		m.apps = slices.DeleteFunc(m.apps, func(l listed[catalog.AppEntry]) bool { return l.name == r })
		m.drop("apps." + r)
		claimed(*m, m.appHeads, r, claim[catalog.AppEntry]{unit: unit, origin: o, removed: true})
	}
	for _, e := range slices.Concat(a.List, a.Add) {
		claimed(*m, m.appHeads, e.Name, claim[catalog.AppEntry]{unit: unit, origin: o, entry: e})
		if i := slices.IndexFunc(m.apps, func(l listed[catalog.AppEntry]) bool { return l.name == e.Name }); i >= 0 {
			m.apps = slices.Delete(m.apps, i, i+1)
		}
		m.drop("apps." + e.Name)
		m.apps = append(m.apps, listed[catalog.AppEntry]{name: e.Name, entry: e, origin: o})
	}
}

func mergeSettings(m *merged, s catalog.SettingList, o Origin, unit string) {
	for _, r := range s.Remove {
		delete(m.set, r)
		m.setting = slices.DeleteFunc(m.setting, func(n string) bool { return n == r })
		m.drop("settings." + r)
		claimed(*m, m.setHeads, r, claim[catalog.SettingEntry]{unit: unit, origin: o, removed: true})
	}
	for _, name := range sortedNames(s.Set) {
		claimed(*m, m.setHeads, name, claim[catalog.SettingEntry]{unit: unit, origin: o, entry: s.Set[name]})
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
	// computer has no vm, each package's method is the first of its matching
	// methods in prefer's order, then the system's priority, then the file's
	// order; the steps run requires first, managers before what they install,
	// `after` between two the plan holds; a cycle is refused. An app the
	// lock holds is carried by the lock's method wherever the system and
	// the entry still allow it, at the lock's version where it meets the
	// one asked.
	files, err := lib.chain(name, this.Name)
	if err != nil {
		return Resolved{}, err
	}
	top := lib.Machinefiles[name]
	if top.File.Machine == nil {
		return Resolved{}, &MixinError{Name: name}
	}
	m := merge(files)
	r := Resolved{Name: name, Lock: lock, Kind: catalog.MachineRole(m.values["machine.kind"].Text), System: m.systemFor(name, this)}
	if refused := m.judged(lib.Catalog, r.System); refused != nil {
		return Resolved{}, refused
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
			return Resolved{}, &UnknownPackageError{Name: b.name, By: b.origin.Name, At: b.origin.File}
		}
	}
	for _, a := range m.apps {
		if listed := p.app(a.entry, a.origin, "listed"); listed != nil {
			return Resolved{}, listed
		}
	}
	for _, b := range m.bake {
		if _, ok := p.at[b.name]; ok || slices.ContainsFunc(p.skipped, func(s Skip) bool { return s.Package == b.name }) {
			continue
		}
		if baked := p.app(catalog.AppEntry{Name: b.name}, b.origin, "baked"); baked != nil {
			return Resolved{}, baked
		}
	}
	for _, s := range m.setting {
		l := m.set[s]
		if l.entry.From == catalog.FromComputer && r.Kind == catalog.RoleComputer {
			return Resolved{}, &BadValueError{Package: s, Key: "from", Problem: "a computer takes no values from a computer: from is a VM's", At: at(l.origin, l.entry.Line)}
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
	r.Steps, r.Skipped, r.Scripts = filed(p.baked(steps, m.bake)), p.skipped, m.phased()
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

// value: a package by its name, or by a name it was renamed from.
func (p *planner) lookup(name string) (catalog.Placed, bool) {
	it, ok := p.cat.Package(name)
	if !ok {
		for _, c := range p.cat.Packages {
			if slices.Contains(c.Package.RenamedFrom, name) {
				it, ok = c, true
			}
		}
	}
	return it, ok
}

// value: where an entry of a file is written, `<file>:<line>`, for a
// refusal to point at; empty for a package another one brings in.
func at(o Origin, line int) string {
	switch {
	case o.File == "":
		return ""
	case line > 0:
		return o.File + ":" + strconv.Itoa(line)
	}
	return o.File
}

func (p *planner) pkg(name string, by Origin, where string, want catalog.Kind) (catalog.Placed, error) {
	it, ok := p.lookup(name)
	switch {
	case !ok:
		return catalog.Placed{}, &UnknownPackageError{Name: name, By: by.Name, At: where}
	case want == catalog.KindSetting && it.Package.Kind != catalog.KindSetting:
		return catalog.Placed{}, &UnknownPackageError{Name: name, By: by.Name, Is: "an app: list it under apps", At: where}
	case want != catalog.KindSetting && it.Package.Kind == catalog.KindSetting:
		return catalog.Placed{}, &UnknownPackageError{Name: name, By: by.Name, Is: "a setting: list it under settings", At: where}
	}
	return it, nil
}

func packageOrigin(it catalog.Placed) Origin {
	return Origin{Name: it.Package.Name, Kind: OriginKind(it.Origin), File: it.File}
}

// invariant: what the image bakes is its bake list, each step of the
// image phase whatever that list says (a child's list replaces its
// parent's, never the parent's kernel), and, step by step, what a baked
// step requires (its requires, its provider, its manager), so `apply
// --phase image` never runs a package whose requirement it left.
func (p *planner) baked(steps []Step, bake []listed[struct{}]) []Step {
	at := make(map[string]int, len(steps))
	for i, s := range steps {
		at[s.Package] = i
	}
	queue := make([]string, 0, len(bake))
	for _, b := range bake {
		queue = append(queue, b.name)
	}
	for _, s := range steps {
		if s.Phase == catalog.PhaseImage {
			queue = append(queue, s.Package)
		}
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		i, ok := at[name]
		if !ok || steps[i].Baked {
			continue
		}
		steps[i].Baked = true
		it, _ := p.cat.Package(name)
		for _, l := range p.cat.Links(it.Package, steps[i].Method, p.sys) {
			queue = append(queue, l.Packages...)
		}
	}
	return steps
}

func (p *planner) app(e catalog.AppEntry, o Origin, why string) error {
	where := at(o, e.Line)
	it, err := p.pkg(e.Name, o, where, catalog.KindApp)
	if err != nil {
		return err
	}
	if bad := checkValues(it.Package, e.Options, where); bad != nil {
		return bad
	}
	step, err := p.choose(it, e.Manager, e.Version, why, where)
	if err != nil {
		var none *NoMethodError
		if e.Optional && errors.As(err, &none) {
			p.skipped = append(p.skipped, Skip{Package: it.Package.Name, Why: p.skippedFor(it, e.Manager), Origin: o})
			return nil
		}
		return err
	}
	step.Origin, step.Values, step.Optional = o, e.Options, e.Optional
	p.add(step)
	return nil
}

func (p *planner) setting(name string, e catalog.SettingEntry, o Origin) error {
	where := at(o, e.Line)
	it, err := p.pkg(name, o, where, catalog.KindSetting)
	if err != nil {
		return err
	}
	if bad := checkValues(it.Package, e.Values, where); bad != nil {
		return bad
	}
	step, err := p.choose(it, "", catalog.Range{}, "listed", where)
	if err != nil {
		var none *NoMethodError
		if e.Optional && errors.As(err, &none) {
			p.skipped = append(p.skipped, Skip{Package: name, Why: "optional, no method for " + p.system(), Origin: o})
			return nil
		}
		return err
	}
	step.Origin, step.Values, step.From, step.Optional = o, e.Values, e.From, e.Optional
	p.add(step)
	return nil
}

// value: why an optional package is skipped: no method for the system, or,
// when its entry asks a manager, none by that manager, with the methods
// it has there, so a manager pinned by mistake reads apart from a package
// the system lacks.
func (p *planner) skippedFor(it catalog.Placed, manager catalog.ManagerKey) string {
	if manager == "" {
		return "optional, no method for " + p.system()
	}
	var has []string
	for _, m := range it.Package.Methods {
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
	p.at[s.Package] = len(p.steps)
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
	for _, m := range it.Package.Methods {
		if m.Matches(p.sys) && (manager == "" || m.Key() == string(manager)) {
			matching = append(matching, m)
		}
	}
	if len(matching) == 0 {
		return Step{}, &NoMethodError{Package: it.Package.Name, System: p.system(), Manager: string(manager), Why: why, At: where}
	}
	matching = p.cat.Ordered(matching, p.sys.OS, p.prefer)
	chosen, because := matching[0], p.because(matching[0], len(matching))
	locked, ok := p.lock.App(it.Package.Name)
	if ok && it.Package.Kind != catalog.KindSetting {
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
		return Step{}, &UnpinnableError{Package: it.Package.Name, Version: version.String(), Method: chosen.Key(), Holds: chosen.HoldWords(), Alternative: alt, At: where}
	}
	held, recorded := lockedAt(ok, version, locked.Version), ""
	if !chosen.Pins() {
		held, recorded = "", held
	}
	return Step{
		Package: it.Package.Name, Kind: it.Package.Kind, Method: chosen.Method(), Chosen: because, Spec: chosen,
		Version: version, UpdatesSelf: it.Package.Updates == catalog.UpdatesSelf, Locked: held, Recorded: recorded, Lock: lockFile(ok, p.lock.File), Phase: cmp.Or(it.Package.Phase, catalog.PhaseApply), Needs: slices.Concat(it.Package.Needs, chosen.Needs),
		Restarts: it.Package.AfterApply, Why: why, Defined: packageOrigin(it),
		Chapter: chapterOf(it.Package.Kind), Rule: ruleOf(it.Package, version),
	}, nil
}

func ruleOf(pkg catalog.Package, version catalog.Range) string {
	switch {
	case pkg.Kind == catalog.KindSetting:
		return ""
	case pkg.Updates == catalog.UpdatesSelf:
		return "updates itself"
	}
	return version.Rule()
}

func filed(steps []Step) []Step {
	// invariant: a step brought in by a package has that package's
	// definition as its origin; its file is the bringer's, followed until
	// a step a Machinefile lists, and a loop of bringers ends at the step
	// it started from.
	by := make(map[string]int, len(steps))
	for i, s := range steps {
		by[s.Package] = i
	}
	for i := range steps {
		o, seen := steps[i].Origin, map[string]bool{steps[i].Package: true}
		for {
			j, ok := by[o.Name]
			if !ok || seen[o.Name] || steps[j].Defined != o {
				break
			}
			seen[o.Name] = true
			o = steps[j].Origin
		}
		steps[i].File = o
	}
	return steps
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

// invariant: a package of the plan brings in every package it leans on, each
// resolved like a listed one: what it requires, its method's provider and
// its method's manager; a setting's requires-one-of is met by any one the
// plan holds, else by its first.
func (p *planner) implied() error {
	// invariant: a step brought in is itself judged: the walk goes on while
	// the plan grows.
	for next := 0; next < len(p.steps); {
		s := p.steps[next]
		next++
		it, _ := p.cat.Package(s.Package)
		for _, l := range p.cat.Links(it.Package, s.Method, p.sys) {
			if slices.ContainsFunc(l.Packages, p.holds) {
				continue
			}
			dep, err := p.pkg(l.Packages[0], Origin{Name: s.Package}, "", catalog.KindApp)
			if err != nil {
				return err
			}
			step, err := p.choose(dep, "", catalog.Range{}, broughtBy(l.Kind, s.Package), "")
			if err != nil {
				return err
			}
			step.Origin = s.Defined
			p.add(step)
		}
	}
	return nil
}

func broughtBy(k catalog.LinkKind, pkg string) string {
	switch k {
	case catalog.LinkManager:
		return "the manager of " + pkg
	case catalog.LinkProvidedBy:
		return "the provider of " + pkg
	case catalog.LinkRequires, catalog.LinkRequiresOneOf:
	}
	return "required by " + pkg
}

func (p *planner) holds(name string) bool { _, ok := p.at[name]; return ok }

// invariant: a stable order: among the steps whose dependencies ran, the
// first in the plan's order runs next; what never becomes ready is in a
// cycle, which is printed.
func (p *planner) ordered() ([]Step, error) {
	deps := make([][]int, len(p.steps))
	for i, s := range p.steps {
		it, _ := p.cat.Package(s.Package)
		before := slices.Clone(it.Package.After)
		for _, l := range p.cat.Links(it.Package, s.Method, p.sys) {
			before = append(before, l.Packages...)
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
			p.steps[i].Waits = append(p.steps[i].Waits, p.steps[j].Package)
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
			return nil, &CycleError{Path: p.cycle(deps, done), Of: "the packages' requires and after"}
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
		names = append(names, p.steps[i].Package)
	}
	return append(names, names[0])
}

// invariant: every value names an option of its package and fits its type; a
// secret's is a `vault:` reference, never the secret written out.
func checkValues(it catalog.Package, values map[string]catalog.OptionValue, where string) error {
	for _, k := range sortedNames(values) {
		v := values[k]
		i := slices.IndexFunc(it.Options, func(o catalog.Option) bool { return o.Name == k })
		if i < 0 {
			return &BadValueError{Package: it.Name, Key: k, Problem: "no option of " + it.Name, At: where}
		}
		o := it.Options[i]
		vault := strings.HasPrefix(string(v), catalog.VaultPrefix)
		switch {
		case o.Secret && !vault:
			return &BadValueError{Package: it.Name, Key: k, Problem: "a secret is a vault: reference (`vault:<name>`), never written out", At: where}
		case vault && o.Type != catalog.OptionString:
			return &BadValueError{Package: it.Name, Key: k, Problem: "a vault: reference is a string option's value", At: where}
		case !vault:
			if problem := o.Check(v); problem != "" {
				return &BadValueError{Package: it.Name, Key: k, Problem: problem, At: where}
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
	out = append(out, m.family("machine", "vm", "prefer")...)
	for _, s := range r.Steps {
		group := "apps."
		if s.Kind == catalog.KindSetting {
			group = "settings."
		}
		key := group + s.Package
		text := s.Why
		if !s.Version.IsZero() {
			text = s.Version.String()
		}
		chose := s.Defined
		if s.Lock != "" {
			chose = Origin{Name: r.Name, Kind: Locked, File: s.Lock}
		}
		out = append(out, Value{Key: key, Text: text, Origin: s.Origin}, Value{Key: key + ".method", Text: s.Method, Origin: chose}, Value{Key: key + ".because", Text: s.Chosen, Origin: chose})
		if s.Locked != "" {
			out = append(out, Value{Key: key + ".locked", Text: s.Locked, Origin: chose})
		}
		for _, k := range sortedNames(s.Values) {
			text, withheld := withhold(string(s.Values[k]))
			def, defined := optionDefault(c, s, k)
			out = append(out, Value{Key: key + "." + k, Text: text, Origin: s.Origin, Withheld: withheld, Default: def, Defined: defined})
		}
		if s.From != "" {
			out = append(out, Value{Key: key + ".from", Text: s.From, Origin: s.Origin})
		}
	}
	for _, s := range r.Skipped {
		group := "apps."
		if it, ok := c.Package(s.Package); ok && it.Package.Kind == catalog.KindSetting {
			group = "settings."
		}
		out = append(out, Value{Key: group + s.Package, Text: s.Why, Origin: s.Origin})
	}
	return append(append(out, m.family("accounts", "ports")...), m.runValues()...)
}

func optionDefault(c catalog.Catalog, s Step, name string) (string, *Origin) {
	// invariant: the package's default of an option, read from its definition
	// and never a secret's, and the file that defines it.
	p, ok := c.Package(s.Package)
	if !ok {
		return "", nil
	}
	i := slices.IndexFunc(p.Package.Options, func(o catalog.Option) bool { return o.Name == name })
	if i < 0 {
		return "", nil
	}
	defined := s.Defined
	return p.Package.Options[i].Shown(), &defined
}

func (m merged) family(families ...string) []Value {
	var out []Value
	for _, f := range families {
		for _, k := range m.order {
			if head, _, _ := strings.Cut(k, "."); head == f {
				out = append(out, m.values[k])
			}
		}
	}
	for i, v := range out {
		if !v.Withheld {
			out[i].Text, out[i].Withheld = secret.Screen(v.Text)
		}
	}
	return out
}

func withhold(text string) (string, bool) {
	// value: an option's value as an answer shows it: the whole of it is the
	// word withheld when it holds a secret, since half a token is still one.
	if secret.Shaped(text) {
		return secret.Marker, true
	}
	return text, false
}

func Own(lib Library, name string) (Resolved, error) {
	// invariant: a mixin answers the values its chain says, in the
	// Machinefile's reading order, each app and setting listed as the
	// strongest file lists it; none is resolved against the catalog or a
	// system, so a refusal of a plan is never raised here, and a refusal of
	// the merge is, as the resolution raises it.
	files, err := lib.chain(name, "")
	if err != nil {
		return Resolved{}, err
	}
	m := merge(files)
	if refused := m.judged(lib.Catalog, m.systemFor(name, Machine{})); refused != nil {
		return Resolved{}, refused
	}
	r := Resolved{Name: name}
	for _, f := range files {
		r.Files = append(r.Files, originOf(f))
	}
	r.Values = m.family("machine", "vm", "prefer")
	for _, a := range m.apps {
		r.Values = append(r.Values, Value{Key: "apps." + a.name, Text: "listed", Origin: a.origin})
		for _, k := range sortedNames(a.entry.Options) {
			text, withheld := withhold(string(a.entry.Options[k]))
			r.Values = append(r.Values, Value{Key: "apps." + a.name + "." + k, Text: text, Origin: a.origin, Withheld: withheld})
		}
	}
	for _, n := range m.setting {
		l := m.set[n]
		r.Values = append(r.Values, Value{Key: "settings." + n, Text: "listed", Origin: l.origin})
		for _, k := range sortedNames(l.entry.Values) {
			text, withheld := withhold(string(l.entry.Values[k]))
			r.Values = append(r.Values, Value{Key: "settings." + n + "." + k, Text: text, Origin: l.origin, Withheld: withheld})
		}
		if l.entry.From != "" {
			r.Values = append(r.Values, Value{Key: "settings." + n + ".from", Text: l.entry.From, Origin: l.origin})
		}
	}
	r.Values = append(append(r.Values, m.family("accounts", "ports")...), m.runValues()...)
	return r, nil
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
