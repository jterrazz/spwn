package accounts

import (
	"slices"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
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
	// value: on a Mac, every share it holds with its VMs, and the names its
	// share list lets it copy.
	Shares    []Shared
	ShareList []string
}

func (r Report) Clone() Report {
	out := r
	out.Machines = make([]Machine, len(r.Machines))
	for i, m := range r.Machines {
		m.Accounts = slices.Clone(m.Accounts)
		out.Machines[i] = m
	}
	return out
}

func (r Report) Resolved() Report {
	// value: a shared copy is the Mac's right, so its identity is the Mac's
	// for that provider once the Mac's read has answered.
	if len(r.Machines) == 0 || r.Machines[0].Kind != KindComputer {
		return r
	}
	mac := map[string][]string{}
	for _, a := range r.Machines[0].Accounts {
		if a.State == catalog.StateOK {
			mac[a.Provider] = a.Identity
		}
	}
	out := r.Clone()
	for i := range out.Machines {
		for j, a := range out.Machines[i].Accounts {
			if a.Shared != nil && len(a.Identity) == 1 && a.Identity[0] == Unknown && mac[a.Provider] != nil {
				out.Machines[i].Accounts[j].Identity = mac[a.Provider]
			}
		}
	}
	return out
}

func RowID(machine, provider string) string { return machine + "/" + provider }

func Lendable(list []string, provider string) []string {
	var out []string
	for _, sc := range catalog.ShareScopes {
		if Shareable(sc.Name, list) && (provider == "" || sc.Provider == provider) {
			out = append(out, sc.Name)
		}
	}
	return out
}
