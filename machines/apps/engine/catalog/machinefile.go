package catalog

import (
	"maps"
	"slices"
	"time"
)

// value: the schema every Machinefile of this station names in its
// `machinefile:` key, and the family its versions share; a file naming a
// newer version is refused with `update station`.

const (
	MachinefileSchema = "station.machinefile/v1"
	MachinefileFamily = "station.machinefile"
)

// value: what a machine carries, in the order a reader takes it: WHAT it is
// (machine), HOW it runs (vm, a VM's alone), WHAT it inherits (extends,
// prefer), WHAT it carries (apps, settings), WHAT it is lent (accounts),
// WHAT it may open (ports) and the owner's own lines per phase (run). A
// file without machine is a mixin, only ever extended. An overlay, `<name>.local.yaml`, names the file it changes in
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
	Run      *RunSpec      `yaml:"run,omitempty"`
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
	Mounts    []Mount    `yaml:"mounts,omitempty"`
}

// value: a folder of the computer shared live with a VM made from the
// Machinefile, at a folder of the guest (`~/` opens at each side's home),
// read-only when the VM may not write to it.

type Mount struct {
	Computer string `yaml:"computer"`
	Guest    string `yaml:"guest"`
	ReadOnly bool   `yaml:"read-only,omitempty"`
}

// value: the owner's own shell lines, one block per phase, run as the
// machine's user once its packages are carried out: image, as the image is
// built; boot, at a VM's first start; upgrade, at each upgrade, after its
// apps. The plan shows them before they run.

type RunSpec struct {
	Image   string `yaml:"image,omitempty"`
	Boot    string `yaml:"boot,omitempty"`
	Upgrade string `yaml:"upgrade,omitempty"`
}

// value: the bounds of a phase run inside a VM (image, boot): its packages
// carried out within their own share, then its owner's lines within
// theirs, every file's of the chain within that one share together, so
// the bound of the whole is the sum, and lines that run long
// end as a broken run step, never as the phase cut short.
const (
	PackagesBound = 15 * time.Minute
	RunBound      = 20 * time.Minute
	PhaseBound    = PackagesBound + RunBound
)

const (
	// value: how long past the bound a caller gives an apply
	// (machinefile.apply's Within) the package already running at it may
	// go on: a package manager is never stopped mid-transaction before it,
	// so a package waiting on its manager's lock or installing then ends
	// whole; past it the install is ended, all it started told to stop,
	// then killed.
	InFlightCap = 3 * time.Minute
)

// value: a size in megabytes, gigabytes or terabytes: `512M`, `32G`, `1T`.

type Size string

type Resources struct {
	CPUs   int  `yaml:"cpus,omitempty"`
	Memory Size `yaml:"memory,omitempty"`
	Disk   Size `yaml:"disk,omitempty"`
}

// value: the image a VM's disk is made from, and the packages baked into it
// (their phase is image).

type ImageSpec struct {
	From string   `yaml:"from,omitempty"`
	Bake []string `yaml:"bake,omitempty"`
}

// value: an app a Machinefile lists: its catalog package, the range of
// versions it may move within (its method must hold it), optional (no method for the system skips it),
// the manager it is installed by instead of the priority's, and its option
// values. Written as its name alone, or as a mapping.

type AppEntry struct {
	Name     string                 `yaml:"name"`
	Version  Range                  `yaml:"version,omitempty"`
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

// value: the programs a machine signs itself in to, and those this Mac's
// vault grants a workspace made from the file, each by its wire name.

type AccountsSpec struct {
	SignIns []ProgramName `yaml:"signins,omitempty"`
	Grants  []ProgramName `yaml:"grants,omitempty"`
}

type ProgramName string

func (ProgramName) Enum() []string {
	names := make([]string, 0, len(LoginPrograms))
	for _, p := range LoginPrograms {
		names = append(names, p.Name)
	}
	return names
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
// or a list of ports and ranges, its audience, the ports never shared, and
// the ports given an audience of their own.

type PortRights struct {
	Open      PortSet       `yaml:"open,omitempty"`
	To        Audience      `yaml:"to,omitempty"`
	Except    []int         `yaml:"except,omitempty"`
	Audiences PortAudiences `yaml:"audiences,omitempty"`
}

// value: ports by number, each opened to its own audience whatever open,
// to and except say (`audiences: { 5432: local }`).

type PortAudiences map[int]Audience

func (PortAudiences) JSONSchema() map[string]any {
	return map[string]any{
		"type": "object", "propertyNames": map[string]any{"pattern": "^[1-9][0-9]{0,4}$"},
		"additionalProperties": map[string]any{"type": "string", "enum": Audience("").Enum()},
	}
}

func (p PortRights) Public() []int {
	// value: the ports the rights open to the internet: each port of open when
	// they open it to public and audiences gives it none of its own, then each
	// port audiences opens to public, lowest first.
	var out []int
	if p.To == AudiencePublic {
		for _, s := range p.Open {
			if _, own := p.Audiences[s.From]; s.From == s.To && !own {
				out = append(out, s.From)
			}
		}
	}
	for _, port := range p.Audienced() {
		if p.Audiences[port] == AudiencePublic && !slices.Contains(out, port) {
			out = append(out, port)
		}
	}
	return out
}

func (p PortRights) Over(o PortRights) PortRights {
	// value: these rights with an overlay's changed key by key: each of
	// open, to and except it says replaces that key alone, and audiences
	// merges port by port.
	out := p
	if o.Open != nil {
		out.Open = o.Open
	}
	if o.To != "" {
		out.To = o.To
	}
	if o.Except != nil {
		out.Except = o.Except
	}
	if o.Audiences != nil {
		out.Audiences = maps.Clone(p.Audiences)
		if out.Audiences == nil {
			out.Audiences = PortAudiences{}
		}
		maps.Copy(out.Audiences, o.Audiences)
	}
	return out
}

func (p PortRights) Audienced() []int {
	// value: the ports audiences names, lowest first.
	return slices.Sorted(maps.Keys(p.Audiences))
}

// value: ports as a file writes them: one range or one port, or a list.

type PortSet []PortSpan

type PortSpan struct{ From, To int }

// value: a `vault:<name>` option value: the secret's reference, never the
// secret; plans and logs show it masked.

const VaultPrefix = "vault:"
