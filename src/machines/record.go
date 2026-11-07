package machines

import (
	"time"

	"github.com/jterrazz/studio-station/src/core/machine"
)

type Record struct {
	// value: the declared VM as its Mac keeps it; never observed state.
	Name, Image, Version string
	Lifetime             Lifetime
	Want, Network        string
	// value: a server's health check, a URL or host:port; empty for none.
	Check string
	// value: a workspace's task, what its agent was started on; empty for
	// none.
	Task        string
	Env         map[string]string
	CPUs        int
	MemoryBytes int64
	DiskBytes   int64
	CreatedAt   time.Time
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
}

type SeenVM struct {
	Name        string   `json:"name"`
	Lifetime    Lifetime `json:"lifetime"`
	CPUs        int      `json:"cpus"`
	MemoryBytes int64    `json:"memory_bytes"`
}
