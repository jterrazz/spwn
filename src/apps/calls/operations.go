package calls

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
)

const (
	verbInstall   = "install"
	verbUninstall = "uninstall"
	verbUpdate    = "update"
)

func (s *Shelf) Install(ctx context.Context, r apps.InstallRequest, p call.Progress) ([]apps.Outcome, error) {
	found, err := s.Find(r.Names)
	if err != nil {
		return nil, err
	}
	requested, err := s.by(found, r.Method, r.Version)
	if err != nil {
		return nil, err
	}
	run, requiredBy, err := s.withRequirements(ctx, requested)
	if err != nil {
		return nil, err
	}
	inRun := map[string]bool{}
	ran := map[string]catalog.App{}
	for _, t := range run {
		inRun[t.Name] = true
		ran[t.Name] = t
	}
	// invariant: an app already installed is named by the method that
	// holds it, the one its row reads it by, and its version there.
	states := map[string]catalog.State{}
	for i, rep := range s.reports(ctx, run) {
		states[rep.App.Name] = rep.State
		if rep.State == catalog.StateOK {
			ran[run[i].Name] = s.holding(run[i], rep)
		}
	}
	deps := map[string][]string{}
	for _, t := range run {
		deps[t.Name] = s.requirements(ctx, t)
	}
	ordered := s.topological(run, deps)
	outcomes := map[string]*apps.Outcome{}
	var pending []catalog.App
	for _, t := range ordered {
		if states[t.Name] == catalog.StateOK {
			outcomes[t.Name] = &apps.Outcome{Name: t.Name, Verb: verbInstall, Result: catalog.StateOK, Detail: apps.AlreadyInstalled}
			continue
		}
		pending = append(pending, t)
	}
	for len(pending) > 0 {
		var batch []catalog.App
		if _, ok := s.lined(pending[0]); ok {
			batch = s.bundleBatch(pending, inRun, outcomes, deps)
		} else {
			batch = []catalog.App{pending[0]}
		}
		pending = slices.DeleteFunc(pending, func(t catalog.App) bool { return slices.ContainsFunc(batch, sameName(t)) })
		var ready []catalog.App
		for _, t := range batch {
			if dep := s.failedRequirement(deps[t.Name], outcomes); dep != "" {
				outcomes[t.Name] = &apps.Outcome{Name: t.Name, Verb: verbInstall, Result: catalog.StateBroken, Detail: "requires " + dep + ", which failed"}
				continue
			}
			ready = append(ready, t)
		}
		if len(ready) == 0 {
			continue
		}
		s.announce(p, ready, requiredBy)
		var done []apps.Outcome
		if _, ok := s.lined(ready[0]); ok {
			done = s.bundle(ctx, verbInstall, ready, false)
		} else {
			done = []apps.Outcome{s.landed(ctx, ready[0], s.run(ctx, verbInstall, ready[0], false, contract.Installer.Install))}
		}
		for i := range done {
			o := done[i]
			if o.Result == catalog.StateOK {
				o.Steps = s.then(ctx, s.byName[o.Name], r.Interactive)
			}
			outcomes[o.Name] = &o
		}
	}
	result := make([]apps.Outcome, 0, len(ordered))
	for _, t := range ordered {
		o := outcomes[t.Name]
		if o.Result == catalog.StateOK {
			o.Steps = append(o.Steps, s.supersede(ctx, t)...)
		}
		o.RequiredBy = requiredBy[t.Name]
		if o.Result == catalog.StateOK && o.Detail != apps.AlreadyInstalled {
			o.Path = t.Path
		}
		result = append(result, *o)
	}
	return s.held(ctx, result, ran), nil
}

