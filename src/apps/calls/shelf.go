package calls

import (
	"context"
	"errors"
	"slices"
	"sort"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
)

const commandLane = 16

type ShelfOptions struct {
	Pool        *lanes.Pool
	ThisMachine catalog.MachineKind
	Root        string
	Installers  contract.Installers
	Bundler     contract.Bundler
	VMs         VMs
	// value: the folder of each app's log, `<app>.log`; empty keeps none.
	Logs string
}

type Configurer interface {
	Configure(ctx context.Context, item string, interactive bool) (apps.Step, error)
}

type VMs interface {
	Running(ctx context.Context) (map[string]catalog.MachineKind, error)
}

type Shelf struct {
	m      Machine
	cat    catalog.Catalog
	apps   []catalog.App
	byName map[string]catalog.App
	cfg    Configurer
	o      ShelfOptions
}

func NewShelf(m Machine, cat catalog.Catalog, cfg Configurer, o ShelfOptions) *Shelf {
	byName := make(map[string]catalog.App, len(cat.Apps))
	for _, t := range cat.Apps {
		byName[t.Name] = t
	}
	if o.Pool == nil {
		o.Pool = lanes.NewPool(context.Background(), m, commandLane, 1)
	}
	return &Shelf{m: m, cat: cat, apps: cat.Apps, byName: byName, cfg: cfg, o: o}
}

func (s *Shelf) Placed() []catalog.App {
	var placed []catalog.App
	for _, t := range s.apps {
		if _, ok := t.InstallOn(s.o.ThisMachine); ok {
			placed = append(placed, t)
		}
	}
	return placed
}

func (s *Shelf) Find(names []string) ([]catalog.App, error) {
	seen := map[string]bool{}
	var found []catalog.App
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		t, ok := s.byName[name]
		if !ok {
			return nil, &apps.UnknownError{Name: name}
		}
		if _, ok := t.InstallOn(s.o.ThisMachine); !ok {
			return nil, &apps.ElsewhereError{App: name, Kinds: t.LivesOn()}
		}
		found = append(found, t)
	}
	return found, nil
}

func (s *Shelf) CheckInstall(ctx context.Context, names []string) error {
	found, err := s.Find(names)
	var livesOn *apps.ElsewhereError
	if errors.As(err, &livesOn) {
		livesOn.VM = s.runningVM(ctx, livesOn.Kinds)
	}
	if err != nil {
		return err
	}
	for _, t := range found {
		if !s.installer(t).Can().Install {
			return &apps.ManualInstallError{App: t.Name, Hint: s.install(t).Key()}
		}
	}
	return nil
}

func (s *Shelf) CheckUninstall(names []string) error {
	found, err := s.Find(names)
	if err != nil {
		return err
	}
	for _, t := range found {
		if err := s.uninstallable(t); err != nil {
			return err
		}
	}
	return nil
}

func (s *Shelf) uninstallable(t catalog.App) error {
	if !s.installer(t).Can().Remove {
		return &apps.NotReversibleError{App: t.Name}
	}
	if s.fromAppStore(t) {
		return &apps.AppStoreError{App: t.Name}
	}
	return nil
}

func (s *Shelf) CheckUpgrade(names []string) error {
	found, err := s.Find(names)
	if err != nil {
		return err
	}
	for _, t := range found {
		if !s.installer(t).Can().Update {
			return &apps.NoUpgradePathError{App: t.Name}
		}
	}
	return nil
}

func (s *Shelf) install(t catalog.App) catalog.Install {
	i, _ := t.InstallOn(s.o.ThisMachine)
	return i
}

// invariant: the registry wire builds holds an installer for every method
// the catalogue places.
func (s *Shelf) installer(t catalog.App) contract.Installer {
	return s.o.Installers[s.install(t).Method()]
}

func (s *Shelf) lined(t catalog.App) (contract.Lined, bool) {
	l, ok := s.installer(t).(contract.Lined)
	return l, ok
}

// invariant: the VM named in a refusal is one that runs now and whose kind is a
// place of the tool; a failed listing names none, so the refusal still stands.
func (s *Shelf) runningVM(ctx context.Context, kinds []catalog.MachineKind) string {
	if s.o.VMs == nil {
		return ""
	}
	running, err := s.o.VMs.Running(ctx)
	if err != nil {
		return ""
	}
	names := make([]string, 0, len(running))
	for name, kind := range running {
		if slices.Contains(kinds, kind) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[0]
}
