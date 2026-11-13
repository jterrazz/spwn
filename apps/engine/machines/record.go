package machines

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
)

type Record struct {
	// value: the declared VM as its Mac keeps it; never observed state.
	Name, Image, Version string
	Lifetime             Lifetime
	Want, Network        string
	// value: a server's health check, a URL or host:port; empty for none.
	Check       string
	Env         map[string]string
	CPUs        int
	MemoryBytes int64
	DiskBytes   int64
	CreatedAt   time.Time
	// value: SPEC B5.10: its create's progress while it runs; nil after.
	Birth *Birth
	// value: the process its create runs in, while it runs; nil after.
	Owner *Process
	// value: the name of every agent a workspace held when it last stopped,
	// finished ones too: what it holds while no one reads its journal.
	Agents []string
	// value: the Machinefile it was made from (create --from); empty for
	// one made from an image alone.
	Template string
	// value: the folders of its Mac its last start shared: what a start
	// shares when its Machinefile cannot be read.
	Mounts []Mount
	// value: what it is for, as its create was told; the session that made
	// it, empty for the owner.
	Purpose, Creator string
}

type MacRecord struct {
	User string `json:"user,omitempty"`
	Key  string `json:"key,omitempty"`
	// value: when this Mac last had that Mac's answer, and the VMs it then
	// held: what All machines shows of it while it does not answer.
	Answered time.Time `json:"answered,omitzero"`
	VMs      []SeenVM  `json:"vms,omitempty"`
	// value: the address that Mac last answered with on your LAN: unlock
	// dials it, Tailscale being down until its disk is unlocked.
	LAN string `json:"lan,omitempty"`
	// value: what that Mac is, as it last answered: shown while it is silent.
	Hardware *machine.Hardware `json:"hardware,omitempty"`
	Laptop   bool              `json:"laptop,omitempty"`
	Asleep   bool              `json:"asleep,omitempty"`
	// value: SPEC B5.14: the owner asked to be told when it answers again;
	// the engine tells once and clears it.
	TellBack bool `json:"tell_back,omitempty"`
}

type SeenVM struct {
	Name        string              `json:"name"`
	Kind        catalog.MachineKind `json:"kind"`
	Lifetime    Lifetime            `json:"lifetime"`
	CPUs        int                 `json:"cpus"`
	MemoryBytes int64               `json:"memory_bytes"`
	// value: the agents a stopped workspace of it held, as that Mac last said.
	Agents []string `json:"agents,omitempty"`
}

const PurposeMax = 120 // value: the most characters a VM's purpose holds, on one line.

const ByOwner = "you"

func CreatorOf(session string) string {
	if session == "" {
		return ByOwner
	}
	return session
}
