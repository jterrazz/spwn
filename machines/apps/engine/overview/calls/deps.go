package calls

import (
	"context"
	"time"

	activitycontract "github.com/jterrazz/studio-station/apps/engine/activity/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/call/lanes"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
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
	DefaultName                     func() string
	// value: the owner's home and the system root, so a path reads as the owner types it.
	Home, Root      string
	Version, Commit string
	// value: whether the daemon's job definition is on disk.
	Agent func() bool
}

type Clock interface{ Now() time.Time }

type Deps[Item any] struct {
	// value: the lanes pool, whose shared readings a reload forgets.
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
	// value: the facts of this machine (its kernel, network, protections and
	// updates).
	Facts settingscontract.Inspector
	Meter activitycontract.Meter
	// value: the servers this Mac runs; nil where none can run.
	Servers  Servers
	Settings Settings[Item]
}

type Servers interface {
	// value: whether one of this Mac's VMs says it is a server.
	Hosted(ctx context.Context) bool
}

type Settings[Item any] interface {
	List(ctx context.Context) ([]Item, error)
	// value: the config item a row is, and its state.
	Placed(r Item) (pkg string, state catalog.State)
	// value: Machines keeps a record of applying the setting on this machine.
	Recorded(ctx context.Context, setting string) bool
}
