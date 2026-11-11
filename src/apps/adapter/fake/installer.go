package fake

import (
	"context"
	"errors"
	"maps"
	"strings"

	"github.com/jterrazz/studio-station/src/apps"
	"github.com/jterrazz/studio-station/src/apps/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/world"
)

// value: worded as a tool's own stderr, so a test sees whether station
// shows it raw.
var errScripted = errors.New("exited 1: Error: the run failed, as the world scripted it")

const thisMac = "this-mac"

type State struct {
	Absent    bool
	Installed map[string]string
	Behind    map[string]string
	Fails     bool
}

type Installer struct {
	// invariant: one method's installer on one Mac of the world: what it
	// holds is that Mac's slice, and a fault scripted for it fails its run.
	w      *world.World
	mac    string
	method catalog.Method
}

type LinedInstaller struct {
	// invariant: an installer of a method whose installs ride the one
	// bundle, as D7's brew bundle carries brew, cask, mas, npm and uv.
	*Installer
}

func InstallerOf(w *world.World, mac string, m catalog.Method, lined bool) contract.Installer {
	i := &Installer{w: w, mac: mac, method: m}
	if lined {
		return LinedInstaller{i}
	}
	return i
}

func New(m catalog.Method, s *State) *Installer {
	w := &world.World{ThisMac: thisMac}
	w.Mac(thisMac).Apps = map[catalog.Method]*world.Manager{m: {Absent: s.Absent, Installed: maps.Clone(s.Installed), Behind: maps.Clone(s.Behind)}}
	if s.Fails {
		w.Script = []world.Fault{{Call: "Installer.Install"}, {Call: "Installer.Update"}, {Call: "Installer.Remove"}}
	}
	return &Installer{w: w, mac: thisMac, method: m}
}

func (f *Installer) manager() *world.Manager { return manager(f.w, f.mac, f.method) }

func manager(w *world.World, mac string, m catalog.Method) *world.Manager {
	slice := w.Mac(mac)
	if slice.Apps == nil {
		slice.Apps = map[catalog.Method]*world.Manager{}
	}
	s := slice.Apps[m]
	if s == nil {
		s = &world.Manager{}
		slice.Apps[m] = s
	}
	if s.Installed == nil {
		s.Installed = map[string]string{}
	}
	return s
}

func (f *Installer) Method() catalog.Method { return f.method }

func (f *Installer) Manager() contract.Manager {
	// invariant: a manager names the catalogue app that brings it, the one
	// the real adapter names (World.Managers), else its method's name.
	return contract.Manager{Name: string(f.method), App: managerApp(f.w, f.method)}
}

func (f *Installer) Can() contract.Abilities {
	return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
}

func (f *Installer) Ready(context.Context) error {
	f.w.Lock()
	defer f.w.Unlock()
	if f.manager().Absent {
		return contract.ErrMissing
	}
	return nil
}

func (f *Installer) Inventory(ctx context.Context) (map[string]string, error) {
	if err := f.Ready(ctx); err != nil {
		return nil, err
	}
	f.w.Lock()
	defer f.w.Unlock()
	return maps.Clone(f.manager().Installed), nil
}

func (f *Installer) Outdated(_ context.Context, keys []string) (map[string]apps.Currency, error) {
	f.w.Lock()
	defer f.w.Unlock()
	out := map[string]apps.Currency{}
	for _, k := range keys {
		out[k] = apps.Currency{Verdict: apps.Current}
		if newest, ok := f.manager().Behind[k]; ok {
			out[k] = apps.Currency{Verdict: apps.Behind, Newest: newest}
		}
	}
	return out, nil
}

func (f *Installer) run(ctx context.Context, verb string, installs []catalog.Install, apply func(s *world.Manager, key string)) error {
	if err := f.Ready(ctx); err != nil {
		return err
	}
	f.w.Lock()
	if f.w.Fails("Installer." + verb) {
		f.w.Unlock()
		return errScripted
	}
	held := f.w.Holds("Installer." + verb)
	for n, i := range installs {
		if held && n > 0 {
			break
		}
		apply(f.manager(), i.Key())
		if verb == "Install" {
			land(f.w, f.mac, f.manager(), i.Key())
		}
	}
	f.w.Unlock()
	return wait(ctx, held)
}

func wait(ctx context.Context, held bool) error {
	if !held {
		return nil
	}
	<-ctx.Done()
	return ctx.Err()
}

func installed(s *world.Manager, k string) {
	s.Installed[k] = "1.0.0"
	if v, ok := s.Offers[k]; ok {
		s.Installed[k] = v
	}
}

func updated(s *world.Manager, k string) {
	if v, ok := s.Behind[k]; ok {
		s.Installed[k] = v
		delete(s.Behind, k)
	}
}

func (f *Installer) Install(ctx context.Context, installs []catalog.Install, _ contract.Logs) error {
	return f.run(ctx, "Install", installs, installed)
}

func (f *Installer) Update(ctx context.Context, installs []catalog.Install, _ contract.Logs) error {
	return f.run(ctx, "Update", installs, updated)
}

func (f *Installer) Remove(ctx context.Context, installs []catalog.Install, _ contract.Logs) error {
	return f.run(ctx, "Remove", installs, func(s *world.Manager, k string) { delete(s.Installed, k) })
}

func (l LinedInstaller) Line(i catalog.Install) string { return string(l.method) + " " + i.Key() }

type Bundler struct {
	// invariant: the one bundle of a Mac of the world: each line installs, or
	// updates, its key under its method; a fault scripted for it fails the
	// whole run and changes nothing.
	w   *world.World
	mac string
}

var (
	_ contract.Bundler = (*Bundler)(nil)
	_ contract.Lined   = LinedInstaller{}
)

func BundlerOf(w *world.World, mac string) *Bundler { return &Bundler{w: w, mac: mac} }

func (b *Bundler) Name() string { return "the bundle" }

func (b *Bundler) Bundle(ctx context.Context, lines []string, update bool, _ contract.Logs) error {
	b.w.Lock()
	if b.w.Fails("Bundler.Bundle") {
		b.w.Unlock()
		return errScripted
	}
	held := b.w.Holds("Bundler.Bundle")
	err := b.apply(lines, update, held)
	b.w.Unlock()
	if err != nil {
		return err
	}
	return wait(ctx, held)
}

func (b *Bundler) apply(lines []string, update, first bool) error {
	apply := installed
	if update {
		apply = updated
	}
	for n, line := range lines {
		if first && n > 0 {
			break
		}
		method, key, ok := strings.Cut(line, " ")
		if !ok {
			return errors.New("the bundle cannot read the line " + line)
		}
		s := manager(b.w, b.mac, catalog.Method(method))
		if s.Absent {
			return contract.ErrMissing
		}
		apply(s, key)
		if !update {
			land(b.w, b.mac, s, key)
		}
	}
	return nil
}

// invariant: an install that brings a manager makes it present, as the real
// command then answers: the methods its key lands, and the method whose
// manager is the app of that name (mas installed by the bundle).
func managerApp(w *world.World, m catalog.Method) string {
	if app := w.Managers[m]; app != "" {
		return app
	}
	return string(m)
}

func land(w *world.World, mac string, by *world.Manager, key string) {
	for _, m := range by.Lands[key] {
		manager(w, mac, m).Absent = false
	}
	for m, s := range w.Mac(mac).Apps {
		if s.Absent && managerApp(w, m) == key {
			s.Absent = false
		}
	}
}
