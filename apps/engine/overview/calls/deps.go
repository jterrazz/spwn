package calls

import (
	"context"
	"time"

	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	"github.com/jterrazz/studio-station/apps/engine/awake"
	backgroundcontract "github.com/jterrazz/studio-station/apps/engine/background/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/overview"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
	settingscontract "github.com/jterrazz/studio-station/apps/engine/settings/contract"
	setupcontract "github.com/jterrazz/studio-station/apps/engine/setup/contract"
)

const ReadBudget = defaultBudget

// value: an owner's file the strict decoder refused: what it says, at its
// file and line, and the file the owner fixes.

type Refusal struct{ Text, File string }

type ThisMac struct {
	Kind catalog.MachineKind
	// value: what the process knows of this machine without asking it.
	Hostname, OS, Arch, User, Shell string
	// value: the owner's home and the system root, so a path reads as the owner types it.
	Home, Root      string
	Version, Commit string
	// value: whether the daemon's job definition is on disk.
	Agent func() bool
}

type Clock interface{ Now() time.Time }

type Deps[App, Item, Misplaced any] struct {
	// value: the reads Overview composes. Tool, Item and Stray are the rows of
	// the Apps, Settings and Disk reads, until those reads are their families' calls.
	Pool *lanes.Pool
	// value: the catalog this run reads: which settings need root.
	Catalog catalog.Catalog
	// value: the owner's file this run refused, when one was: this run
	// reads the built-in packages, and says so in Needs you.
	Refused *Refusal
	// value: the Router the Overview reads each section of this machine
	// through, and the machine's kind, which decides its sections.
	Calls   call.Router
	Kind    machine.Kind
	Clock   Clock
	ThisMac ThisMac
	Name    setupcontract.NamePrefs
	// value: the git identity the owner's preferences expect.
	Identity setupcontract.IdentityPrefs
	// value: the facts of this machine (its kernel, network, protections and
	// updates), and those of its owner (git, gpg, GitHub, the ssh key).
	Facts, Owner settingscontract.Inspector
	Meter        activitycontract.Meter
	// value: this Mac's battery, as Power reads it; nil where there is none.
	Battery   Battery
	Listeners portscontract.Listeners
	Jobs      backgroundcontract.Jobs
	// value: the app updates waiting, on a Mac; nil elsewhere.
	Updates Updates
	Tailnet Tailnet
	Engine  Engine
	Awake   Awake
	// value: the servers this Mac runs; nil where none can run.
	Servers    Servers
	Hypervisor Hypervisor
	Apps       Apps[App]
	Settings   Settings[Item]
	Setup      Setup
	Disk       Disk[Misplaced]
}

type Updates interface {
	// value: the manager's name, and how many of its installs have a newer
	// version; apps contract.ErrMissing where the manager is not installed.
	Manager() string
	Behind(ctx context.Context) (int, error)
}

type Tailnet interface {
	Status(ctx context.Context) (ports.Status, error)
}

type Engine interface {
	Ping(ctx context.Context) (version string, err error)
}

type Awake interface {
	// value: this Mac's keep-awake, held by its engine; ok false where it
	// cannot be read (a VM, no power management, the engine down).
	Session(ctx context.Context) (s overview.AwakeState, ok bool)
}

type Servers interface {
	// value: whether one of this Mac's VMs says it is a server.
	Hosted(ctx context.Context) bool
}

type Battery interface {
	Read() (awake.Reading, error)
}

type Hypervisor interface {
	Name() string
	Installed() bool
}

type Apps[App any] interface {
	Inventory(ctx context.Context) ([]App, error)
}

type Settings[Item any] interface {
	List(ctx context.Context) ([]Item, error)
	// value: the config item a row is, and its state.
	Placed(r Item) (pkg string, state catalog.State)
}

type Setup interface {
	Check(ctx context.Context) ([]overview.Step, error)
}

type DiskReading[Misplaced any] struct {
	Rows      []overview.Disk
	Misplaced []Misplaced
}

type Disk[Misplaced any] interface {
	// invariant: each reading is the whole disk read so far; the last is final.
	Watch(ctx context.Context) <-chan DiskReading[Misplaced]
}
