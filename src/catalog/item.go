package catalog

// value: the schema every item file of this station names in its `item:`
// key, and the family its versions share; a file naming a newer version is
// refused with `update station`.

const (
	Schema       = "station.catalog/v1"
	SchemaFamily = "station.catalog"
)

// value: what an item IS (how it installs is its methods): a manager installs
// other items, an app is run or used, a setting is a state of the system.

type Kind string

const (
	KindManager Kind = "manager"
	KindApp     Kind = "app"
	KindSetting Kind = "setting"
)

func (Kind) Enum() []string {
	return []string{string(KindManager), string(KindApp), string(KindSetting)}
}

// value: the operating system a method or a disk row is for, as a machine's
// `system.os` names it; `any` is every one.

type OS string

const (
	OSMacOS  OS = "macos"
	OSUbuntu OS = "ubuntu"
	OSDebian OS = "debian"
	OSAny    OS = "any"
)

func (OS) Enum() []string {
	return []string{string(OSMacOS), string(OSUbuntu), string(OSDebian), string(OSAny)}
}

// value: a lineage of systems, for a method every member of it runs: `linux`
// is every Linux, `debian` Debian and the systems built on it (Ubuntu).

type Family string

const (
	FamilyLinux  Family = "linux"
	FamilyDebian Family = "debian"
)

func (Family) Enum() []string { return []string{string(FamilyLinux), string(FamilyDebian)} }

type Arch string

const (
	ArchARM64 Arch = "arm64"
	ArchAMD64 Arch = "amd64"
)

func (Arch) Enum() []string { return []string{string(ArchARM64), string(ArchAMD64)} }

// value: what a machine is, in the Machinefile's words: a computer you own,
// a workspace (a VM station sets up), a server (a VM running services).

type MachineRole string

const (
	RoleComputer  MachineRole = "computer"
	RoleWorkspace MachineRole = "workspace"
	RoleServer    MachineRole = "server"
)

func (MachineRole) Enum() []string {
	return []string{string(RoleComputer), string(RoleWorkspace), string(RoleServer)}
}

// value: a selector's `machine`: one machine kind, or a list of them.

type Roles []MachineRole

// value: which machines a method or a disk row is for. Every key narrows,
// an absent one keeps all: `os` or `family` the system, `version` a range of
// its versions (`>=24.04`, `>=14, <27`), `arch` its processor, `machine`
// what the machine is.

type Selector struct {
	OS      OS     `yaml:"os,omitempty"`
	Family  Family `yaml:"family,omitempty"`
	Version string `yaml:"version,omitempty"`
	Arch    Arch   `yaml:"arch,omitempty"`
	Machine Roles  `yaml:"machine,omitempty"`
}

// value: what an item needs from the owner, grouped at the head of a plan:
// the password sudo asks, a terminal to ask at, a license to accept (Xcode's),
// the App Store signed in.

type Need string

const (
	NeedSudo          Need = "sudo"
	NeedTerminal      Need = "terminal"
	NeedLicense       Need = "license"
	NeedAppStoreLogin Need = "app-store-login"
)

func (Need) Enum() []string {
	return []string{string(NeedSudo), string(NeedTerminal), string(NeedLicense), string(NeedAppStoreLogin)}
}

// value: when an item is applied: baked into a VM image, at a VM's first
// start, or by `apply` (the default).

type Phase string

const (
	PhaseImage Phase = "image"
	PhaseBoot  Phase = "boot"
	PhaseApply Phase = "apply"
)

func (Phase) Enum() []string {
	return []string{string(PhaseImage), string(PhaseBoot), string(PhaseApply)}
}

// value: a method key a manager item installs by, the one its methods
// name: homebrew manages brew and cask.

type ManagerKey string

func (ManagerKey) Enum() []string {
	return []string{"brew", "cask", "mas", "apt", "npm", "uv", "bun", "mise", "github-release", "skill"}
}

