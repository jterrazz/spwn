package calls

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/settings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	schedule "github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
)

const (
	verbInstall   = "install"
	verbUninstall = "uninstall"
)

type CatalogueOptions struct {
	ThisMachine catalog.MachineKind
	Defaults    map[string]map[string]string
	// value: the Applier of every effect of the catalogue.
	Appliers []settingscontract.Applier
	// value: this run has admin rights, which a root setting needs.
	Admin bool
	// value: the app a setting requires is installed here.
	Has func(app string) bool
	// value: the app whose declared disk paths hold a file, other than the
	// apps named; "" when none does.
	HeldBy func(file string, but []string) string
	// value: this Mac reports a battery, so it is a laptop, and no setting
	// of the Always-on Mac group lives on it.
	Laptop bool
}

type Catalogue struct {
	packages []catalog.Setting
	byName   map[string]catalog.Setting
	o        CatalogueOptions
	appliers settingscontract.Appliers
}

func NewCatalogue(pkgs []catalog.Setting, o CatalogueOptions) *Catalogue {
	byName := make(map[string]catalog.Setting, len(pkgs))
	for _, it := range pkgs {
		byName[it.Name] = it
	}
	return &Catalogue{packages: pkgs, byName: byName, o: o, appliers: settingscontract.NewAppliers(o.Appliers...)}
}

type handler struct {
	check     func(ctx context.Context) (catalog.State, string)
	install   func(ctx context.Context, values map[string]string) (settings.Change, error)
	uninstall func(ctx context.Context) (settings.Change, error)
}

func (s *Catalogue) Placed() []catalog.Setting {
	var placed []catalog.Setting
	for _, it := range s.packages {
		if s.Lives(it) {
			placed = append(placed, it)
		}
	}
	return placed
}

func (s *Catalogue) Packages() []catalog.Setting { return s.packages }

func (s *Catalogue) Lives(it catalog.Setting) bool {
	// invariant: a setting lives on a machine of a kind it names, and the
	// Always-on Mac group never on a laptop: a Mac that moves is not one a
	// server is hosted on.
	return slices.Contains(it.Kinds, s.o.ThisMachine) && (!s.o.Laptop || settings.GroupOf(it) != settings.AlwaysOnMac)
}

func (s *Catalogue) Needs(it catalog.Setting) string {
	if s.applies(it) {
		app, _ := s.held(it)
		return app
	}
	return it.Requires
}

// invariant: a file a setting writes that another app's disk declares (a
// hook in Claude Code's settings) is written only where that app is
// installed: never a folder of an app that is not here.
func (s *Catalogue) held(it catalog.Setting) (app, file string) {
	d, ok := it.Effect.(catalog.Dotfiles)
	if !ok || s.o.HeldBy == nil {
		return "", ""
	}
	for _, c := range d.Files {
		if app := s.o.HeldBy(c.To, it.Needs()); app != "" && (s.o.Has == nil || !s.o.Has(app)) {
			return app, c.To
		}
	}
	return "", ""
}

// value: the words of a setting held by an app that is not here.
func notHere(app string) string { return app + " is not here: nothing written" }

func (s *Catalogue) Admit(name string, interactive bool) error {
	found, err := s.Find([]string{name})
	if err != nil {
		return err
	}
	return s.admit(found[0], verbInstall, interactive)
}

func (s *Catalogue) Apply(ctx context.Context, name string, values map[string]string, interactive bool) (settings.Change, error) {
	return s.Install(ctx, name, values, interactive)
}

func (s *Catalogue) List(ctx context.Context) ([]settings.Report, error) {
	reports, err := s.Watch(ctx)
	if err != nil {
		return nil, err
	}
	return schedule.Last(reports), nil
}

func (s *Catalogue) Watch(ctx context.Context) (<-chan []settings.Report, error) {
	// invariant: the first value is every placed package, those with a check
	// pending; every check is a read of its own, so they run at once, each row
	// fills as its check answers, and the list costs its slowest check.
	placed := s.Placed()
	reports := make([]settings.Report, len(placed))
	checks := make([]func(context.Context) (catalog.State, string), len(placed))
	for i, it := range placed {
		h, err := s.handler(it)
		if err != nil {
			return nil, err
		}
		reports[i] = settings.Report{Package: it, State: catalog.StateAbsent, Detail: "run once", Applies: s.applies(it)}
		if h.check != nil {
			reports[i].State, reports[i].Detail, checks[i] = catalog.StatePending, "", h.check
		}
	}
	type answer struct {
		i      int
		state  catalog.State
		detail string
	}
	answers := make(chan answer, len(placed))
	left := 0
	for i, check := range checks {
		if check == nil {
			continue
		}
		left++
		go func() {
			state, detail := check(ctx)
			answers <- answer{i: i, state: state, detail: detail}
		}()
	}
	out := make(chan []settings.Report)
	go func() {
		defer close(out)
		send := func() bool {
			select {
			case out <- slices.Clone(reports):
				return true
			case <-ctx.Done():
				return false
			}
		}
		if !send() {
			return
		}
		for ; left > 0; left-- {
			a := <-answers
			reports[a.i].State, reports[a.i].Detail = a.state, a.detail
			if !send() {
				return
			}
		}
	}()
	return out, nil
}

