package machinefile

import (
	"slices"
	"strconv"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

// value: which of three groups a Machinefile is shown in: the owner's own
// and a shared source's files that make a machine, the mixins whatever
// their source, and the files station ships.

type Shelf string

const (
	ShelfYours   Shelf = "yours"
	ShelfMixin   Shelf = "mixin"
	ShelfShipped Shelf = "shipped"
)

func (Shelf) Enum() []string {
	return []string{string(ShelfYours), string(ShelfMixin), string(ShelfShipped)}
}

// value: one Machinefile as the window shows it: its name, its group, the
// layer and the path it is read at, its description, the files it
// extends, what it makes (nil for a mixin), its text as its source holds
// it (a secret inside it withheld), the apps and settings it lists itself,
// this machine's overlay of it, and the machines that follow it.

type FileView struct {
	Name        string
	Shelf       Shelf
	Layer       Layer
	Path        string
	Description string
	Extends     []string
	Head        *Head `json:",omitempty"`
	Text        string
	Apps        []string
	Settings    []string
	Local       *Local `json:",omitempty"`
	Followers   []Follower
}

// value: what a Machinefile makes, read over its whole chain: the kind
// of machine, its system, and for a VM how it runs.

type Head struct {
	Kind   catalog.MachineRole
	System catalog.SystemSpec
	VM     *HeadVM `json:",omitempty"`
}

type HeadVM struct {
	Lifetime     catalog.Lifetime
	CPUs         int
	Memory, Disk string
	From         string
}

// value: this machine's overlay of a Machinefile, the one file a surface
// may write: its path, its text, the apps it adds and removes, each value
// it sets (a secret withheld whole), the settings it removes, and whether
// it replaces the file's accounts and ports.

type Local struct {
	Path            string
	Text            string
	AppsAdded       []string
	AppsRemoved     []string
	SettingsSet     []SettingSet
	SettingsRemoved []string
	Accounts, Ports bool
}

type SettingSet struct {
	Setting, Option, Value string
}

// value: a machine that follows a Machinefile: a Mac bound to it, a VM
// made from it, by its name and the Mac it runs on.

type Follower struct {
	Machine string
	Mac     string
	ThisMac bool
}

func FilesOf(lib Library) []FileView {
	// invariant: every Machinefile the library holds, in name order, each
	// read alone: a file whose chain does not resolve still shows its own
	// keys, with no head.
	out := make([]FileView, 0, len(lib.Machinefiles))
	for _, name := range sortedNames(lib.Machinefiles) {
		p := lib.Machinefiles[name]
		v := FileView{
			Name: name, Layer: p.Layer, Path: p.Path, Description: p.File.Description, Extends: p.File.Extends,
			Text: secret.Kept(string(p.Body)), Apps: listedApps(p.File), Settings: listedSettings(p.File),
		}
		if files, err := lib.chain(name); err == nil {
			v.Head = headOf(merge(files))
		}
		switch {
		case v.Head == nil:
			v.Shelf = ShelfMixin
		case p.Layer == Builtin:
			v.Shelf = ShelfShipped
		default:
			v.Shelf = ShelfYours
		}
		if o, ok := lib.Overlays[name]; ok {
			v.Local = localOf(o)
		}
		out = append(out, v)
	}
	return out
}

func headOf(m merged) *Head {
	if !m.machine {
		return nil
	}
	at := func(key string) string { return m.values[key].Text }
	h := &Head{Kind: catalog.MachineRole(at("machine.kind")), System: catalog.SystemSpec{OS: catalog.OS(at("machine.system.os")), Version: at("machine.system.version"), Arch: catalog.Arch(at("machine.system.arch"))}}
	if h.Kind != catalog.RoleComputer {
		vm := &HeadVM{Lifetime: catalog.Lifetime(at("vm.lifetime")), Memory: at("vm.resources.memory"), Disk: at("vm.resources.disk"), From: at("vm.image.from")}
		vm.CPUs, _ = strconv.Atoi(at("vm.resources.cpus"))
		h.VM = vm
	}
	return h
}

func listedApps(f catalog.Machinefile) []string {
	out := []string{}
	if f.Apps != nil {
		for _, a := range f.Apps.List {
			out = append(out, a.Name)
		}
	}
	return out
}

func listedSettings(f catalog.Machinefile) []string {
	out := []string{}
	if f.Settings != nil && !f.Settings.Changes {
		out = append(out, sortedNames(f.Settings.Set)...)
	}
	return out
}

func localOf(o Placed) *Local {
	l := &Local{Path: o.Path, Text: secret.Kept(string(o.Body)), AppsAdded: []string{}, AppsRemoved: []string{}, SettingsSet: []SettingSet{}, SettingsRemoved: []string{}, Accounts: o.File.Accounts != nil, Ports: o.File.Ports != nil}
	if a := o.File.Apps; a != nil {
		for _, e := range a.Add {
			l.AppsAdded = append(l.AppsAdded, e.Name)
		}
		l.AppsRemoved = append(l.AppsRemoved, a.Remove...)
	}
	if s := o.File.Settings; s != nil {
		for _, name := range sortedNames(s.Set) {
			values := s.Set[name].Values
			if len(values) == 0 {
				l.SettingsSet = append(l.SettingsSet, SettingSet{Setting: name})
			}
			for _, k := range sortedNames(values) {
				text, _ := withhold(string(values[k]))
				l.SettingsSet = append(l.SettingsSet, SettingSet{Setting: name, Option: k, Value: text})
			}
		}
		l.SettingsRemoved = append(l.SettingsRemoved, slices.Sorted(slices.Values(s.Remove))...)
	}
	return l
}