// invariant: an app comes with what it requires: each requirement missing
// here (an app it names, its method's manager) joins the run, named by the
// apps that require it. One the catalogue does not know, or this machine
// cannot carry, refuses the run before anything is installed, in the words
// of asking it by name (`no app named "y"`, `y lives on Macs only`).
func (s *Shelf) withRequirements(ctx context.Context, requested []catalog.App) ([]catalog.App, map[string][]string, error) {
	states := s.statesWithRequirements(ctx, requested)
	run := slices.Clone(requested)
	inRun := map[string]bool{}
	for _, t := range requested {
		inRun[t.Name] = true
	}
	requiredBy := map[string][]string{}
	for i := 0; i < len(run); i++ {
		t := run[i]
		for _, dep := range s.requirements(ctx, t) {
			if inRun[dep] {
				if requiredBy[dep] != nil && !slices.Contains(requiredBy[dep], t.Name) {
					requiredBy[dep] = append(requiredBy[dep], t.Name)
				}
				continue
			}
			if settled(states[dep]) {
				continue
			}
			d, ok := s.byName[dep]
			if !ok {
				return nil, nil, &apps.UnknownError{Name: dep}
			}
			if _, placed := d.InstallOn(s.o.ThisMachine); !placed {
				return nil, nil, &apps.ElsewhereError{App: dep, Kinds: d.LivesOn()}
			}
			if !s.installer(d).Can().Install {
				return nil, nil, &apps.ManualInstallError{App: d.Name, Hint: s.install(d).Key()}
			}
			if _, read := states[dep]; !read {
				for name, state := range s.statesWithRequirements(ctx, []catalog.App{d}) {
					states[name] = state
				}
				if settled(states[dep]) {
					continue
				}
			}
			inRun[dep] = true
			requiredBy[dep] = []string{t.Name}
			run = append(run, d)
		}
	}
	return run, requiredBy, nil
}

func (s *Shelf) announce(p call.Progress, batch []catalog.App, requiredBy map[string][]string) {
	// invariant: a step is said before it runs, with why it runs. One that
	// hands the terminal to a program which may ask the owner is said as a
	// line of its own, which no spinner draws over while a prompt waits: an
	// install that needs the password, and every curl installer, whose
	// script may read /dev/tty whatever its stdin is.
	if p == nil {
		return
	}
	asks := false
	names := make([]string, 0, len(batch))
	var by []string
	var needs []catalog.Need
	for _, t := range batch {
		names = append(names, t.Name)
		for _, r := range requiredBy[t.Name] {
			if !slices.Contains(by, r) {
				by = append(by, r)
			}
		}
		needs = append(needs, t.Needs...)
		asks = asks || len(t.Needs) > 0 || s.install(t).Method() == catalog.MethodScript
	}
	slices.Sort(needs)
	needs = slices.Compact(needs)
	var why []string
	if asks {
		why = append(why, call.Asks)
	}
	if len(by) > 0 {
		why = append(why, apps.ReasonRequiredBy+strings.Join(by, ", "))
	}
	for _, n := range needs {
		why = append(why, apps.ReasonNeeds+string(n))
	}
	p.Step("Installing "+strings.Join(names, ", "), strings.Join(why, "\n"))
}

// invariant: a superseded formula goes only once the app replacing it is
// installed by its own install, listed by its method or its bundle in
// /Applications, never because the command the formula itself provides
// answers: the formula alone is a half install, named, never removed.
func (s *Shelf) supersede(ctx context.Context, t catalog.App) []apps.Step {
	old, ok := t.Superseded()
	if !ok {
		return nil
	}
	listings := s.listings(ctx, []catalog.App{t})
	if !listedOld(t, listings) || !s.ownInstall(t, listings) {
		return nil
	}
	step := apps.Step{Item: t.Supersedes + " formula", Result: catalog.StateOK, Detail: "uninstalled: the app replaces it"}
	if err := s.o.Installers[old.Method()].Remove(ctx, []catalog.Install{old}, s.logs([]catalog.App{t})); err != nil {
		words, why := s.failed(verbUninstall, []catalog.App{t}, err)
		step.Detail, _ = words(t.Name)
		step.Why = why
		step.Result = catalog.StateBroken
	}
	return []apps.Step{step}
}

func (s *Shelf) Uninstall(ctx context.Context, names []string, word bool) ([]apps.Outcome, error) {
	// invariant: without the owner's word for an app (an apply's removal,
	// which the ledger decides), one whose method records what its installs
	// add and recorded adding nothing for it is refused before anything
	// runs: what is there is the owner's.
	requested, err := s.Find(names)
	if err != nil {
		return nil, err
	}
	reports := s.reports(ctx, requested)
	for _, r := range reports {
		if !settled(r.State) {
			return nil, &apps.NotInstalledError{App: r.App.Name}
		}
	}
	held := make([]catalog.App, 0, len(requested))
	for i, t := range requested {
		held = append(held, s.holding(t, reports[i]))
		if !word && s.unrecorded(ctx, held[i]) {
			return nil, &apps.UnrecordedError{App: t.Name}
		}
	}
	out := make([]apps.Outcome, 0, len(requested))
	for _, held := range held {
		o := s.undone(ctx, held, s.run(ctx, verbUninstall, held, word, contract.Installer.Remove), true)
		o.Method = s.install(held).Method()
		out = append(out, o)
	}
	return out, nil
}

