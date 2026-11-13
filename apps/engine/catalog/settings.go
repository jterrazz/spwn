package catalog

import (
	"io/fs"
	"strings"
)

type SettingCategory string

const (
	CategoryShell    SettingCategory = "Shell"
	CategoryTerminal SettingCategory = "Terminal"
	CategorySecurity SettingCategory = "Security"
	CategoryEditor   SettingCategory = "Editor"
	CategorySystem   SettingCategory = "System"
	CategoryHeadless SettingCategory = "Headless"
	CategoryNetwork  SettingCategory = "Network"
	CategoryAgents   SettingCategory = "Agents"
)

type InputKind int

const (
	InputText InputKind = iota
	InputPassword
	InputSelect
	InputConfirm
)

type Input struct {
	Name, Label, Help string
	Kind              InputKind
	Options           []string
	DefaultEnv        string
	Required          bool
}

type Effect interface{ Kind() EffectKind }

type EffectKind string

const (
	EffectTouch       EffectKind = "touch"
	EffectDotfile     EffectKind = "dotfile"
	EffectGitAliases  EffectKind = "git-aliases"
	EffectShell       EffectKind = "shell"
	EffectAgentHooks  EffectKind = "agent-hooks"
	EffectMuxMachines EffectKind = "mux-machines"
	// value: no Applier writes it yet: applied, it is refused as unhandled.
	EffectMacOSDefaults EffectKind = "macos-defaults"
)

type Touch struct{ Path string }

type Dotfiles struct {
	Files  []Copy
	Reload []string
}

type Copy struct {
	From string      `yaml:"from"`
	To   string      `yaml:"to"`
	Mode fs.FileMode `yaml:"permissions,omitempty"`
}

type GitAliases struct {
	// value: aliases station keeps in one marked block of this git config file.
	File    string     `yaml:"file"`
	Aliases []GitAlias `yaml:"aliases"`
}

type GitAlias struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
}

type MuxMachines struct{}

type AgentHooks struct {
	// value: the apps whose agents station hooks, by catalog name; each
	// installed one gets its hook, the others are left alone.
	Apps []string `yaml:"apps"`
}

type Procedure string

const (
	ProcGitSigning           Procedure = "git-signing"
	ProcSSHKey               Procedure = "ssh-key"
	ProcGitHubLogin          Procedure = "github-login"
	ProcSpotlight            Procedure = "spotlight-exclude"
	ProcDNS                  Procedure = "dns"
	ProcDockReset            Procedure = "dock-reset"
	ProcDockSpacer           Procedure = "dock-spacer"
	ProcAutologin            Procedure = "autologin"
	ProcRestartAfterPowerCut Procedure = "restart-after-power-cut"
	ProcNeverSleeps          Procedure = "never-sleeps"
	ProcLockAfterLogin       Procedure = "lock-after-login"
	ProcSshd                 Procedure = "sshd"
	ProcClaudeToken          Procedure = "claude-token"
	ProcBackupExclusions     Procedure = "backup-exclusions"
	ProcTailscaleCLI         Procedure = "tailscale-cli"
)

func (Touch) Kind() EffectKind         { return EffectTouch }
func (Dotfiles) Kind() EffectKind      { return EffectDotfile }
func (GitAliases) Kind() EffectKind    { return EffectGitAliases }
func (AgentHooks) Kind() EffectKind    { return EffectAgentHooks }
func (MuxMachines) Kind() EffectKind   { return EffectMuxMachines }
func (MacOSDefaults) Kind() EffectKind { return EffectMacOSDefaults }

func (p Procedure) Kind() EffectKind { return EffectKind("procedure:" + string(p)) }

func (p Procedure) Name() string { return string(p) }

func Procedures() []string {
	// value: the procedures this station carries. A builtin a newer catalog
	// names and this list lacks decodes, and its apply is refused with
	// `update station`.
	return []string{
		string(ProcGitSigning), string(ProcSSHKey), string(ProcGitHubLogin), string(ProcSpotlight), string(ProcDNS),
		string(ProcDockReset), string(ProcDockSpacer), string(ProcAutologin), string(ProcRestartAfterPowerCut),
		string(ProcNeverSleeps), string(ProcLockAfterLogin), string(ProcSshd), string(ProcClaudeToken),
		string(ProcBackupExclusions), string(ProcTailscaleCLI),
	}
}

type Declares struct {
	Port int `yaml:"port"`
	// value: the public route that reaches it, by its --via name, and the
	// owner's words for the setting on the port's row.
	Via   string `yaml:"via"`
	Label string `yaml:"label"`
}

type Setting struct {
	// value: a procedure is its own kind of effect: one Applier each.
	Name, Description, Help string
	// value: the port the setting opens to the machine's network: applied,
	// it is declared public; off, its declaration is taken back.
	Declares *Declares `json:"-"`
	Category SettingCategory
	Kinds    []MachineKind
	Requires string
	// value: the apps any one of which serves instead of Requires.
	OrRequires []string `json:",omitempty"`
	Inputs     []Input
	// value: how this machine applies it; never on the wire, so a far
	// Mac's report of the package decodes here (it holds an interface).
	Effect      Effect `json:"-"`
	Interactive bool
	// value: it opens System Settings for the owner to finish it.
	Guided     bool
	NeedsRoot  bool
	Reversible bool
	RunOnce    bool
	Writes     []Stored
	// value: a setting that writes no file: the values station sets, as
	// its Applier reads them back.
	Sets []Value
}

func (it Setting) Needs() []string {
	// value: the apps of which one must be installed for the setting to
	// apply, Requires first; none for a setting that needs no app.
	if it.Requires == "" {
		return nil
	}
	return append([]string{it.Requires}, it.OrRequires...)
}

func (it Setting) NeedsSaid() string {
	// value: the owner's words for what it needs: "claude or codex".
	return strings.Join(it.Needs(), " or ")
}

type Value struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

const GitShortcutsSetting = "git-shortcuts"

const TailscaleCLISetting = "tailscale-cli"

const AgentHooksSetting = "agent-hooks"

const MuxMachinesSetting = "herdr-machine"

const GitShortcutsGroup = "git" // value: the group of v2's run shortcuts, now git's own aliases, and the program each alias is said with.

func GitShortcutLine(alias string) string { return GitShortcutsGroup + " " + alias }

const ContainerShortcutsWas = "docker"

const ContainerSource = "docker"

func (p Procedure) Known() bool {
	for _, k := range Procedures() {
		if k == string(p) {
			return true
		}
	}
	return false
}
