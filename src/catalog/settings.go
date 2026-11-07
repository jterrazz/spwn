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
	EffectTouch        EffectKind = "touch"
	EffectDotfile      EffectKind = "dotfile"
	EffectGitAliases   EffectKind = "git-aliases"
	EffectZshFunctions EffectKind = "zsh-functions"
	EffectAgentHooks   EffectKind = "agent-hooks"
)

type Touch struct{ Path string }

type Dotfiles struct {
	Files  []Copy
	Reload []string
}

type Copy struct {
	From, To string
	Mode     fs.FileMode
}

type GitAliases struct {
	// value: aliases station keeps in one marked block of this git config file.
	File    string
	Aliases []GitAlias
}

type GitAlias struct{ Name, Command string }

type ZshFunctions struct {
	// value: the embedded script of the functions station's zsh block defines.
	From      string
	Functions []string
}

type AgentHooks struct {
	// value: the apps whose agents station hooks, by catalogue name; each
	// installed one gets its hook, the others are left alone.
	Apps []string
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
)

func (Touch) Kind() EffectKind        { return EffectTouch }
func (Dotfiles) Kind() EffectKind     { return EffectDotfile }
func (GitAliases) Kind() EffectKind   { return EffectGitAliases }
func (ZshFunctions) Kind() EffectKind { return EffectZshFunctions }
func (AgentHooks) Kind() EffectKind   { return EffectAgentHooks }

func (p Procedure) Kind() EffectKind { return EffectKind("procedure:" + string(p)) }

func (p Procedure) Name() string { return string(p) }