func (s *Shelf) Upgrade(ctx context.Context, r apps.UpgradeRequest) ([]apps.Outcome, error) {
	via := map[catalog.Method]bool{}
	for _, k := range r.Via {
		owner, _ := s.cat.Manager(string(k))
		if it, ok := s.cat.Item(owner); ok {
			for _, m := range it.Item.Manages {
				via[catalog.Method(m)] = true
			}
		}
	}
	reaches := func(in catalog.Install) bool {
		can := s.o.Installers[in.Method()].Can()
		if len(r.Via) > 0 {
			return via[in.Method()] && can.Update
		}
		return can.UpdateAll && can.Update
	}
	var asked []catalog.App
	if len(r.Names) > 0 {
		found, err := s.Find(r.Names)
		if err != nil {
			return nil, err
		}
		asked = found
	} else {
		for _, t := range s.Placed() {
			if slices.ContainsFunc(t.InstallsOn(s.o.ThisMachine), reaches) {
				asked = append(asked, t)
			}
		}
	}
	reports := map[string]apps.Report{}
	var candidates []catalog.App
	for i, rep := range s.reports(ctx, asked) {
		reports[rep.App.Name] = rep
		if t := s.holding(asked[i], rep); len(r.Names) > 0 || reaches(s.install(t)) {
			candidates = append(candidates, t)
		}
	}
	ran := map[string]catalog.App{}
	for _, t := range candidates {
		ran[t.Name] = t
	}
	var outcomes []apps.Outcome
	var bundled, single []catalog.App
	for _, t := range candidates {
		// invariant: a bare update and a method's update take the Update
		// available rows only; a named app is updated whenever it is installed.
		state := apps.StateOf(reports[t.Name])
		upgradable := state == apps.UpdateAvailable || (len(r.Names) > 0 && state.Installed())
		switch {
		case reports[t.Name].Detail == DetailAppStore && len(r.Names) > 0:
			outcomes = append(outcomes, apps.Outcome{Name: t.Name, Verb: verbUpdate, Result: catalog.StateAbsent, Detail: "the App Store updates it"})
		case reports[t.Name].Detail == DetailAppStore:
		case !upgradable && len(r.Names) > 0:
			return nil, &apps.NotInstalledError{App: t.Name}
		case !upgradable:
		case s.isLined(t):
			bundled = append(bundled, t)
		default:
			single = append(single, t)
		}
	}
	if len(bundled) > 0 {
		outcomes = append(outcomes, s.bundle(ctx, verbUpdate, bundled, true)...)
	}
	for _, t := range single {
		outcomes = append(outcomes, s.run(ctx, verbUpdate, t, false, contract.Installer.Update))
	}
	return s.held(ctx, outcomes, ran), nil
}

func (s *Shelf) held(ctx context.Context, outcomes []apps.Outcome, ran map[string]catalog.App) []apps.Outcome {
	// invariant: an outcome names the method that holds its app, the one
	// the run installed or updated it by, and the version that method
	// lists once the verb has run, read once per method; a method that
	// lists none, the app's own read of it (its check, its version read).
	versions := map[catalog.Method]map[string]string{}
	for i := range outcomes {
		t, ok := ran[outcomes[i].Name]
		if !ok {
			continue
		}
		in := s.install(t)
		outcomes[i].Method = in.Method()
		if outcomes[i].Result != catalog.StateOK {
			continue
		}
		if !in.Method().KeepsList() {
			outcomes[i].Version = s.readInstalled(ctx, t, apps.Report{}).Version
			continue
		}
		listed, read := versions[in.Method()]
		if !read {
			told, cancel := context.WithTimeout(ctx, inventoryBudget)
			listed, _ = s.installer(t).Inventory(told)
			cancel()
			versions[in.Method()] = listed
		}
		outcomes[i].Version = listed[in.Key()]
	}
	return outcomes
}

func (s *Shelf) bundleBatch(pending []catalog.App, inRequest map[string]bool, done map[string]*apps.Outcome, deps map[string][]string) []catalog.App {
	var batch []catalog.App
	for _, t := range pending {
		if !s.isLined(t) {
			continue
		}
		ready := true
		for _, dep := range deps[t.Name] {
			if inRequest[dep] && done[dep] == nil && !slices.ContainsFunc(batch, func(b catalog.App) bool { return b.Name == dep }) {
				ready = false
			}
		}
		if ready {
			batch = append(batch, t)
		}
	}
	return batch
}

