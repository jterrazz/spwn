package ports

import "time"

type Phase string

const (
	PhaseNone    Phase = ""
	PhaseShared  Phase = "shared"
	PhaseClosing Phase = "closing"
	PhaseWaiting Phase = "waiting"
	PhaseRefused Phase = "refused"
	PhaseGone    Phase = "gone"
	PhaseUnknown Phase = "unknown"
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
	Pinned    bool   `json:"pinned"`
	Profile   string `json:"profile"`
	URLs      URLSet `json:"urls"`
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
	// value: what listens on each VM of this Mac, as its engine heard it.
	Listening []Machine `json:"listening,omitempty"`
	// value: every port this Mac and its VMs listen on, in its two classes.
	Ports []Port `json:"ports,omitempty"`
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
