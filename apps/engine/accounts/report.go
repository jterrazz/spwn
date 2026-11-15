package accounts

import (
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const (
	KindComputer = "computer"
	KindVM       = "vm"
)

type Account struct {
	Provider string        `json:"provider"`
	State    catalog.State `json:"state"`
	Reason   string        `json:"reason,omitempty"`
	Identity []string      `json:"identity,omitempty"`
	Origin   string        `json:"origin,omitempty"`
	Scopes   []string      `json:"scopes,omitempty"`
	Source   string        `json:"source,omitempty"`
	Detail   string        `json:"detail,omitempty"`
	Notes    []string      `json:"notes,omitempty"`
	Hint     string        `json:"hint,omitempty"`
	Shared   *Loan         `json:"shared,omitempty"`
	Redacted bool          `json:"-"`
}

type Loan struct {
	Name  string    `json:"name"`
	At    time.Time `json:"at"`
	Until time.Time `json:"until"`
}

type Machine struct {
	Name     string    `json:"name"`
	Kind     string    `json:"kind"`
	ThisMac  bool      `json:"this_mac,omitempty"`
	Lifetime string    `json:"lifetime,omitempty"`
	Guest    string    `json:"guest,omitempty"`
	Running  bool      `json:"-"`
	Pending  bool      `json:"-"`
	Accounts []Account `json:"accounts"`
}

type Report struct {
	Machines []Machine
	// value: the engine's state when this Mac's VMs could not be asked.
	Engine string
	Now    time.Time
}