func (s *Shelf) isLined(t catalog.App) bool {
	_, ok := s.lined(t)
	return ok
}

func (s *Shelf) bundle(ctx context.Context, verb string, list []catalog.App, update bool) []apps.Outcome {
	lines := make([]string, 0, len(list))
	for _, t := range list {
		l, _ := s.lined(t)
		lines = append(lines, l.Line(s.install(t)))
	}
	failure, why := func(string) (string, string) { return "", "" }, apps.Why("")
	if err := s.o.Bundler.Bundle(ctx, lines, update, s.logs(list)); err != nil {
		failure, why = s.failed(verb, list, err)
	}
	out := make([]apps.Outcome, 0, len(list))
	if verb == verbUpdate {
		for _, t := range list {
			o := apps.Outcome{Name: t.Name, Verb: verb, Result: catalog.StateOK}
			if why != "" {
				o.Detail, o.Log = failure(t.Name)
				o.Result, o.Why = catalog.StateBroken, why
			}
			out = append(out, o)
		}
		return out
	}
	// invariant: one exit code answers for N tools, so each tool's verdict is
	// whether the inventory lists it installed once the bundle has run, behind
	// its newest or not: a bundle uses what is there and upgrades nothing. A
	// failed bundle's tool counts only by its own install, so a command
	// another install provides (a superseded formula's) never passes for it.
	states := s.states(ctx, list)
	var listings map[catalog.Method]listing
	if why != "" {
		listings = s.listings(ctx, list)
	}
	for _, t := range list {
		o := apps.Outcome{Name: t.Name, Verb: verb, Result: catalog.StateOK}
		if !settled(states[t.Name]) || why != "" && !s.ownInstall(t, listings) {
			o.Detail, o.Log = failure(t.Name)
			o.Result, o.Why = catalog.StateBroken, why
			if why == "" {
				o.Detail = "not listed after " + s.o.Bundler.Name()
			}
		}
		out = append(out, o)
	}
	return out
}

// invariant: a script reports only its own exit code, so an install that left
// none of the commands the item provides behind is a failure, never an OK.
func (s *Shelf) landed(ctx context.Context, t catalog.App, o apps.Outcome) apps.Outcome {
	if o.Result != catalog.StateOK || s.install(t).Method() != catalog.MethodScript || len(t.Provides) == 0 {
		return o
	}
	if !settled(s.states(ctx, []catalog.App{t})[t.Name]) {
		o.Result, o.Detail = catalog.StateBroken, "installed, but "+t.Provides[0]+" is not found on the search path"
	}
	return o
}

type verbOf func(contract.Installer, context.Context, []catalog.Install, contract.Logs) error

func (s *Shelf) run(ctx context.Context, verb string, t catalog.App, word bool, op verbOf) apps.Outcome {
	o := apps.Outcome{Name: t.Name, Verb: verb, Result: catalog.StateOK}
	err := op(s.serving(t, word), ctx, []catalog.Install{s.install(t)}, s.logs([]catalog.App{t}))
	var signedOut *apps.SignedOutError
	var notYet *apps.NotYetProvidedError
	switch {
	case errors.As(err, &signedOut):
		o.Result, o.Detail = catalog.StateBroken, signedOut.Error()
	case errors.As(err, &notYet):
		o.Result, o.Why, o.Detail = catalog.StateDegraded, apps.WhyPartial, s.firstOpen(t, notYet)
	case err != nil:
		words, why := s.failed(verb, []catalog.App{t}, err)
		o.Detail, o.Log = words(t.Name)
		o.Why = why
		o.Result = catalog.StateBroken
	}
	return o
}

func (s *Shelf) then(ctx context.Context, t catalog.App, interactive bool) []apps.Step {
	steps := make([]apps.Step, 0, len(t.Then))
	for _, item := range t.Then {
		if s.cfg == nil {
			steps = append(steps, apps.Step{Item: item, Result: catalog.StateAbsent, Detail: "not applied: station settings apply " + item})
			continue
		}
		step, err := s.cfg.Configure(ctx, item, interactive)
		if err != nil {
			step = apps.Step{Item: item, Result: catalog.StateBroken, Detail: err.Error()}
		}
		steps = append(steps, step)
	}
	return steps
}

