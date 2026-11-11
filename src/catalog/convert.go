package catalog

import "slices"

// invariant: a method's selector is the kinds of machine it is placed on, in
// the order Mac, Linux, macOS VM: `os: macos` is a Mac and a macOS VM, `only`
// keeps the Mac itself or the VMs.
func (m MethodSpec) kinds() []MachineKind {
	var all []MachineKind
	switch m.OS {
	case OSMacOS:
		all = []MachineKind{Mac, MacOS}
	case OSLinux:
		all = []MachineKind{Linux}
	case OSAny:
		all = []MachineKind{Mac, Linux, MacOS}
	}
	switch m.Only {
	case OnlyMac:
		return slices.DeleteFunc(all, notMac)
	case OnlyVM:
		return slices.DeleteFunc(all, isMac)
	}
	return all
}

func isMac(k MachineKind) bool  { return k == Mac }
func notMac(k MachineKind) bool { return k != Mac }

// invariant: checkMethod let through exactly one app manager key.
func (m MethodSpec) install(provides []string) Install {
	switch {
	case m.Brew != "":
		return BrewInstall{Formula: m.Brew, Trusted: m.Trusted}
	case m.Cask != "":
		return CaskInstall{Token: m.Cask, Trusted: m.Trusted}
	case m.Mas != nil:
		return MasInstall{Name: m.Mas.Name, ID: m.Mas.ID}
	case m.Npm != "":
		return NpmInstall{Package: m.Npm}
	case m.Uv != "":
		return UvInstall{Package: m.Uv}
	case m.Bun != "" && m.Package != "":
		return BunInstall{Package: m.Package, Name: m.Bun}
	case m.Bun != "":
		return BunInstall{Package: m.Bun}
	case m.Mise != nil:
		return MiseInstall{App: m.Mise.Name, Version: m.Mise.Version}
	case m.Script != nil && m.Script.Run != "":
		return ShellInstall{Line: m.Script.Run}
	case m.Script != nil:
		return ScriptInstall{URL: m.Script.URL, Args: m.Script.Args, Env: m.Script.Env, Upgrade: m.Script.Upgrade}
	case m.Skill != nil:
		return SkillInstall{Repo: m.Skill.Repo, Skill: m.Skill.Name}
	case m.ProvidedBy != "":
		return ProvidedByInstall{Item: m.ProvidedBy, Commands: provides}
	}
	return AppInstall{Hint: m.Manual}
}

// invariant: checkMethod let through exactly one setting key.
func (m MethodSpec) effect() Effect {
	switch {
	case m.File != nil && m.File.Touch != "":
		return Touch{Path: m.File.Touch}
	case m.File != nil:
		return Dotfiles{Files: m.File.Copy, Reload: m.File.Reload}
	case m.GitAliases != nil:
		return *m.GitAliases
	case m.ZshFunctions != nil:
		return *m.ZshFunctions
	case m.AgentHooks != nil:
		return *m.AgentHooks
	case m.MuxMachines != nil:
		return MuxMachines{}
	}
	return m.Builtin
}

func (it Item) app(category Category) App {
	t := App{
		Name: it.Name, Kind: it.Kind, Description: it.Description, Category: category, Replaces: it.Replaces,
		Supersedes: it.Supersedes, AppStore: it.AppStore, Provides: it.Provides, Bundle: it.Bundle,
		Requires: it.Requires, Then: it.Then,
	}
	for _, m := range it.Methods {
		t.Installs = append(t.Installs, LivesOn{Kinds: m.kinds(), Install: m.install(it.Provides)})
		for _, p := range m.Path {
			if !slices.Contains(t.Path, p) {
				t.Path = append(t.Path, p)
			}
		}
	}
	if it.Version != nil {
		t.Version = *it.Version
	}
	if it.Live != nil {
		t.Live = *it.Live
	}
	if it.Disk != nil {
		t.Disk = *it.Disk
	}
	return t
}

func (it Item) setting(category SettingCategory) Setting {
	s := Setting{
		Name: it.Name, Description: it.Description, Help: it.Help, Declares: it.Declares, Category: category,
		Interactive: slices.Contains(it.Needs, NeedTerminal), NeedsRoot: slices.Contains(it.Needs, NeedSudo),
		Guided: it.Guided, Reversible: it.Reversible, RunOnce: it.RunOnce, Writes: it.Writes, Sets: it.Sets,
	}
	switch {
	case len(it.Requires) > 0:
		s.Requires = it.Requires[0]
	case len(it.RequiresOneOf) > 0:
		s.Requires = it.RequiresOneOf[0]
		if len(it.RequiresOneOf) > 1 {
			s.OrRequires = it.RequiresOneOf[1:]
		}
	}
	for _, o := range it.Options {
		s.Inputs = append(s.Inputs, Input{
			Name: o.Name, Label: o.Label, Help: o.Help, Kind: InputKind(slices.Index(o.Type.Enum(), string(o.Type))),
			Options: o.Values, DefaultEnv: o.DefaultEnv, Required: o.Required,
		})
	}
	s.Kinds = it.Methods[0].kinds()
	s.Effect = it.Methods[0].effect()
	return s
}
