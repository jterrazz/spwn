package calls

import (
	"context"
	"errors"
	"slices"

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

func (s *Shelf) Install(ctx context.Context, r apps.InstallRequest) ([]apps.Outcome, error) {
	requested, err := s.Find(r.Names)
	if err != nil {
		return nil, err
	}
	states := s.statesWithRequirements(ctx, requested)
	requested = s.withProviders(requested, states)
	inRequest := map[string]bool{}
	for _, t := range requested {
		inRequest[t.Name] = true
	}
	for _, t := range requested {
		for _, dep := range s.requirements(t) {
			if !inRequest[dep] && !settled(states[dep]) {
				return nil, &apps.NeedsError{App: t.Name, Needs: dep}
			}
		}
	}
	ordered := s.topological(requested)
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
			batch = s.bundleBatch(pending, inRequest, outcomes)
		} else {
			batch = []catalog.App{pending[0]}
		}
		pending = slices.DeleteFunc(pending, func(t catalog.App) bool { return slices.ContainsFunc(batch, sameName(t)) })
		var ready []catalog.App
		for _, t := range batch {
			if dep := s.failedRequirement(t, outcomes); dep != "" {
				outcomes[t.Name] = &apps.Outcome{Name: t.Name, Verb: verbInstall, Result: catalog.StateBroken, Detail: "requires " + dep + ", which failed"}
				continue
			}
			ready = append(ready, t)
		}
		if len(ready) == 0 {
			continue
		}
		var done []apps.Outcome
		if _, ok := s.lined(ready[0]); ok {
			done = s.bundle(ctx, verbInstall, ready, false)
		} else {
			done = []apps.Outcome{s.landed(ctx, ready[0], s.run(ctx, verbInstall, ready[0], contract.Installer.Install))}
		}
		for i := range done {
			o := done[i]
			if o.Result == catalog.StateOK {
				o.Steps = s.then(ctx, s.byName[o.Name], r.Interactive)
			}
			outcomes[o.Name] = &o
		}
	}
	for _, t := range ordered {
		if o := outcomes[t.Name]; o.Result == catalog.StateOK {
			o.Steps = append(o.Steps, s.supersede(ctx, t)...)
		}
	}
	result := make([]apps.Outcome, 0, len(ordered))
	for _, t := range requested {
		result = append(result, *outcomes[t.Name])
	}
	return s.held(ctx, result), nil
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
	if err := s.o.Installers[old.Method()].Remove(ctx, []catalog.Install{old}, nil); err != nil {
		step.Detail, step.Why = s.failed(verbUninstall, []catalog.App{t}, err)
		step.Result = catalog.StateBroken
	}
	return []apps.Step{step}
}

func (s *Shelf) Uninstall(ctx context.Context, names []string) ([]apps.Outcome, error) {
	requested, err := s.Find(names)
	if err != nil {
		return nil, err
	}
	states := s.states(ctx, requested)
	for _, t := range requested {
		if !settled(states[t.Name]) {
			return nil, &apps.NotInstalledError{App: t.Name}
		}
	}
	out := make([]apps.Outcome, 0, len(requested))
	for _, t := range requested {
		o := s.run(ctx, verbUninstall, t, contract.Installer.Remove)
		o.Method = s.install(t).Method()
		out = append(out, o)
	}
	return out, nil
}

func (s *Shelf) Upgrade(ctx context.Context, r apps.UpgradeRequest) ([]apps.Outcome, error) {
	var candidates []catalog.App
	if len(r.Names) > 0 {
		found, err := s.Find(r.Names)
		if err != nil {
			return nil, err
		}
		candidates = found
	} else {
		via := map[catalog.Method]bool{}
		for _, mm := range catalog.Managers {
			if slices.Contains(r.Via, mm.Manager) {
				for _, m := range mm.Methods {
					via[m] = true
				}
			}
		}
		for _, t := range s.Placed() {
			can := s.installer(t).Can()
			reached := can.UpdateAll
			if len(r.Via) > 0 {
				reached = via[s.install(t).Method()]
			}
			if reached && can.Update {
				candidates = append(candidates, t)
			}
		}
	}
	reports := map[string]apps.Report{}
	for _, rep := range s.reports(ctx, candidates) {
		reports[rep.App.Name] = rep
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
		outcomes = append(outcomes, s.run(ctx, verbUpdate, t, contract.Installer.Update))
	}
	return s.held(ctx, outcomes), nil
}

