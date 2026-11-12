package fake

import (
	"context"
	"errors"
	"maps"
	"slices"
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

type HoldingInstaller struct {
	// invariant: an installer of a method that holds what it adds by app,
	// as apt does: a key the install adds is the app's, one the Mac held
	// before is the owner's, and a removal takes back a key with the last
	// app that holds it, one it never recorded only on the owner's word.
	*Installer
}

func InstallerOf(w *world.World, mac string, m catalog.Method, lined bool) contract.Installer {
	i := &Installer{w: w, mac: mac, method: m}
	switch {
	case lined:
		return LinedInstaller{i}
	case m == catalog.MethodApt:
		return HoldingInstaller{i}
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
	// invariant: a manager is the one the real adapter names
	// (World.Managers), else its method's name and an app of that name.
	if named, ok := f.w.Managers[f.method]; ok {
		return contract.Manager{Name: named.Name, App: named.App}
	}
	if !f.method.KeepsList() {
		return contract.Manager{}
	}
	return contract.Manager{Name: string(f.method), App: string(f.method)}
}

func Named(adapter contract.Installer) world.Named {
	// invariant: a world names a method as its real adapter does: its manager
	// and what it can do.
	m, can := adapter.Manager(), adapter.Can()
	return world.Named{Name: m.Name, App: m.App, Install: can.Install, Update: can.Update, Remove: can.Remove, UpdateAll: can.UpdateAll}
}

func (f *Installer) Can() contract.Abilities {
	// invariant: what the real adapter of its method can do: a script's
	// install updates but is never removed, a shell line or a command
	// another item provides only installs, an app bundle is the owner's; a
	// world that names its managers gives their abilities too.
	if named, ok := f.w.Managers[f.method]; ok {
		return contract.Abilities{Install: named.Install, Update: named.Update, Remove: named.Remove, UpdateAll: named.UpdateAll}
	}
	switch f.method {
	case catalog.MethodScript:
		return contract.Abilities{Install: true, Update: true, UpdateAll: true}
	case catalog.MethodShell, catalog.MethodProvidedBy:
		return contract.Abilities{Install: true}
	case catalog.MethodApp:
		return contract.Abilities{}
	default:
		return contract.Abilities{Install: true, Update: true, Remove: true, UpdateAll: true}
	}
}

func (f *Installer) Ready(context.Context) error {
	if f.Manager().Name == "" {
		return nil
	}
	f.w.Lock()
	defer f.w.Unlock()
	if f.manager().Absent {
		return contract.ErrMissing
	}
	return nil
}

func (f *Installer) Inventory(ctx context.Context) (map[string]string, error) {
	if !f.method.KeepsList() {
		return nil, contract.ErrUnlisted
	}
	if err := f.Ready(ctx); err != nil {
		return nil, err
	}
	f.w.Lock()
	defer f.w.Unlock()
	return maps.Clone(f.manager().Installed), nil
}

func (f *Installer) Outdated(_ context.Context, keys []string) (map[string]apps.Currency, error) {
	if !f.method.ReadsCurrency() {
		return nil, nil
	}
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
	keys := make([]string, 0, len(installs))
	for _, i := range installs {
		keys = append(keys, i.Key())
	}
	return f.runKeys(ctx, verb, keys, apply)
}

func (f *Installer) runKeys(ctx context.Context, verb string, keys []string, apply func(s *world.Manager, key string)) error {
	if !f.allows(verb) {
		return contract.ErrCannot
	}
	if err := f.Ready(ctx); err != nil {
		return err
	}
	f.w.Lock()
	if f.w.Fails("Installer." + verb) {
		f.w.Unlock()
		return errScripted
	}
	held := f.w.Holds("Installer." + verb)
	for n, k := range keys {
		if held && n > 0 {
			break
		}
		apply(f.manager(), k)
		if verb == "Install" {
			land(f.w, f.mac, f.manager(), k)
		}
	}
	f.w.Unlock()
	return wait(ctx, held)
}

func (f *Installer) allows(verb string) bool {
	can := f.Can()
	switch verb {
	case "Install":
		return can.Install
	case "Update":
		return can.Update
	}
	return can.Remove
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
	if err := f.unopened(installs); err != nil {
		return err
	}
	if err := f.run(ctx, "Install", installs, installed); err != nil {
		return err
	}
	f.pin(installs)
	return nil
}

// invariant: an install that carries a version holds that version, as
// the real adapter's global use of it leaves it; a channel (latest, lts,
// stable) holds whatever the world offers.
func (f *Installer) pin(installs []catalog.Install) {
	f.w.Lock()
	defer f.w.Unlock()
	for _, i := range installs {
		if v := catalog.VersionOf(i); v != "" && strings.ContainsAny(v[:1], "0123456789") {
			f.manager().Installed[i.Key()] = v
		}
	}
}

// invariant: a command another item brings at its first launch is absent
// while that item was never opened on this Mac, as on a fresh one.
func (f *Installer) unopened(installs []catalog.Install) error {
	f.w.Lock()
	defer f.w.Unlock()
	for _, i := range installs {
		if by, ok := i.(catalog.ProvidedByInstall); ok && slices.Contains(f.w.Mac(f.mac).Unopened, by.Item) && len(by.Commands) > 0 {
			return &apps.NotYetProvidedError{Item: by.Item, Command: by.Commands[0]}
		}
	}
	return nil
}

func (f *Installer) Update(ctx context.Context, installs []catalog.Install, _ contract.Logs) error {
	return f.run(ctx, "Update", installs, updated)
}

func (f *Installer) Remove(ctx context.Context, installs []catalog.Install, _ contract.Logs) error {
	return f.run(ctx, "Remove", installs, func(s *world.Manager, k string) { delete(s.Installed, k) })
}

var errHolds = errors.New("this method installs and removes for an app: through InstallFor and RemoveFor")

func (HoldingInstaller) Install(context.Context, []catalog.Install, contract.Logs) error {
	return errHolds
}

func (HoldingInstaller) Remove(context.Context, []catalog.Install, contract.Logs) error {
	return errHolds
}

func (h HoldingInstaller) InstallFor(ctx context.Context, f contract.For, installs []catalog.Install, logs contract.Logs) error {
	h.w.Lock()
	s := h.manager()
	var added []string
	for _, i := range installs {
		k := i.Key()
		_, had := s.Installed[k]
		held, recorded := s.Holders[k]
		if had && !recorded {
			continue
		}
		if s.Holders == nil {
			s.Holders = map[string][]string{}
		}
		if !slices.Contains(held, f.App) {
			s.Holders[k] = append(slices.Clone(held), f.App)
		}
		added = append(added, k)
	}
	h.w.Unlock()
	err := h.Installer.Install(ctx, installs, logs)
	h.w.Lock()
	defer h.w.Unlock()
	for _, k := range added {
		if _, there := s.Installed[k]; !there {
			letGo(s, k, f.App)
		}
	}
	return err
}

func (h HoldingInstaller) RemoveFor(ctx context.Context, f contract.For, installs []catalog.Install, _ contract.Logs) error {
	h.w.Lock()
	s := h.manager()
	var take []string
	ours := false
	for _, k := range slices.Sorted(maps.Keys(s.Holders)) {
		if !slices.Contains(s.Holders[k], f.App) {
			continue
		}
		ours = true
		if _, there := s.Installed[k]; there && len(s.Holders[k]) == 1 {
			take = append(take, k)
		}
	}
	if !ours && !f.Word {
		h.w.Unlock()
		return &apps.UnrecordedError{App: f.App}
	}
	if !ours {
		for _, i := range installs {
			if _, recorded := s.Holders[i.Key()]; !recorded {
				take = append(take, i.Key())
			}
		}
	}
	h.w.Unlock()
	if err := h.runKeys(ctx, "Remove", take, func(s *world.Manager, k string) { delete(s.Installed, k) }); err != nil {
		return err
	}
	h.w.Lock()
	defer h.w.Unlock()
	for k := range s.Holders {
		letGo(s, k, f.App)
	}
	return nil
}

func (h HoldingInstaller) Holds(_ context.Context, app string) (bool, error) {
	h.w.Lock()
	defer h.w.Unlock()
	for _, held := range h.manager().Holders {
		if slices.Contains(held, app) {
			return true, nil
		}
	}
	return false, nil
}

func letGo(s *world.Manager, key, app string) {
	held := slices.DeleteFunc(slices.Clone(s.Holders[key]), func(a string) bool { return a == app })
	if len(held) == 0 {
		delete(s.Holders, key)
		return
	}
	s.Holders[key] = held
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
	_ contract.Holder  = HoldingInstaller{}
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
	if named, ok := w.Managers[m]; ok {
		return named.App
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