// value: where a manager stands in its system's default priority, lowest
// first: the first manager of the list that has a method for an item
// installs it, unless a Machinefile says `prefer:`. A system's order is
// these numbers across the manager items, each number one manager's. A
// `script` method has no manager and no rank: it is tried after every
// manager of the system; a `github-release` method is ranked by the item
// that manages it.

type Priority struct {
	MacOS  int `yaml:"macos,omitempty"`
	Ubuntu int `yaml:"ubuntu,omitempty"`
	Debian int `yaml:"debian,omitempty"`
}

// value: an app's port and what serves on it: a label a listening port is
// shown with ("probably Vite"), and what the port asks once it is shared:
// `http` says a dev server that speaks HTTP, `env` the variables its server
// is started with, each value a template where `{url}` is the shared URL and
// `{host}` its host (known only when the port is shared).

type PortHint struct {
	Port  int               `yaml:"port"`
	Label string            `yaml:"label"`
	HTTP  bool              `yaml:"http,omitempty"`
	Env   map[string]string `yaml:"env,omitempty"`
}

// value: why an item goes, and the item to take instead.

type Deprecation struct {
	Why string `yaml:"why"`
	Use string `yaml:"use,omitempty"`
}

// value: what has to happen once an item is applied for it to take: a
// process restarted, the owner logged out, the machine rebooted; a plan
// lists each once.

type AfterApply string

const (
	RestartDock    AfterApply = "restart-dock"
	RestartFinder  AfterApply = "restart-finder"
	RestartMenuBar AfterApply = "restart-menu-bar"
	AfterLogout    AfterApply = "logout"
	AfterReboot    AfterApply = "reboot"
)

func (AfterApply) Enum() []string {
	return []string{string(RestartDock), string(RestartFinder), string(RestartMenuBar), string(AfterLogout), string(AfterReboot)}
}

// value: another source whose copy of an app stands for its install:
// present, never managed, upgraded or removed by station.

type Source string

const SourceAppStore Source = "app-store"

func (Source) Enum() []string { return []string{string(SourceAppStore)} }

// value: who moves an app's version: `self`, the app's own updater
// (Claude, Ghostty, a browser), so station checks it is there and never
// updates it nor records its version.

type Updates string

const UpdatesSelf Updates = "self"

func (Updates) Enum() []string { return []string{string(UpdatesSelf)} }

// value: what a method can pin: nothing (it installs the newest), or the
// version a Machinefile asks.

type Pin string

const (
	PinNone    Pin = "none"
	PinVersion Pin = "version"
)

func (Pin) Enum() []string { return []string{string(PinNone), string(PinVersion)} }

type OptionType string

const (
	OptionString  OptionType = "string"
	OptionInteger OptionType = "integer"
	OptionBoolean OptionType = "boolean"
	OptionEnum    OptionType = "enum"
)

func (OptionType) Enum() []string {
	return []string{string(OptionString), string(OptionInteger), string(OptionBoolean), string(OptionEnum)}
}

