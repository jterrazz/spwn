package ports

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type Phase string

const (
	PhaseNone    Phase = ""
	PhaseShared  Phase = "shared"
	PhaseClosing Phase = "closing"
	PhaseWaiting Phase = "waiting"
	PhaseRefused Phase = "refused"
	// value: a port that listens outside its machine's rights: not shared.
	PhaseNotAllowed Phase = "not-allowed"
	PhaseGone       Phase = "gone"
	PhaseUnknown    Phase = "unknown"
)

type URLSet struct {
	Local  string `json:"local"`
	Public string `json:"public"`
}

type Target struct {
	VM   string
	Port int
}

type SharedPort struct {
	VM        string `json:"vm"`
	GuestPort int    `json:"guest_port"`
	Reserved  int    `json:"reserved"`
	State     Phase  `json:"state"`
	Process   string `json:"process"`
	HTTP      bool   `json:"http"`
	// value: station.ports.list/v1's pinned, kept for its readers: always
	// false, no reservation is pinned since ports are rights.
	Pinned bool `json:"pinned"`
	// value: station.ports.list/v1's profile: the catalog package whose env a
	// share of this port sets (`vite`), empty for any other.
	Profile string `json:"profile"`
	// value: the label the catalog's packages give this port ("Vite"), a hint
	// of what probably serves it, never a decision.
	Label string `json:"label,omitempty"`
	// value: who it is shared with: local, private or public.
	To   catalog.Audience `json:"to,omitempty"`
	URLs URLSet           `json:"urls"`
	// value: the address it listens on inside its VM, as its guest reports it.
	Bind       string     `json:"bind"`
	LastSeenAt *time.Time `json:"last_seen_at"`
	Detail     string     `json:"detail"`
	// value: when the owner shared it: a share of the last few seconds is
	// fresh on the panel.
	ReservedAt time.Time `json:"reserved_at,omitzero"`
}

type EngineInfo struct {
	Running bool   `json:"running"`
	Version string `json:"version"`
	// value: the agent's plist is on this Mac: an engine not running is
	// stopped, not missing (R9.D3); never on the wire.
	Installed bool `json:"-"`
}

type MacPorts struct {
	Name       string       `json:"name"`
	ThisMac    bool         `json:"this_mac"`
	Reachable  bool         `json:"reachable"`
	Engine     EngineInfo   `json:"engine"`
	PublicName string       `json:"public_name"`
	Serve      string       `json:"serve"`
	HTTPS      string       `json:"https"`
	Range      *Range       `json:"range,omitempty"`
	Shared     []SharedPort `json:"shared"`
	// value: the listeners of a VM that no reservation holds and a refusal
	// kept from sharing (the VM at its most, the range taken), one count
	// per VM: they are never told port by port.
	NotShared []NotShared `json:"not_shared,omitempty"`
	// value: the listeners of each VM on ports the system keeps for itself,
	// present now, one count per VM: never shared by themselves, never told
	// port by port.
	Quiet []Quiet `json:"quiet,omitempty"`
	// value: what listens on each VM of this Mac, as its engine heard it.
	Listening []Machine `json:"listening,omitempty"`
	// value: every port this Mac and its VMs listen on, in its two classes.
	Ports []Port `json:"ports,omitempty"`
}

type NotShared struct {
	VM     string `json:"vm"`
	Count  int    `json:"count"`
	Detail string `json:"detail"`
}

type Quiet struct {
	VM    string `json:"vm"`
	Count int    `json:"count"`
}

type Range struct {
	Low      int `json:"low"`
	High     int `json:"high"`
	Reserved int `json:"reserved"`
	Free     int `json:"free"`
	MaxPerVM int `json:"max_per_vm"`
}

const AllMachinesSchema = "station.ports.all/v1"

type AllMachines struct {
	Schema string     `json:"schema"`
	Macs   []MacPorts `json:"macs"`
}

type Listing struct {
	AllMachines AllMachines `json:"all"`
	// value: this Mac's node on the tailnet, zero inside a VM: its Mac's
	// sharing is said by each of its rows.
	Tailnet Status `json:"tailnet"`
}
