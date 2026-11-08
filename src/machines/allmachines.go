package machines

import (
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/machine"
)

const SuperviseRetired = "supervise is retired: the engine supervises VMs (station setup)"

type Node struct {
	Online bool
	IP     string
}

type Reserved struct {
	GuestPort, MacPort int
	Profile, URL       string
	PublicName         string
	Off                string
}

type Resources struct {
	CPUs        int
	MemoryBytes int64
	DiskBytes   int64
}

func (r Resources) Size() Size {
	return Size{CPUs: r.CPUs, Memory: r.MemoryBytes, Disk: r.DiskBytes}
}

type VMState string

const (
	Running   VMState = "running"
	Stopped   VMState = "stopped"
	Suspended VMState = "suspended"
	Stale     VMState = "stale"
	Broken    VMState = "broken"
	// value: on a Mac that does not answer: as it was at its last answer,
	// its state unknown.
	UnknownState VMState = "unknown"
)

type VM struct {
	Name, Image, Version, Mac  string
	Kind                       catalog.MachineKind
	Lifetime                   Lifetime
	State                      VMState
	Resources                  Resources
	SizeGB                     int
	IP, Network, Guest, Detail string
	// value: the state its record asks for, running or stopped.
	Want string
	// value: a server's health check; empty for none.
	Check string
	// value: a server's own tailnet device, by its name on the tailnet,
	// which the owner's tooling installed; empty for none.
	Device    string
	CreatedAt time.Time
	// value: its ssh connection opened, whether station's guest answers on
	// it or not (a server's tooling may replace it).
	SSH bool
	// value: a server's restarts since its Mac's engine started, and its
	// health check as last run.
	Restarts    Restarts
	CheckStatus CheckStatus
	// value: when its engine last booted it, while it runs; zero when not
	// known.
	BootAt time.Time
	// value: a server's record of the last 30 days, its check's last runs,
	// and the step its restart runs now (nil when none runs).
	Uptime  Uptime
	Latency []LatencyPoint
	Restart *RestartWalk
	// value: SPEC B5.5: why a server is down, and the last lines its
	// console kept then; empty while it is up.
	Cause     string
	LastLines []string
	// value: SPEC B5.10: how far its create is while it runs, and when its
	// create made it ready.
	Birth   *Birth
	ReadyAt time.Time
	// value: the agent a workspace is said by, as its guest last reported
	// it; nil while none is known.
	Agent *AgentView
	// value: the agents a workspace held when it last stopped, which a
	// stopped one or one of a silent Mac still holds.
	Agents []string `json:",omitempty"`
}

type AgentView struct {
	State string
	Task  string
	// value: what it waits on the owner for, and since when.
	Ask   string
	Since time.Time
	// value: when it finished, and how, in its app's words.
	Ended  time.Time
	Reason string
}

const AgentFinished = "finished"

func (vm VM) Finished() bool { return vm.Agent != nil && vm.Agent.State == AgentFinished }

type EngineView struct {
	Known, Running bool
	// value: the agent's plist is on this host: a daemon not running is
	// stopped, not missing (R9.D3); unknown for another host.
	Installed bool
	Version   string
}

type MacView struct {
	Name, User, StationVersion string
	ThisMac, Reachable         bool
	// value: when this Mac last had its answer; zero for one never heard.
	Answered time.Time
	// value: its address on your LAN, as it reads it; empty when it cannot say.
	LAN    string
	Node   Node
	Engine EngineView
	VMs    []VM
	// value: SPEC B5.12: what it is, as it read itself at its last answer,
	// and whether it has a lid (a laptop) or not (a desktop).
	Hardware machine.Hardware
	Laptop   bool
	// value: SPEC B5.15: at its last answer it was on battery with its lid
	// closed, so its silence since is sleep.
	Asleep bool
}

type AllMachines struct {
	Macs []MacView
	// value: when it was read.
	Read time.Time
}

type CreateRequest struct {
	Image, Name string
	// value: a workspace's, throwaway or permanent; a server's is always on.
	Lifetime Lifetime
	Network  string
	Size     Resources
	// value: the create makes a server, which is always on.
	Server bool
	// value: a server's health check, a URL or host:port; empty for none.
	Check string
}

type AgentNotStartedError struct {
	// value: a workspace made and answering, whose agent did not start.
	Name  string
	Cause error
}

func (e *AgentNotStartedError) Error() string {
	return e.Name + " is ready, but its agent did not start: " + e.Cause.Error()
}

type ServerLifetimeError struct {
	// value: a server create that also asks a workspace's lifetime, by its
	// flag.
	Name, Flag string
}

func (e *ServerLifetimeError) Error() string {
	return "a server is always on: --" + e.Flag + " is for workspaces"
}

func ServerSize() Resources {
	// value: a server's size when the create names none (SPEC §7.1); a
	// workspace's is its image's.
	return Resources{CPUs: 4, MemoryBytes: 8 << 30, DiskBytes: 50e9}
}

type Created struct {
	VM    VM
	Ports []Reserved
	Env   map[string]string
	// value: the Mac it was made on reports a battery: a server there stops
	// whenever the Mac sleeps.
	OnALaptop bool
}

type Step struct{ Label string }

// value: SPEC §7.2's health steps, in order, a workspace's three.
const (
	StepRunning = "Running"
	StepAddress = "Has an address"
	StepSSH     = "SSH answers"
)

type DeleteReport struct {
	Name string
	// value: SPEC B5.22: the agents it hosted, stopped before it went.
	Agents   []string
	Warnings []string
}

type RestartEvent struct{ Phase, Detail string }

type RestartReport struct {
	Name string
	// value: a VM was restarted, its health steps passed; IP is its address.
	VM     bool
	IP     string
	Waited time.Duration
}

type Change struct {
	// value: the sizes an edit changes; a zero one is kept.
	Resources Resources
	// value: a server's health check to set, or NoCheck to remove it.
	Check   string
	NoCheck bool
}

func (c Change) None() bool { return c == Change{} }

type Edited struct {
	Name string
	// value: the sizes the edit asked, the others zero.
	Changed Resources
	// value: the health check it set, or that it removed one.
	Check   string
	NoCheck bool
	// value: the VM runs, so the change waits for its next start.
	Running bool
}

func FindVM(f AllMachines, name string) (MacView, VM, bool) {
	for _, h := range f.Macs {
		for _, vm := range h.VMs {
			if vm.Name == name {
				return h, vm, true
			}
		}
	}
	return MacView{}, VM{}, false
}
