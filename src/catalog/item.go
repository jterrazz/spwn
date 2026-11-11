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

type OS string

const (
	OSMacOS OS = "macos"
	OSLinux OS = "linux"
	OSAny   OS = "any"
)

func (OS) Enum() []string { return []string{string(OSMacOS), string(OSLinux), string(OSAny)} }

// value: a method's selector narrowed to a Mac itself or to a VM.

type Only string

const (
	OnlyMac Only = "mac"
	OnlyVM  Only = "vm"
)

func (Only) Enum() []string { return []string{string(OnlyMac), string(OnlyVM)} }

// value: what a setting needs from the owner when it applies: the one root
// line typed, or a terminal to ask at.

type Need string

const (
	NeedSudo     Need = "sudo"
	NeedTerminal Need = "terminal"
)

func (Need) Enum() []string { return []string{string(NeedSudo), string(NeedTerminal)} }

type OptionType string

func (OptionType) Enum() []string { return []string{"text", "password", "select", "confirm"} }

type Item struct {
	Item string `yaml:"item"`
	Kind Kind   `yaml:"kind"`
	Name string `yaml:"name"`
	// value: where the item stands in the catalog's lists, lowest first; an
	// item without one comes after every ordered item, by name.
	Order       int      `yaml:"order,omitempty"`
	Description string   `yaml:"description"`
	Help        string   `yaml:"help,omitempty"`
	Replaces    string   `yaml:"replaces,omitempty"`
	Provides    []string `yaml:"provides,omitempty"`
	Requires    []string `yaml:"requires,omitempty"`
	// value: a setting's apps of which any one serves.
	RequiresOneOf []string     `yaml:"requires-one-of,omitempty"`
	Then          []string     `yaml:"then,omitempty"`
	Bundle        string       `yaml:"bundle,omitempty"`
	Version       *VersionRead `yaml:"version,omitempty"`
	Supersedes    string       `yaml:"supersedes,omitempty"`
	AppStore      bool         `yaml:"app-store,omitempty"`
	Live          *Liveness    `yaml:"live,omitempty"`
	Needs         []Need       `yaml:"needs,omitempty"`
	Guided        bool         `yaml:"guided,omitempty"`
	Reversible    bool         `yaml:"reversible,omitempty"`
	RunOnce       bool         `yaml:"run-once,omitempty"`
	Options       []Option     `yaml:"options,omitempty"`
	Declares      *Declares    `yaml:"declares,omitempty"`
	Sets          []Value      `yaml:"sets,omitempty"`
	Methods       []MethodSpec `yaml:"methods"`
	Writes        []Stored     `yaml:"writes,omitempty"`
	Disk          *Disk        `yaml:"disk,omitempty"`
}

type Option struct {
	Name     string     `yaml:"name"`
	Label    string     `yaml:"label"`
	Help     string     `yaml:"help,omitempty"`
	Type     OptionType `yaml:"type"`
	Values   []string   `yaml:"values,omitempty"`
	Required bool       `yaml:"required,omitempty"`
	// value: the environment variable whose value is the default.
	DefaultEnv string `yaml:"default-env,omitempty"`
}

type MethodSpec struct {
	OS   OS   `yaml:"os"`
	Only Only `yaml:"only,omitempty"`
	// value: absolute folders the method lands its commands in, off the
	// system's PATH, which station's own search adds at once.
	Path []string `yaml:"path,omitempty"`
	// value: the source a manager fetches when it differs from the name it
	// knows the install by (bun's `github:tobi/qmd`, known as qmd).
	Package string `yaml:"package,omitempty"`
	// value: a brew or cask from a third-party tap the owner trusts.
	Trusted    bool          `yaml:"trusted,omitempty"`
	Brew       string        `yaml:"brew,omitempty"`
	Cask       string        `yaml:"cask,omitempty"`
	Mas        *MasMethod    `yaml:"mas,omitempty"`
	Npm        string        `yaml:"npm,omitempty"`
	Uv         string        `yaml:"uv,omitempty"`
	Bun        string        `yaml:"bun,omitempty"`
	Mise       *MiseMethod   `yaml:"mise,omitempty"`
	Script     *ScriptMethod `yaml:"script,omitempty"`
	Skill      *SkillMethod  `yaml:"skill,omitempty"`
	ProvidedBy string        `yaml:"provided-by,omitempty"`
	// value: no manager installs it: the owner does, as the hint says.
	Manual       string        `yaml:"manual,omitempty"`
	File         *FileMethod   `yaml:"file,omitempty"`
	Builtin      Procedure     `yaml:"builtin,omitempty"`
	GitAliases   *GitAliases   `yaml:"git-aliases,omitempty"`
	ZshFunctions *ZshFunctions `yaml:"zsh-functions,omitempty"`
	AgentHooks   *AgentHooks   `yaml:"agent-hooks,omitempty"`
	MuxMachines  *MuxMachines  `yaml:"mux-machines,omitempty"`
}

type MasMethod struct {
	ID   int    `yaml:"id"`
	Name string `yaml:"name"`
}

type MiseMethod struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ScriptMethod struct {
	URL  string   `yaml:"url,omitempty"`
	Args []string `yaml:"args,omitempty"`
	Env  []string `yaml:"env,omitempty"`
	// value: the shell line that updates what the installer put, for an
	// installer that refuses to run over its own install.
	Upgrade string `yaml:"upgrade,omitempty"`
	Run     string `yaml:"run,omitempty"`
}

type SkillMethod struct {
	Repo string `yaml:"repo"`
	Name string `yaml:"name"`
}

type FileMethod struct {
	Touch  string   `yaml:"touch,omitempty"`
	Copy   []Copy   `yaml:"copy,omitempty"`
	Reload []string `yaml:"reload,omitempty"`
}
