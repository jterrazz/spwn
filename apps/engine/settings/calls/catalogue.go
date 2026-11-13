package calls

import (
	"context"
	"errors"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/settings"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	schedule "github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
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
}

type Catalogue struct {
	items    []catalog.Setting
	byName   map[string]catalog.Setting
	o        CatalogueOptions
	appliers settingscontract.Appliers
}

func NewCatalogue(items []catalog.Setting, o CatalogueOptions) *Catalogue {
	byName := make(map[string]catalog.Setting, len(items))
	for _, it := range items {
		byName[it.Name] = it
	}
	return &Catalogue{items: items, byName: byName, o: o, appliers: settingscontract.NewAppliers(o.Appliers...)}
}

type handler struct {
	check     func(ctx context.Context) (catalog.State, string)
	install   func(ctx context.Context, values map[string]string) (settings.Change, error)
	uninstall func(ctx context.Context) (settings.Change, error)
}

func (s *Catalogue) Placed() []catalog.Setting {
	var placed []catalog.Setting
	for _, it := range s.items {
		if slices.Contains(it.Kinds, s.o.ThisMachine) {
			placed = append(placed, it)
		}
	}
	return placed
}

func (s *Catalogue) Items() []catalog.Setting { return s.items }

func (s *Catalogue) Lives(it catalog.Setting) bool { return slices.Contains(it.Kinds, s.o.ThisMachine) }

func (s *Catalogue) Needs(it catalog.Setting) string {
	if s.applies(it) {
		return ""
	}
	return it.Requires
}

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
	// invariant: the first value is every placed item, those with a check
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
		reports[i] = settings.Report{Item: it, State: catalog.StateAbsent, Detail: "run once", Applies: s.applies(it)}
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
			return nil, &settings.UnknownItemError{Name: name}
		}
		if !slices.Contains(it.Kinds, s.o.ThisMachine) {
			return nil, &settings.WrongLivesOnError{Item: name, Kinds: it.Kinds, ThisMachine: s.o.ThisMachine}
		}
		found = append(found, it)
	}
	return found, nil
}

func (s *Catalogue) NeedsInput(it catalog.Setting) bool {
	return len(it.Inputs) > 0 || it.Interactive
}

func (s *Catalogue) Defaults(item string) map[string]string {
	return clone(s.o.Defaults[item])
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
	case !slices.Contains(it.Kinds, s.o.ThisMachine):
		return &settings.WrongLivesOnError{Item: it.Name, Kinds: it.Kinds, ThisMachine: s.o.ThisMachine}
	case verb == verbUninstall && !it.Reversible:
		return &settings.NotReversibleError{Item: it.Name}
	case verb == verbInstall && s.NeedsInput(it) && !interactive:
		return &settings.NeedsTerminalError{Item: it.Name}
	case it.NeedsRoot && !s.o.Admin:
		return &settings.NeedsRootError{Item: it.Name, Verb: verb}
	}
	return nil
}

func (s *Catalogue) Install(ctx context.Context, name string, values map[string]string, interactive bool) (settings.Change, error) {
	return s.apply(name, verbInstall, interactive, func(h handler, it catalog.Setting) (settings.Change, error) {
		merged := clone(s.o.Defaults[it.Name])
		for k, v := range values {
			merged[k] = v
		}
		for _, in := range it.Inputs {
			if in.Required && merged[in.Name] == "" {
				return settings.Change{}, errors.New(it.Name + " needs " + in.Name)
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
	failed := settings.Change{Item: name, Verb: verb, Result: catalog.StateBroken}
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
	change.Item, change.Verb = it.Name, verb
	if err != nil {
		change.Result = catalog.StateBroken
		change.Steps = append(change.Steps, settings.ChangeStep{Label: "failed", State: catalog.StateBroken, Detail: firstLine(err.Error())})
		return change, err
	}
	change.Result = catalog.StateOK
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
