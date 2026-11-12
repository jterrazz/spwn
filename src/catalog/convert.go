package catalog

import "slices"

// invariant: a selector is the kinds of machine it is for, in the order
// Mac, Linux, macOS VM. Station's Linux machines are Ubuntu VMs and all of
// its machines arm64, so `os: debian`, a Linux computer and `arch: amd64`
// match none of them; a version range is judged by the resolution
// against a machine's system, and here keeps every version.
func (s Selector) kinds() []MachineKind {
	var all []MachineKind
	switch {
	case s.Family != "" || s.OS == OSUbuntu:
		all = []MachineKind{Linux}
	case s.OS == OSMacOS:
		all = []MachineKind{Mac, MacOS}
	case s.OS == OSAny, s.OS == "":
		all = []MachineKind{Mac, Linux, MacOS}
	}
	if s.Arch == ArchAMD64 {
		return nil
	}
	if len(s.Machine) == 0 {
		return all
	}
	computer := slices.Contains(s.Machine, RoleComputer)
	vm := slices.Contains(s.Machine, RoleWorkspace) || slices.Contains(s.Machine, RoleServer)
	kept := make([]MachineKind, 0, len(all))
	for _, k := range all {
		if k == Mac && computer || k != Mac && vm {
			kept = append(kept, k)
		}
	}
	return kept
}

func (s Selector) zero() bool {
	return s.OS == "" && s.Family == "" && s.Version == "" && s.Arch == "" && len(s.Machine) == 0
}

func selectedKinds(s Selector) []MachineKind {
	if s.zero() {
		return nil
	}
	return s.kinds()
}

// invariant: a disk row with no selector is on every machine (On empty);
// one with a selector is on its kinds. The selector is the file's words, On
// the model's: a converted row keeps On alone, as one built in Go does.
func storedOn(rows []Stored) []Stored {
	out := slices.Clone(rows)
	for i := range out {
		out[i].On, out[i].Placed = selectedKinds(out[i].Placed), Selector{}
	}
	return out
}

func actionsOn(rows []Action) []Action {
	out := slices.Clone(rows)
	for i := range out {
		out[i].On, out[i].Placed = selectedKinds(out[i].Placed), Selector{}
	}
	return out
}

func diskOn(d Disk) Disk {
	d.Paths = storedOn(d.Paths)
	d.Free = actionsOn(d.Free)
	d.Erase = actionsOn(d.Erase)
	d.Misplaced = slices.Clone(d.Misplaced)
	for i := range d.Misplaced {
		d.Misplaced[i].Free = actionsOn(d.Misplaced[i].Free)
	}
	return d
}

// invariant: checkMethod let through exactly one app manager key.
func (m MethodSpec) install(provides []string) Install {
	if len(m.Provides) > 0 {
		provides = m.Provides
	}
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
	case m.Apt != nil:
		return AptInstall{Packages: m.Apt.Packages}
	case m.GitHubRelease != nil:
		g := m.GitHubRelease
		return GitHubReleaseInstall{Repo: g.Repo, Asset: g.Asset, Checksums: g.Checksums, Unverified: g.Unverified, Commands: provides}
	case m.Script != nil && m.Script.Run != "":
		return ShellInstall{Line: m.Script.Run, Check: m.Script.Check, Undo: m.Script.Undo}
	case m.Script != nil && m.Script.URL != "":
		return ScriptInstall{URL: m.Script.URL, Args: m.Script.Args, Env: m.Script.Env, Check: m.Script.Check, Undo: m.Script.Undo, Upgrade: m.Upgrade, OnTerminal: m.Script.OnTerminal, OffTerminal: m.Script.OffTerminal}
	case m.Script != nil:
		return AppInstall{Hint: m.Script.Hint}
	case m.Skill != nil:
		return SkillInstall{Repo: m.Skill.Repo, Skill: m.Skill.Name}
	case m.ProvidedBy != "":
		return ProvidedByInstall{Item: m.ProvidedBy, Commands: provides}
	}
	return nil
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
	case m.MacOSDefaults != nil:
		return *m.MacOSDefaults
	}
	return m.Builtin
}

func (it Item) app(category Category) App {
	t := App{
		Name: it.Name, Kind: it.Kind, Description: it.Description, Category: category, Replaces: it.Replaces,
		Supersedes: it.Supersedes, AppStore: slices.Contains(it.Accepts, SourceAppStore), UpdatesSelf: it.Updates == UpdatesSelf, Provides: it.Provides, Bundle: it.Bundle,
		Requires: it.Requires, Then: it.Then, Needs: it.Needs,
	}
	for _, m := range it.Methods {
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
		t.Disk = diskOn(*it.Disk)
	}
	return t
}

func (it Item) setting(category SettingCategory) Setting {
	s := Setting{
		Name: it.Name, Description: it.Description, Help: it.Help, Declares: it.Declares, Category: category,
		Interactive: slices.Contains(it.Needs, NeedTerminal), NeedsRoot: slices.Contains(it.Needs, NeedSudo),
		Guided: it.Guided, Reversible: it.Reversible, RunOnce: it.RunOnce, Writes: storedOn(it.Writes), Sets: it.Sets,
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
			Name: o.Name, Label: o.Label, Help: o.Help, Kind: o.input(),
			Options: o.Values, DefaultEnv: o.DefaultEnv, Required: o.Required,
		})
	}
	s.Kinds = it.Methods[0].kinds()
	s.Effect = it.Methods[0].effect()
	return s
}

// invariant: a form asks an option by its type: a secret string hidden as
// it is typed, an enum as a choice, a boolean as a yes or no, an integer
// and any other string as text.
func (o Option) input() InputKind {
	switch o.Type {
	case OptionString:
		if o.Secret {
			return InputPassword
		}
	case OptionEnum:
		return InputSelect
	case OptionBoolean:
		return InputConfirm
	case OptionInteger:
	}
	return InputText
}
