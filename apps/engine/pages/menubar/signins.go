package menubar

import (
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
)

type SignIns struct {
	// value: each machine's sign-ins and grants as accounts.list answers
	// them, each Mac then its VMs; This Mac's vault (vault.list) and, by
	// password, the machines grants.new says it may go to. A nil vault is
	// still being read, or did not answer.
	Machines []MachineSignIns
	Vault    *accounts.VaultRead
	Offers   map[string]accounts.GrantOffer
}

type MachineSignIns struct {
	// value: the machine read and the Mac it is on, itself for a Mac.
	Name, Mac string
	ThisMac   bool
	VM        bool
	// value: why it gave no reading; empty once it answered.
	Silent string
	Rows   []accounts.ProgramRow
}

func (s SignIns) within(mac, vm string) SignIns {
	// invariant: one Mac's sign-ins and its VMs', or one VM's alone; the
	// vault and its offers stay, since a grant is made from This Mac.
	out := SignIns{Vault: s.Vault, Offers: s.Offers}
	for _, m := range s.Machines {
		if m.Mac == mac && (vm == "" || m.Name == vm) {
			out.Machines = append(out.Machines, m)
		}
	}
	return out
}

func (s SignIns) grants() int {
	n := 0
	for _, m := range s.Machines {
		for _, r := range m.Rows {
			if r.How == accounts.HowGranted {
				n++
			}
		}
	}
	return n
}

func (s SignIns) signedIn() int {
	n := 0
	for _, m := range s.Machines {
		for _, r := range m.Rows {
			if r.State == accounts.ProgramSignedIn && r.How != accounts.HowGranted {
				n++
			}
		}
	}
	return n
}

func (s SignIns) machine(name string) (MachineSignIns, bool) {
	i := slices.IndexFunc(s.Machines, func(m MachineSignIns) bool { return m.Name == name })
	if i < 0 {
		return MachineSignIns{}, false
	}
	return s.Machines[i], true
}

// value: the machines a password may be granted to, in the order
// grants.new lists them, each once over every password: the targets of
// the sheet's To.
func (s SignIns) targets() []string {
	var out []string
	if s.Vault == nil {
		return nil
	}
	for _, e := range s.Vault.Entries {
		for _, t := range s.Offers[e.ID].Targets {
			if t.Fit == accounts.TargetFits && !slices.Contains(out, t.To) {
				out = append(out, t.To)
			}
		}
	}
	return out
}

// value: whether the password may go to the machine, and the engine's
// line for it there.
func (s SignIns) fit(entry, to string) (fits bool, line string) {
	for _, t := range s.Offers[entry].Targets {
		if t.To == to {
			return t.Fit == accounts.TargetFits, t.Line
		}
	}
	return false, ""
}

func (s SignIns) presence() string {
	for _, o := range s.Offers {
		if o.Presence != "" {
			return o.Presence
		}
	}
	return ""
}

// value: what a sign-in row says under its program: who, its scopes and
// plan for its own sign-in, the password it came from for a grant.
func signInWords(r accounts.ProgramRow) []string {
	if r.How == accounts.HowGranted {
		label := r.EntryLabel
		if label == "" {
			label = r.Entry
		}
		return []string{"from the password " + label}
	}
	var out []string
	if r.Who != "" {
		out = append(out, r.Who)
	}
	if len(r.Scopes) > 0 {
		out = append(out, strings.Join(r.Scopes, ", "))
	}
	if r.Plan != "" {
		out = append(out, r.Plan)
	}
	return out
}

// value: a program's logo, drawn by the brand its label names.
func logo(p accounts.Program) page.Glyph {
	for _, provider := range catalog.Providers {
		if provider.Label == p.Label() {
			return page.Glyph{Kind: "account:" + provider.ID}
		}
	}
	return page.Glyph{Kind: "account:" + string(p)}
}
