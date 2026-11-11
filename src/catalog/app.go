package catalog

type MachineKind string

const (
	Mac   MachineKind = "mac"
	Linux MachineKind = "linux"
	MacOS MachineKind = "macos"
)

func StatusV1Kind(k MachineKind) string {
	// value: alias of v2's kind word for a Mac, which station.status/v1 keeps
	// byte for byte.
	if k == Mac {
		return "host"
	}
	return string(k)
}

func (k MachineKind) Article() string {
	if k == Mac {
		return "a Mac"
	}
	return "a " + string(k) + " VM"
}

type LivesOn struct {
	Kinds   []MachineKind
	Install Install
}

type VersionRead struct {
	Args    []string `yaml:"args,omitempty"`
	Pattern string   `yaml:"pattern,omitempty"`
	Path    string   `yaml:"path,omitempty"`
}

type Liveness struct {
	Process string   `yaml:"process,omitempty"`
	Command []string `yaml:"command,omitempty"`
}

type App struct {
	Name string
	// value: manager for an app that installs other apps, app otherwise.
	Kind        Kind `json:",omitempty"`
	Description string
	Category    Category
	Replaces    string
	Supersedes  string
	AppStore    bool
	Installs    []LivesOn
	Provides    []string
	// value: the absolute folders the item's methods land its commands in,
	// off the system's PATH, which station's own search adds so what it
	// installs is found in the same run: its methods' `path:`.
	Path     []string
	Bundle   string
	Version  VersionRead
	Requires []string
	Then     []string
	Live     Liveness
	Disk     Disk
}

// invariant: which machine kinds accept a method is the `kinds` column of the
// method table (src/domain/tools/methods.go reads it from here), so a placement and
// the dispatch that runs it can never disagree.

var MethodKinds = map[Method][]MachineKind{
	MethodBrew:   {Mac, MacOS},
	MethodCask:   {Mac, MacOS},
	MethodMas:    {Mac, MacOS},
	MethodApp:    {Mac, MacOS},
	MethodNpm:    {Mac, MacOS},
	MethodUv:     {Mac, MacOS},
	MethodBun:    {Mac, Linux, MacOS},
	MethodMise:   {Mac, Linux, MacOS},
	MethodScript: {Mac, Linux, MacOS},
	MethodShell:  {Mac, Linux, MacOS},
	MethodSkill:  {Mac, Linux, MacOS},
	// invariant: a command another item brings lives wherever that item does.
	MethodProvidedBy: {Mac, Linux, MacOS},
}

func (MachineKind) Enum() []string { return []string{string(Mac), string(Linux), string(MacOS)} }

func (t App) LivesOn() []MachineKind {
	var kinds []MachineKind
	for _, k := range []MachineKind{Mac, Linux, MacOS} {
		if _, ok := t.InstallOn(k); ok {
			kinds = append(kinds, k)
		}
	}
	return kinds
}

const AppStoreReceipt = "Contents/_MASReceipt"

func (t App) Superseded() (Install, bool) {
	// invariant: what a tool supersedes is a formula, named by Supersedes.
	if t.Supersedes == "" {
		return nil, false
	}
	return BrewInstall{Formula: t.Supersedes}, true
}

func (t App) InstallOn(k MachineKind) (Install, bool) {
	for _, p := range t.Installs {
		for _, kind := range p.Kinds {
			if kind == k {
				return p.Install, true
			}
		}
	}
	return nil, false
}

// value: the items the code names: the hypervisor app and its launcher setting.
const (
	HypervisorApp      = "tart"
	HypervisorLauncher = HypervisorApp + "-launcher"
)