type Declares struct {
	Port int
	// value: the public route that reaches it, by its --via name, and the
	// owner's words for the setting on the port's row.
	Via, Label string
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
	// Mac's report of the item decodes here (it holds an interface).
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

func (it Setting) AlwaysOn() bool {
	// invariant: Always-on Mac changes how a Mac sleeps, logs in and is
	// reached, so it is applied by name only, never by setup.
	return it.Category == CategoryHeadless
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

func (it Setting) NeedsOwner() bool {
	// invariant: a value, admin rights, the owner at System Settings, or no
	// trace to tell it was applied: setup lists it as a next step.
	return len(it.Inputs) > 0 || it.Interactive || it.Guided || it.NeedsRoot || it.RunOnce
}

type Value struct{ Key, Value string }

var (
	everyKind = []MachineKind{Mac, Linux, MacOS}
	macKinds  = []MachineKind{Mac, MacOS}
	macKind   = []MachineKind{Mac}
	// value: the kinds a VM is: inside one, station manages only a
	// workspace's settings.
	vmKinds = []MachineKind{Linux, MacOS}
)

var Settings = []Setting{
	{Name: "hushlogin", Description: "Silence the terminal login message", Category: CategoryTerminal, Kinds: everyKind, Effect: Touch{Path: "~/.hushlogin"}, Writes: []Stored{conf("~/.hushlogin")}},
	{Name: "ghostty-dotfiles", Description: "The Ghostty settings and its Catppuccin themes", Category: CategoryTerminal, Kinds: macKinds, Requires: "ghostty", Effect: Dotfiles{Files: []Copy{
		{From: "ghostty/config", To: "~/.config/ghostty/config"},
		{From: "ghostty/themes/catppuccin-espresso-blur", To: "~/.config/ghostty/themes/catppuccin-espresso-blur"},
		{From: "ghostty/themes/catppuccin-latte-blur", To: "~/.config/ghostty/themes/catppuccin-latte-blur"},
	}}},
	{Name: "tmux-dotfiles", Description: "The tmux settings", Category: CategoryTerminal, Kinds: everyKind, Requires: "tmux", Effect: Dotfiles{
		Files:  []Copy{{From: "tmux/tmux.conf", To: "~/.tmux.conf"}},
		Reload: []string{"tmux", "source-file", "~/.tmux.conf"},
	}},
	{Name: "starship-dotfiles", Description: "The Starship prompt settings", Category: CategoryTerminal, Kinds: everyKind, Requires: "starship", Effect: Dotfiles{Files: []Copy{{From: "starship/starship.toml", To: "~/.config/starship.toml"}}}},
	{Name: "git-signing", Description: "Sign every commit with a GPG key", Category: CategorySecurity, Kinds: macKinds, Requires: "gpg", Effect: ProcGitSigning, Sets: []Value{{"commit.gpgsign", "true"}, {"gpg.program", "gpg"}}, Interactive: true, Writes: []Stored{kept("~/.gnupg"), noted(conf("~/.gitconfig"), "`git config --global` user.signingkey, commit.gpgsign")}},
	{Name: "ssh-key", Description: "An ed25519 SSH key, kept in the Keychain", Category: CategorySecurity, Kinds: macKind, Effect: ProcSSHKey, Sets: []Value{{"~/.ssh/id_ed25519", "present"}, {"~/.ssh/config", "the Keychain stanza"}}, Interactive: true, Writes: []Stored{kept("~/.ssh/id_ed25519"), kept("~/.ssh/id_ed25519.pub"), noted(conf("~/.ssh/config"), "the `Host *` stanza: `UseKeychain yes`, `IdentityFile ~/.ssh/id_ed25519`")}},
	{Name: "github-login", Description: "The GitHub CLI logged in over SSH", Category: CategorySecurity, Kinds: everyKind, Requires: "gh", Effect: ProcGitHubLogin, Sets: []Value{{"gh", "logged in"}}, Interactive: true, Writes: []Stored{noted(kept("{config}/gh/hosts.yml"), "gh's login")}},
	{Name: "spotlight-exclude", Description: "Exclude ~/Developer from Spotlight (guided, opens Settings)", Category: CategorySecurity, Kinds: macKinds, Effect: ProcSpotlight, Sets: []Value{{"~/Developer", "excluded from Spotlight"}}, Guided: true, Writes: []Stored{noted(conf("/System/Volumes/Data/.Spotlight-V100"), "the Spotlight privacy list, through System Settings")}},
	{Name: "dns", Description: "Encrypted DNS via Quad9 (DoH)", Category: CategorySecurity, Kinds: macKind, Effect: ProcDNS, Sets: []Value{{"encrypted DNS", "Quad9 DoH"}}, Guided: true, Writes: []Stored{noted(conf("{state}/station/dns/quad9-dns.mobileconfig"), "installed as the profile `com.jterrazz.dns.quad9`")}},
	{Name: "claude-token", Description: "The Claude Code subscription token a VM may borrow", Help: "run `claude setup-token` and paste the token", Category: CategorySecurity, Kinds: macKind, Effect: ProcClaudeToken, Sets: []Value{{"token", "kept by the studio credential store"}, {"share list", "holds it"}}, Reversible: true, Inputs: []Input{
		{Name: "token", Label: "Claude Code token", Help: "run `claude setup-token` and paste the token", Kind: InputPassword, Required: true},
	}, Writes: []Stored{noted(kept("{config}/studio-os"), "`operator:claude:oauth-token` in the OS credential store")}},
	{Name: "zed-dotfiles", Description: "The Zed settings and its formatter settings", Category: CategoryEditor, Kinds: macKinds, Requires: "zed", Effect: Dotfiles{Files: []Copy{
		{From: "zed/settings.json", To: "~/.config/zed/settings.json"},
		{From: "zed/.oxfmtrc.json", To: "~/.config/zed/.oxfmtrc.json"},
	}}},
	{Name: "dock-reset", Description: "Put the Dock back to the system defaults", Category: CategorySystem, Kinds: macKinds, Effect: ProcDockReset, Sets: []Value{{"Dock", "back to the system defaults"}}, RunOnce: true, Writes: []Stored{noted(conf("~/Library/Preferences/com.apple.dock.plist"), "`com.apple.dock` defaults")}},
	{Name: "dock-spacer", Description: "Add a small spacer tile to the dock", Category: CategorySystem, Kinds: macKinds, Effect: ProcDockSpacer, Sets: []Value{{"Dock", "spacer tile added"}}, RunOnce: true, Writes: []Stored{noted(conf("~/Library/Preferences/com.apple.dock.plist"), "`com.apple.dock` defaults: a spacer tile")}},
	{Name: "autologin", Description: "GUI auto-login for jterrazz.agent (FileVault-aware)", Category: CategoryHeadless, Kinds: macKind, Effect: ProcAutologin, Sets: []Value{{"autoLoginUser", "jterrazz.agent"}, {"/etc/kcpassword", "present"}}, Interactive: true, NeedsRoot: true, Reversible: true, Inputs: []Input{
		{Name: "password", Label: "Password of jterrazz.agent", Kind: InputPassword, DefaultEnv: "AGENT_PASSWORD", Required: true},
	}, Writes: []Stored{noted(kept("/etc/kcpassword"), "the agent user's login password, obfuscated"), noted(conf("/Library/Preferences/com.apple.loginwindow"), "`autoLoginUser` in `com.apple.loginwindow`")}},
	{Name: "restart-after-power-cut", Description: "Restarts after a power cut, and wakes for the network", Category: CategoryHeadless, Kinds: macKind, Effect: ProcRestartAfterPowerCut, Sets: []Value{{"autorestart", "1"}, {"womp", "1"}}, NeedsRoot: true, Reversible: true, Writes: []Stored{noted(conf("/Library/Preferences/SystemConfiguration/com.apple.PowerManagement.plist"), "pmset: autorestart, wake for network")}},
	{Name: "never-sleeps", Description: "Never sleeps: the engine keeps this Mac awake, its servers with it", Category: CategoryHeadless, Kinds: macKind, Effect: ProcNeverSleeps, Sets: []Value{{"never sleeps", "held by the engine"}}, Reversible: true, Writes: []Stored{noted(conf("{config}/station/preferences.json"), "`never_sleeps`, the engine holds an idle-sleep assertion while it is set")}},
	{Name: "lock-after-login", Description: "A LaunchAgent that locks the screen ~20s after auto-login", Category: CategoryHeadless, Kinds: macKind, Effect: ProcLockAfterLogin, Sets: []Value{{"agent", "present"}, {"loaded", "yes"}}, NeedsRoot: true, Reversible: true, Writes: []Stored{noted(conf("/Users/jterrazz.agent/Library/LaunchAgents/ai.jterrazz.lock-after-login.plist"), "the agent user's `lock-after-login.plist`"), noted(cache("/Users/jterrazz.agent/.openclaw/logs"), "its logs")}},
	{Name: "sshd", Description: "macOS Remote Login (sshd)", Category: CategoryHeadless, Kinds: macKind, Effect: ProcSshd, Declares: &Declares{Port: 22, Via: RouteLAN, Label: "Remote login"}, Sets: []Value{{"Remote Login", "on"}}, NeedsRoot: true, Reversible: true, Writes: []Stored{noted(conf("/System/Library/LaunchDaemons/ssh.plist"), "Remote Login, through systemsetup")}},
	{Name: "backup-exclusions", Description: "Keep ~/.cache and station's VMs out of Time Machine", Category: CategorySystem, Kinds: macKind, Effect: ProcBackupExclusions, Reversible: true, Writes: []Stored{
		noted(conf("{cache}"), "a sticky Time Machine exclusion (`tmutil addexclusion`)"),
		noted(conf("{data}/station/vms"), "a sticky Time Machine exclusion (`tmutil addexclusion`)"),
	}},
	{Name: "tart-launcher", Description: "The tart launcher: Tart's home is station's VMs folder", Category: CategorySystem, Kinds: macKind, Effect: Dotfiles{Files: []Copy{{From: "tart/tart", To: "~/.local/bin/tart", Mode: 0o755}}}},
	{Name: GitShortcutsSetting, Description: "Git shortcuts: git wip, unwip, sync, feat, fix, chore, branches, lg, st, pushup", Category: CategoryShell, Kinds: everyKind, Effect: GitShortcuts, Reversible: true, Writes: []Stored{noted(conf("~/.gitconfig"), "station's `[alias]` block")}},
	{Name: AgentHooksSetting, Description: "The agent hook: station sees what Claude Code and Codex do, and what they wait for", Category: CategoryAgents, Kinds: vmKinds, Requires: "claude", OrRequires: []string{"codex"}, Effect: AgentHooks{Apps: []string{"claude", "codex"}}, Reversible: true, Writes: []Stored{
		noted(conf("~/.claude/settings.json"), "station's hook under `hooks`, every other key the owner's"),
		noted(conf("~/.codex/config.toml"), "`notify`, unless the owner set it to another program"),
	}},
	{Name: "docker-shortcuts", Description: "Docker shortcuts: dclean, dclean-images, dreset, each asks first", Category: CategoryShell, Kinds: everyKind, Effect: DockerShortcuts, Reversible: true, Writes: []Stored{noted(conf("{config}/station/zsh/docker-shortcuts.zsh"), "the functions station's zsh block defines")}},
}

const GitShortcutsSetting = "git-shortcuts"

const AgentHooksSetting = "agent-hooks"

const GitShortcutsGroup = "git" // value: the group of v2's run shortcuts, now git's own aliases, and the program each alias is said with.

func GitShortcutLine(alias string) string { return GitShortcutsGroup + " " + alias }

var GitShortcuts = GitAliases{File: "~/.gitconfig", Aliases: []GitAlias{
	{Name: "wip", Command: "!git add --all && git commit -m WIP"},
	{Name: "unwip", Command: "!git reset --soft HEAD~1 && git reset HEAD"},
	{Name: "sync", Command: "!git fetch -p; git pull"},
	{Name: "feat", Command: commitAs("feat")},
	{Name: "fix", Command: commitAs("fix")},
	{Name: "chore", Command: commitAs("chore")},
	{Name: "branches", Command: "branch"},
	{Name: "lg", Command: "log --oneline -10"},
	{Name: "st", Command: "status"},
	{Name: "pushup", Command: "push -u origin HEAD"},
}}

const ContainerShortcutsWas = "docker"

var DockerShortcuts = ZshFunctions{From: "zsh/docker-shortcuts.zsh", Functions: []string{"dclean", "dclean-images", "dreset"}}

func commitAs(prefix string) string {
	return `!f() { git add . && git commit -m "` + prefix + `: $*"; }; f`
}

const ContainerSource = "docker"