func (s *Catalogue) Find(names []string) ([]catalog.Setting, error) {
	seen := map[string]bool{}
	var found []catalog.Setting
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		it, ok := s.byName[name]
		if !ok {
			return nil, &settings.UnknownPackageError{Name: name}
		}
		if !s.Lives(it) {
			return nil, &settings.WrongLivesOnError{Package: name, Kinds: it.Kinds, ThisMachine: s.o.ThisMachine, Laptop: s.o.Laptop && settings.GroupOf(it) == settings.AlwaysOnMac}
		}
		found = append(found, it)
	}
	return found, nil
}

func (s *Catalogue) NeedsInput(it catalog.Setting) bool {
	return len(it.Inputs) > 0 || it.Interactive
}

func (s *Catalogue) Defaults(pkg string) map[string]string {
	return clone(s.o.Defaults[pkg])
}

func (s *Catalogue) CheckInstall(names []string, interactive bool) error {
	found, err := s.Find(names)
	if err != nil {
		return err
	}
	for _, it := range found {
		if err := s.admit(it, verbInstall, interactive); err != nil {
			return err
		}
	}
	return nil
}

func (s *Catalogue) CheckUninstall(names []string) error {
	found, err := s.Find(names)
	if err != nil {
		return err
	}
	for _, it := range found {
		if err := s.admit(it, verbUninstall, true); err != nil {
			return err
		}
	}
	return nil
}

// invariant: the place, the reversibility, the terminal and the root are local
// facts, judged before any question and before any effect.
func (s *Catalogue) admit(it catalog.Setting, verb string, interactive bool) error {
	switch {
	case !s.Lives(it):
		return &settings.WrongLivesOnError{Package: it.Name, Kinds: it.Kinds, ThisMachine: s.o.ThisMachine, Laptop: s.o.Laptop && settings.GroupOf(it) == settings.AlwaysOnMac}
	case verb == verbUninstall && !it.Reversible:
		return &settings.NotReversibleError{Package: it.Name}
	case verb == verbInstall && s.NeedsInput(it) && !interactive:
		return &settings.NeedsTerminalError{Package: it.Name}
	case it.NeedsRoot && !s.o.Admin:
		return &settings.NeedsRootError{Package: it.Name, Verb: verb}
	}
	return nil
}

func (s *Catalogue) Install(ctx context.Context, name string, values map[string]string, interactive bool) (settings.Change, error) {
	return s.apply(name, verbInstall, interactive, func(h handler, it catalog.Setting) (settings.Change, error) {
		// invariant: a fragment goes into a file another app reads (a hook
		// into its settings), so it is written only where the app it requires
		// is there: never a hook whose command is absent.
		if d, ok := it.Effect.(catalog.Dotfiles); ok && d.Fragment() && !s.applies(it) {
			return settings.Change{}, &fail.Said{
				Message: it.Name + " is written only where " + it.NeedsSaid() + " is installed, and here it is not: nothing written; station apps install " + it.Requires,
			}
		}
		if app, file := s.held(it); app != "" {
			return settings.Change{Result: catalog.StateAbsent, Steps: []settings.ChangeStep{{Label: file, State: catalog.StateAbsent, Detail: notHere(app)}}}, nil
		}
		merged := clone(s.o.Defaults[it.Name])
		for k, v := range values {
			merged[k] = v
		}
		for _, in := range it.Inputs {
			if in.Required && merged[in.Name] == "" {
				return settings.Change{}, &fail.Said{Message: it.Name + " needs " + in.Name, Hint: "give the input " + in.Name + " of " + it.Name}
			}
		}
		return h.install(ctx, merged)
	})
}

func (s *Catalogue) Uninstall(ctx context.Context, name string) (settings.Change, error) {
	return s.apply(name, verbUninstall, true, func(h handler, _ catalog.Setting) (settings.Change, error) {
		return h.uninstall(ctx)
	})
}

func (s *Catalogue) apply(name, verb string, interactive bool, run func(handler, catalog.Setting) (settings.Change, error)) (settings.Change, error) {
	failed := settings.Change{Package: name, Verb: verb, Result: catalog.StateBroken}
	found, err := s.Find([]string{name})
	if err != nil {
		return failed, err
	}
	it := found[0]
	if refusal := s.admit(it, verb, interactive); refusal != nil {
		return failed, refusal
	}
	h, err := s.handler(it)
	if err != nil {
		return failed, err
	}
	change, err := run(h, it)
	change.Package, change.Verb = it.Name, verb
	if err != nil {
		change.Result = catalog.StateBroken
		change.Steps = append(change.Steps, settings.ChangeStep{Label: "failed", State: catalog.StateBroken, Detail: firstLine(err.Error())})
		return change, err
	}
	if change.Result != catalog.StateAbsent {
		change.Result = catalog.StateOK
	}
	return change, nil
}

func (s *Catalogue) applies(it catalog.Setting) bool {
	needs := it.Needs()
	if len(needs) == 0 {
		return true
	}
	return s.o.Has != nil && slices.ContainsFunc(needs, s.o.Has)
}

func clone(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
