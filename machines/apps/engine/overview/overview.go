package overview

import (
	"path"
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

func Late(name string, budget time.Duration) Check {
	// invariant: a row past its budget answers unknown, whatever its tool says.
	return Check{Name: name, State: catalog.StateUnknown, Value: "?", Detail: "no answer in " + budget.String()}
}

func rank(s catalog.State) int {
	switch s {
	case catalog.StateOK:
		return 0
	case catalog.StateUnknown:
		return 1
	case catalog.StateDegraded:
		return 2
	case catalog.StateAbsent:
		return 3
	case catalog.StateBroken:
		return 4
	case catalog.StatePending:
	}
	return 1
}

func Worse(a, than catalog.State) bool { return rank(a) > rank(than) }

func SetupOf(steps []Step) Setup {
	state := catalog.StateOK
	for _, step := range steps {
		if Worse(step.State, state) {
			state = step.State
		}
	}
	if steps == nil {
		steps = []Step{}
	}
	return Setup{State: state, Steps: steps}
}

func Gateways(thisMac catalog.MachineKind, t ports.Status) []Check {
	// invariant: the tailnet's Serve is the way every VM service leaves the host, so a
	// host whose tailnet is connected says whether it is on, and the admin step
	// when it is not.
	if thisMac != catalog.Mac || !t.Connected {
		return []Check{}
	}
	if t.ServeOn {
		return []Check{{Name: catalog.ServeRow, State: catalog.StateOK, Value: "on"}}
	}
	return []Check{{Name: catalog.ServeRow, State: catalog.StateDegraded, Value: "off", Detail: catalog.ServeOffDetail}}
}

func Protection(item catalog.State, name string) Check {
	// invariant: a protection a config item installs is read from that item's own state,
	// so status and `station config` can never disagree about it.
	switch item {
	case catalog.StateOK:
		return Check{Name: name, State: catalog.StateOK, Value: "on"}
	case catalog.StateAbsent:
		return Check{Name: name, State: catalog.StateAbsent, Value: "off"}
	case catalog.StateDegraded, catalog.StateBroken, catalog.StateUnknown, catalog.StatePending:
	}
	return Check{Name: name, State: catalog.StateUnknown}
}

func Protections() [][2]string {
	// value: the config items whose state is a protection row, and its name.
	return [][2]string{{"sshd", "remote-login"}, {"dns", "encrypted-dns"}}
}

func ShellName(shell string) string {
	if shell == "" {
		return ""
	}
	return path.Base(shell)
}