func (s *Shelf) held(ctx context.Context, outcomes []apps.Outcome) []apps.Outcome {
	// invariant: an outcome names the method that holds its app and the
	// version the method lists once the verb has run, read once per method.
	versions := map[catalog.Method]map[string]string{}
	for i := range outcomes {
		t, ok := s.byName[outcomes[i].Name]
		if !ok {
			continue
		}
		in := s.install(t)
		outcomes[i].Method = in.Method()
		if outcomes[i].Result != catalog.StateOK {
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

func (s *Shelf) bundleBatch(pending []catalog.App, inRequest map[string]bool, done map[string]*apps.Outcome) []catalog.App {
	var batch []catalog.App
	for _, t := range pending {
		if !s.isLined(t) {
			continue
		}
		ready := true
		for _, dep := range s.requirements(t) {
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
	failure, why := "", apps.Why("")
	if err := s.o.Bundler.Bundle(ctx, lines, update, nil); err != nil {
		failure, why = s.failed(verb, list, err)
	}
	out := make([]apps.Outcome, 0, len(list))
	if verb == verbUpdate {
		for _, t := range list {
			o := apps.Outcome{Name: t.Name, Verb: verb, Result: catalog.StateOK}
			if why != "" {
				o.Result, o.Detail, o.Why = catalog.StateBroken, failure, why
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
			o.Result, o.Detail, o.Why = catalog.StateBroken, failure, why
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

type verbOf func(contract.Installer, context.Context, []catalog.Install, call.Progress) error

func (s *Shelf) run(ctx context.Context, verb string, t catalog.App, op verbOf) apps.Outcome {
	o := apps.Outcome{Name: t.Name, Verb: verb, Result: catalog.StateOK}
	err := op(s.installer(t), ctx, []catalog.Install{s.install(t)}, nil)
	var signedOut *apps.SignedOutError
	switch {
	case errors.As(err, &signedOut):
		o.Result, o.Detail = catalog.StateBroken, signedOut.Error()
	case err != nil:
		o.Detail, o.Why = s.failed(verb, []catalog.App{t}, err)
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

// invariant: an app another item provides requires that item, and an install
// of it installs that item first, reported as its own outcome; one already
// there is left alone.
func (s *Shelf) withProviders(list []catalog.App, states map[string]catalog.State) []catalog.App {
	var out []catalog.App
	for _, t := range list {
		if by, ok := s.install(t).(catalog.ProvidedByInstall); ok && !settled(states[by.Item]) && !slices.ContainsFunc(list, sameName(catalog.App{Name: by.Item})) {
			if p, placed := s.byName[by.Item]; placed && !slices.ContainsFunc(out, sameName(p)) {
				if _, lives := p.InstallOn(s.o.ThisMachine); lives {
					out = append(out, p)
				}
			}
		}
		out = append(out, t)
	}
	return out
}

func (s *Shelf) requirements(t catalog.App) []string {
	reqs := slices.Clone(t.Requires)
	if by, ok := s.install(t).(catalog.ProvidedByInstall); ok && !slices.Contains(reqs, by.Item) {
		reqs = append([]string{by.Item}, reqs...)
	}
	if needs := s.installer(t).Manager().Needs; needs != "" && needs != t.Name && !slices.Contains(reqs, needs) {
		reqs = append([]string{needs}, reqs...)
	}
	return reqs
}

func (s *Shelf) failedRequirement(t catalog.App, outcomes map[string]*apps.Outcome) string {
	for _, dep := range s.requirements(t) {
		if o := outcomes[dep]; o != nil && o.Result == catalog.StateBroken {
			return dep
		}
	}
	return ""
}

func (s *Shelf) topological(list []catalog.App) []catalog.App {
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
		for _, dep := range s.requirements(t) {
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
		for _, dep := range s.requirements(t) {
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
	_, listed := listings[own.Method()].keys[own.Key()]
	return listed || s.bundled(t)
}

func settled(s catalog.State) bool { return s == catalog.StateOK || s == catalog.StateDegraded }

func sameName(t catalog.App) func(catalog.App) bool {
	return func(o catalog.App) bool { return o.Name == t.Name }
}