// value: what an app's install cannot run without, the catalog's links of
// the install this machine carries, in the order they are said: its
// method's manager while that manager is missing here, its provider, then
// what it requires (an app requires none of a group: only a setting does).
func (s *Shelf) requirements(ctx context.Context, t catalog.App) []string {
	links := s.cat.InstallLinks(t, s.install(t))
	missing := errors.Is(s.installer(t).Ready(ctx), contract.ErrMissing)
	var reqs []string
	for _, kind := range []catalog.LinkKind{catalog.LinkManager, catalog.LinkProvidedBy, catalog.LinkRequires} {
		for _, l := range links {
			if l.Kind == kind && (kind != catalog.LinkManager || missing) && !slices.Contains(reqs, l.Items[0]) {
				reqs = append(reqs, l.Items[0])
			}
		}
	}
	return reqs
}

func (s *Shelf) failedRequirement(deps []string, outcomes map[string]*apps.Outcome) string {
	for _, dep := range deps {
		if o := outcomes[dep]; o != nil && o.Result == catalog.StateBroken {
			return dep
		}
	}
	return ""
}

func (s *Shelf) topological(list []catalog.App, deps map[string][]string) []catalog.App {
	inRequest := map[string]catalog.App{}
	for _, t := range list {
		inRequest[t.Name] = t
	}
	var ordered []catalog.App
	visited := map[string]bool{}
	var visit func(t catalog.App)
	visit = func(t catalog.App) {
		if visited[t.Name] {
			return
		}
		visited[t.Name] = true
		for _, dep := range deps[t.Name] {
			if d, ok := inRequest[dep]; ok {
				visit(d)
			}
		}
		ordered = append(ordered, t)
	}
	for _, t := range list {
		visit(t)
	}
	return ordered
}

func (s *Shelf) States(ctx context.Context, names []string) (map[string]catalog.State, error) {
	list, err := s.Find(names)
	if err != nil {
		return nil, err
	}
	return s.states(ctx, list), nil
}

func (s *Shelf) states(ctx context.Context, list []catalog.App) map[string]catalog.State {
	out := map[string]catalog.State{}
	for _, r := range s.reports(ctx, list) {
		out[r.App.Name] = r.State
	}
	return out
}

func (s *Shelf) statesWithRequirements(ctx context.Context, list []catalog.App) map[string]catalog.State {
	all := map[string]catalog.App{}
	var walk func(t catalog.App)
	walk = func(t catalog.App) {
		if _, ok := all[t.Name]; ok {
			return
		}
		all[t.Name] = t
		for _, dep := range s.requirements(ctx, t) {
			if d, ok := s.byName[dep]; ok {
				if _, placed := d.InstallOn(s.o.ThisMachine); placed {
					walk(d)
				}
			}
		}
	}
	for _, t := range list {
		walk(t)
	}
	flat := make([]catalog.App, 0, len(all))
	for _, t := range all {
		flat = append(flat, t)
	}
	return s.states(ctx, flat)
}

// value: the app is installed by its own install: its method lists it, or
// its bundle is in /Applications.
func (s *Shelf) ownInstall(t catalog.App, listings map[catalog.Method]listing) bool {
	own := s.install(t)
	_, listed := listedAll(listings[own.Method()].keys, own)
	return listed || s.bundled(t)
}

func settled(s catalog.State) bool { return s == catalog.StateOK || s == catalog.StateDegraded }

func sameName(t catalog.App) func(catalog.App) bool {
	return func(o catalog.App) bool { return o.Name == t.Name }
}

// invariant: a command its provider brings at first launch is a next step,
// never a failure: the owner opens the provider once, then asks again.
func (s *Shelf) firstOpen(t catalog.App, e *apps.NotYetProvidedError) string {
	again := "station apps install " + t.Name
	if by, ok := s.byName[e.Item]; ok && by.Bundle != "" {
		return "open " + by.Bundle + " once, then " + again
	}
	return e.Error() + ": open " + e.Item + " once, then " + again
}

const verbMove = "move"

