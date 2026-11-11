package catalog

// value: the schema every Machinefile of this station names in its
// `machinefile:` key, and the family its versions share; a file naming a
// newer version is refused with `update station`.

const (
	MachinefileSchema = "station.machinefile/v1"
	MachinefileFamily = "station.machinefile"
)

// value: what a machine carries, in the order a reader takes it: WHAT it is
// (machine), HOW it runs (vm, a VM's alone), WHAT it inherits (extends,
// prefer), WHAT it carries (apps, settings), WHAT it is lent (accounts) and
// WHAT it may open (ports). A file without machine is a mixin, only ever
// extended. An overlay, `<name>.local.yaml`, names the file it changes in
// overrides and carries apps, settings, accounts and ports alone.

type Machinefile struct {
	Machinefile string       `yaml:"machinefile"`
	Name        string       `yaml:"name,omitempty"`
	Overrides   string       `yaml:"overrides,omitempty"`
	Description string       `yaml:"description,omitempty"`
	Machine     *MachineSpec `yaml:"machine,omitempty"`
	VM          *VMSpec      `yaml:"vm,omitempty"`
	Extends     []string     `yaml:"extends,omitempty"`
	// value: method keys tried first, in this order, before the system's
	// default priority.
	Prefer   []ManagerKey  `yaml:"prefer,omitempty"`
	Apps     *AppList      `yaml:"apps,omitempty"`
	Settings *SettingList  `yaml:"settings,omitempty"`
	Accounts *AccountsSpec `yaml:"accounts,omitempty"`
	Ports    *PortRights   `yaml:"ports,omitempty"`
}

type MachineSpec struct {
	Kind   MachineRole `yaml:"kind"`
	System *SystemSpec `yaml:"system,omitempty"`
}

// value: the system a machine runs: its os, its version (`24.04`, `26`),
// its processor.

type SystemSpec struct {
	OS      OS     `yaml:"os,omitempty"`
	Version string `yaml:"version,omitempty"`
	Arch    Arch   `yaml:"arch,omitempty"`
}

// value: how long a VM is kept: persistent, until deleted; ephemeral,
// deleted once powered off from inside.

type Lifetime string

const (
	LifetimePersistent Lifetime = "persistent"
	LifetimeEphemeral  Lifetime = "ephemeral"
)

func (Lifetime) Enum() []string {
	return []string{string(LifetimePersistent), string(LifetimeEphemeral)}
}

// value: how a VM is on the network: nat, behind its computer with no
// address on the LAN; bridged, its own LAN address.

type Network string

const (
	NetworkNAT     Network = "nat"
	NetworkBridged Network = "bridged"
)

func (Network) Enum() []string { return []string{string(NetworkNAT), string(NetworkBridged)} }

type VMSpec struct {
	Lifetime  Lifetime   `yaml:"lifetime,omitempty"`
	Resources *Resources `yaml:"resources,omitempty"`
	Network   Network    `yaml:"network,omitempty"`
	Image     *ImageSpec `yaml:"image,omitempty"`
}

// value: a size in megabytes, gigabytes or terabytes: `512M`, `32G`, `1T`.

type Size string

type Resources struct {
	CPUs   int  `yaml:"cpus,omitempty"`
	Memory Size `yaml:"memory,omitempty"`
	Disk   Size `yaml:"disk,omitempty"`
}

// value: the image a VM's disk is made from, and the items baked into it
// (their phase is image).

type ImageSpec struct {
	From string   `yaml:"from,omitempty"`
	Bake []string `yaml:"bake,omitempty"`
}

// value: an app a Machinefile lists: its catalog item, the version asked
// (the method must pin it), optional (no method for the system skips it),
// the manager it is installed by instead of the priority's, and its option
// values. Written as its name alone, or as a mapping.

type AppEntry struct {
	Name     string                 `yaml:"name"`
	Version  string                 `yaml:"version,omitempty"`
	Optional bool                   `yaml:"optional,omitempty"`
	Manager  ManagerKey             `yaml:"manager,omitempty"`
	Options  map[string]OptionValue `yaml:"options,omitempty"`
	// value: the line its file lists it at, for a refusal to point at.
	Line int `yaml:"-"`
}

// value: a Machinefile's apps, a list; an overlay's, what it adds to the
// list and the names it removes from it.

type AppList struct {
	List   []AppEntry
	Add    []AppEntry
	Remove []string
	// value: the file writes the overlay's form, add and remove.
	Changes bool
}

// value: a setting a Machinefile carries: its option values by name, a
// secret's as a `vault:<name>` reference; `from: computer`, its values
// read on the computer the VM runs on; optional, no method for the
// system skips it.

type SettingEntry struct {
	From     string
	Optional bool
	Values   map[string]OptionValue
	// value: the line its file names it at, for a refusal to point at.
	Line int
}

// value: the one source `from` names today: the computer a VM runs on.

const FromComputer = "computer"

// value: a Machinefile's settings by name; an overlay's, the settings it
// sets (replacing a same-name one whole) and the names it removes.

type SettingList struct {
	Set    map[string]SettingEntry
	Remove []string
	// value: the file writes the overlay's form, set and remove.
	Changes bool
}

// value: what a machine is lent, and for how long (`7d`, `12h`): each
// account a VM is shared; and on a computer, the accounts it keeps signed
// in.

type AccountsSpec struct {
	Share []AccountShare `yaml:"share,omitempty"`
	Keep  []string       `yaml:"keep,omitempty"`
}

type AccountShare struct {
	Account string `yaml:"account"`
	For     string `yaml:"for,omitempty"`
}

// value: who a port a machine opens is shared with: its computer alone,
// the owner's devices on the private network, or the internet.

type Audience string

const (
	AudienceLocal   Audience = "local"
	AudiencePrivate Audience = "private"
	AudiencePublic  Audience = "public"
)

func (Audience) Enum() []string {
	return []string{string(AudienceLocal), string(AudiencePrivate), string(AudiencePublic)}
}

// value: the ports a machine may open and to whom: a range (`3000-9999`)
// or a list of ports and ranges, its audience, and the ports never shared.

type PortRights struct {
	Open   PortSet  `yaml:"open,omitempty"`
	To     Audience `yaml:"to,omitempty"`
	Except []int    `yaml:"except,omitempty"`
}

// value: ports as a file writes them: one range or one port, or a list.

type PortSet []PortSpan

type PortSpan struct{ From, To int }

// value: a `vault:<name>` option value: the secret's reference, never the
// secret; plans and logs show it masked.

const VaultPrefix = "vault:"