type Item struct {
	Item        string   `yaml:"item"`
	Kind        Kind     `yaml:"kind"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Help        string   `yaml:"help,omitempty"`
	Replaces    string   `yaml:"replaces,omitempty"`
	Provides    []string `yaml:"provides,omitempty"`
	Requires    []string `yaml:"requires,omitempty"`
	// value: a setting's apps of which any one serves.
	RequiresOneOf []string `yaml:"requires-one-of,omitempty"`
	// value: items applied before this one when both are, never installed
	// for it (requires does that).
	After []string `yaml:"after,omitempty"`
	Then  []string `yaml:"then,omitempty"`
	Phase Phase    `yaml:"phase,omitempty"`
	// value: the names it had before, so a Machinefile that lists one still
	// finds it.
	RenamedFrom []string     `yaml:"renamed-from,omitempty"`
	Deprecated  *Deprecation `yaml:"deprecated,omitempty"`
	// value: a manager's method keys, and its place in each system's
	// default priority.
	Manages    []ManagerKey `yaml:"manages,omitempty"`
	Priority   *Priority    `yaml:"priority,omitempty"`
	Ports      []PortHint   `yaml:"ports,omitempty"`
	Bundle     string       `yaml:"bundle,omitempty"`
	Version    *VersionRead `yaml:"version-read,omitempty"`
	Supersedes string       `yaml:"supersedes,omitempty"`
	Accepts    []Source     `yaml:"accepts,omitempty"`
	Updates    Updates      `yaml:"updates,omitempty"`
	Live       *Liveness    `yaml:"live,omitempty"`
	Needs      []Need       `yaml:"needs,omitempty"`
	AfterApply []AfterApply `yaml:"after-apply,omitempty"`
	Guided     bool         `yaml:"guided,omitempty"`
	Reversible bool         `yaml:"reversible,omitempty"`
	RunOnce    bool         `yaml:"run-once,omitempty"`
	Options    []Option     `yaml:"options,omitempty"`
	Declares   *Declares    `yaml:"declares,omitempty"`
	Sets       []Value      `yaml:"sets,omitempty"`
	Methods    []MethodSpec `yaml:"methods"`
	Writes     []Stored     `yaml:"writes,omitempty"`
	Disk       *Disk        `yaml:"disk,omitempty"`
}

// value: one value the owner gives an item, a form field in the menu bar:
// its type, its default, an integer's bounds, an enum's values, and examples
// a form shows as its placeholder.

type Option struct {
	Name     string        `yaml:"name"`
	Label    string        `yaml:"label"`
	Help     string        `yaml:"help,omitempty"`
	Type     OptionType    `yaml:"type"`
	Default  *OptionValue  `yaml:"default,omitempty"`
	Min      *int          `yaml:"min,omitempty"`
	Max      *int          `yaml:"max,omitempty"`
	Values   []string      `yaml:"values,omitempty"`
	Examples []OptionValue `yaml:"examples,omitempty"`
	Required bool          `yaml:"required,omitempty"`
	// value: a string never shown back nor written to a log: a password, a
	// token.
	Secret bool `yaml:"secret,omitempty"`
	// value: the environment variable whose value is the default.
	DefaultEnv string `yaml:"default-env,omitempty"`
}

// value: an option's default or example as the file writes it, a string,
// an integer or a boolean, kept as its text.

type OptionValue string

type MethodSpec struct {
	Selector `yaml:",inline"`
	// value: absolute folders the method lands its commands in, off the
	// system's PATH, which station's own search adds at once.
	Path []string `yaml:"path,omitempty"`
	// value: the source a manager fetches when it differs from the name it
	// knows the install by (bun's `github:tobi/qmd`, known as qmd).
	Package string `yaml:"package,omitempty"`
	// value: the commands this method lands when they differ from the
	// item's (Debian's fd is `fdfind`).
	Provides []string `yaml:"provides,omitempty"`
	// value: the shell line that upgrades what this method installed, where
	// its manager's own upgrade does not fit.
	Upgrade string `yaml:"upgrade,omitempty"`
	// value: what this method can pin, where it differs from its manager's.
	Pin Pin `yaml:"pin,omitempty"`
	// value: what this method alone asks of the owner (apt's sudo on a
	// Linux computer, mas's App Store login); the item's needs are every
	// method's.
	Needs []Need `yaml:"needs,omitempty"`
	// value: a brew or cask from a third-party tap the owner trusts.
	Trusted       bool                 `yaml:"trusted,omitempty"`
	Brew          string               `yaml:"brew,omitempty"`
	Cask          string               `yaml:"cask,omitempty"`
	Mas           *MasMethod           `yaml:"mas,omitempty"`
	Apt           *AptMethod           `yaml:"apt,omitempty"`
	Npm           string               `yaml:"npm,omitempty"`
	Uv            string               `yaml:"uv,omitempty"`
	Bun           string               `yaml:"bun,omitempty"`
	Mise          *MiseMethod          `yaml:"mise,omitempty"`
	GitHubRelease *GitHubReleaseMethod `yaml:"github-release,omitempty"`
	Script        *ScriptMethod        `yaml:"script,omitempty"`
	Skill         *SkillMethod         `yaml:"skill,omitempty"`
	ProvidedBy    string               `yaml:"provided-by,omitempty"`
	File          *FileMethod          `yaml:"file,omitempty"`
	Builtin       Procedure            `yaml:"builtin,omitempty"`
	MacOSDefaults *MacOSDefaults       `yaml:"macos-defaults,omitempty"`
	GitAliases    *GitAliases          `yaml:"git-aliases,omitempty"`
	ZshFunctions  *ZshFunctions        `yaml:"zsh-functions,omitempty"`
	AgentHooks    *AgentHooks          `yaml:"agent-hooks,omitempty"`
	MuxMachines   *MuxMachines         `yaml:"mux-machines,omitempty"`
}

type MasMethod struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type AptMethod struct {
	Packages []string `yaml:"packages"`
}

type MiseMethod struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// value: a command from a GitHub release: the repository, the asset's name
// pattern (`{os}` and `{arch}` the machine's: `linux` or `darwin`, `arm64`
// or `amd64`), the asset of checksums when the release ships one under a
// name station does not look for, and, for a release that ships no
// checksum at all, why it is installed unverified.

type GitHubReleaseMethod struct {
	Repo       string `yaml:"repo"`
	Asset      string `yaml:"asset"`
	Checksums  string `yaml:"checksums,omitempty"`
	Unverified string `yaml:"unverified,omitempty"`
}

// value: the escape hatch: `check` is the shell line whose exit 0 says it is
// there (the item's provides, without one); the install is `run`, a shell
// line, or `url`, an installer fetched and run; `undo` takes it back; a
// script with neither is installed by the owner, as `hint` says.

type ScriptMethod struct {
	Check string   `yaml:"check,omitempty"`
	Run   string   `yaml:"run,omitempty"`
	URL   string   `yaml:"url,omitempty"`
	Args  []string `yaml:"args,omitempty"`
	Env   []string `yaml:"env,omitempty"`
	// value: the variables added when the run has a terminal, and when it
	// has none: an installer that asks there and keeps quiet elsewhere.
	OnTerminal  []string `yaml:"on-terminal,omitempty"`
	OffTerminal []string `yaml:"off-terminal,omitempty"`
	Undo        string   `yaml:"undo,omitempty"`
	Hint        string   `yaml:"hint,omitempty"`
}

type SkillMethod struct {
	Repo string `yaml:"repo"`
	Name string `yaml:"name"`
}

// value: how a file method writes a file: `file`, the whole file (a backup
// and a diff in the plan), or `block`, station's delimited block in a file
// the owner keeps.

type WriteMode string

const (
	WriteFile  WriteMode = "file"
	WriteBlock WriteMode = "block"
)

func (WriteMode) Enum() []string { return []string{string(WriteFile), string(WriteBlock)} }

type FileMethod struct {
	Touch  string    `yaml:"touch,omitempty"`
	Copy   []Copy    `yaml:"copy,omitempty"`
	Mode   WriteMode `yaml:"mode,omitempty"`
	Reload []string  `yaml:"reload,omitempty"`
}

// value: keys of one macOS defaults domain, the `defaults` command's: check
// reads them, run writes them, undo writes back the values read before.

type MacOSDefaults struct {
	Domain string `yaml:"domain"`
	// value: the keys of this Mac alone (defaults' `-currentHost`).
	ThisMac bool         `yaml:"this-mac,omitempty"`
	Keys    []DefaultKey `yaml:"keys"`
}

// value: one key, its type in defaults' words, and its value, written out
// or an option's (`${option.size}`).

type DefaultKey struct {
	Key   string      `yaml:"key"`
	Type  DefaultType `yaml:"type"`
	Value OptionValue `yaml:"value"`
}

type DefaultType string

func (DefaultType) Enum() []string { return []string{"string", "integer", "float", "boolean"} }