func (s *Shelf) Move(ctx context.Context, name string, to catalog.Method, p call.Progress) ([]apps.Outcome, error) {
	// invariant: a move installs the app by the method asked before it
	// uninstalls the copy another method holds, and uninstalls it only
	// once that install is in: a failed install leaves the old copy as it
	// was. A copy no method of the catalog holds (on PATH by hand) is the
	// owner's, kept; one its method cannot uninstall is kept and said.
	found, err := s.Find([]string{name})
	if err != nil {
		return nil, err
	}
	t := found[0]
	if _, err = s.by(found, to, ""); err != nil {
		var none *apps.NoMethodError
		if errors.As(err, &none) {
			if was := s.reports(ctx, found)[0]; was.Managed {
				none.Held = was.Method
			}
		}
		return nil, err
	}
	was := s.reports(ctx, found)[0]
	if !settled(was.State) {
		return nil, &apps.NotInstalledError{App: name}
	}
	if was.Managed && was.Method == to {
		return s.held(ctx, []apps.Outcome{{Name: name, Verb: verbMove, Result: catalog.StateOK, Detail: "already installed by " + string(to)}}, map[string]catalog.App{name: s.holding(t, was)}), nil
	}
	out, err := s.Install(ctx, apps.InstallRequest{Names: []string{name}, Method: to}, p)
	if err != nil {
		return out, err
	}
	for i := range out {
		if out[i].Name != name {
			continue
		}
		out[i].Verb = verbMove
		if out[i].Result != catalog.StateOK {
			return out, nil
		}
		out[i].Detail = "moved from " + string(was.Method)
		if !was.Managed {
			out[i].Detail = "installed by " + string(to)
		}
		out[i].Steps = append(out[i].Steps, s.leave(ctx, t, was))
	}
	return out, nil
}

func (s *Shelf) leave(ctx context.Context, t catalog.App, was apps.Report) apps.Step {
	old := s.holding(t, was)
	step := apps.Step{Item: string(was.Method) + " copy", Result: catalog.StateOK}
	switch {
	case !was.Managed:
		step.Item, step.Detail = "copy on PATH", "kept: it is yours, installed by none of its methods"
	case s.uninstallable(old) != nil:
		step.Detail = "kept: station cannot uninstall it"
	case s.unrecorded(ctx, old):
		step.Detail = "kept: yours, station added none of its packages; `station apps remove " + t.Name + "` takes it on your word"
	default:
		o := s.undone(ctx, old, s.run(ctx, verbUninstall, old, false, contract.Installer.Remove), false)
		step.Result, step.Detail, step.Why = o.Result, "uninstalled", o.Why
		if o.Result != catalog.StateOK {
			step.Detail = o.Detail
		}
	}
	return step
}

// invariant: a script's or a shell line's undo is done only once the app
// reads gone: its check fails in the folder the install lands in, and, for
// a removal (onPath; a move's new copy answers there by design), none of
// its commands answers on the owner's PATH. Otherwise the outcome is no
// success, and says what still answers, and where.
func (s *Shelf) undone(ctx context.Context, t catalog.App, o apps.Outcome, onPath bool) apps.Outcome {
	in := s.install(t)
	if o.Result != catalog.StateOK || !byLine(in) {
		return o
	}
	if check := catalog.CheckOf(in); check != "" && s.readChecked(ctx, t, apps.Report{}, check).State == catalog.StateOK {
		o.Result, o.Detail = catalog.StateBroken, "undone, but its check still passes in "+catalog.ScriptBin
		return o
	}
	for _, name := range t.Provides {
		if where, ok := s.m.LookPath(name); ok && onPath {
			o.Result, o.Detail = catalog.StateBroken, "undone, but "+name+" still answers at "+s.shown(where)
			return o
		}
	}
	return o
}

// value: the install is a line station runs (a script, a shell line),
// taken back by its own `undo`.
func byLine(in catalog.Install) bool {
	switch in.(type) {
	case catalog.ScriptInstall, catalog.ShellInstall:
		return true
	}
	return false
}

// value: an app's installer as its verbs meet it: a Holder is told the app
// it serves, and whether the owner gave their word for it.
func (s *Shelf) serving(t catalog.App, word bool) contract.Installer {
	in := s.installer(t)
	if h, ok := in.(contract.Holder); ok {
		return serving{Installer: in, holder: h, f: contract.For{App: t.Name, Word: word}}
	}
	return in
}

type serving struct {
	contract.Installer
	holder contract.Holder
	f      contract.For
}

func (s serving) Install(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return s.holder.InstallFor(ctx, s.f, installs, logs)
}

func (s serving) Remove(ctx context.Context, installs []catalog.Install, logs contract.Logs) error {
	return s.holder.RemoveFor(ctx, s.f, installs, logs)
}

// value: the app's method records what its installs add, and recorded
// adding nothing for it; a record that does not read is no answer, and
// leaves the removal to refuse it.
func (s *Shelf) unrecorded(ctx context.Context, t catalog.App) bool {
	h, ok := s.installer(t).(contract.Holder)
	if !ok {
		return false
	}
	holds, err := h.Holds(ctx, t.Name)
	return err == nil && !holds
}
