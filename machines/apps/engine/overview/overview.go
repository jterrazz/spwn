package overview

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/activity"
	"github.com/jterrazz/studio-station/apps/engine/background"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type Snapshot[App, Item, Misplaced any] struct {
	// value: one reading of this machine. Tool, Item and Stray are the rows of
	// the Apps, Settings and Disk reads, which keep their own types.
	GeneratedAt time.Time
	Station     Station
	Machine     Machine
	System      System
	Environment Environment
	Workspace   Workspace[Misplaced]
	Apps        []App
	Settings    []Item
	Setup       Setup
	Identity    []Check
	Background  []background.Task
	Gateways    []Check
	Tailnet     ports.Status
	// value: the Mac readiness checklist, empty on a Mac that needs none.
	Readiness []Check
	Awake     AwakeState
	// value: Awake was read (a Mac whose engine answers).
	AwakeKnown bool
	// value: a section whose reading has not answered; never past a whole read.
	AppsPending, SettingsPending bool
	// value: what failed the reading, when a section's read did.
	Err error
}

type Station struct {
	Version, Commit string
	Engine          Check
}

type Machine struct {
	Name       string
	Registered bool
	Kind       catalog.MachineKind
	Hostname   string
	OS         string
	Arch       string
	User       string
	Shell      string
}

type Check struct {
	Name   string
	State  catalog.State
	Value  string
	Detail string
}

type Step struct {
	Name   string
	State  catalog.State
	Detail string
}

type Setup struct {
	State catalog.State
	Steps []Step
}

type System struct {
	Uptime      string
	Battery     string
	MemoryTotal int64
	Reading     activity.Reading
	Pending     bool
}

type Port struct {
	PID  int
	Name string
	Port string
}

type Environment struct {
	Network     []Check
	Ports       []Port
	Protections []Check
	Updates     []Check
}

type Disk struct {
	Name  string
	Level catalog.Level
	Bytes int64
}

type Workspace[Misplaced any] struct {
	Disk      []Disk
	Misplaced []Misplaced
	// value: the disk row has not answered; never past a whole read.
	DiskPending bool
}

type AwakeState struct {
	Held, Lid bool
	Until     time.Time
}
