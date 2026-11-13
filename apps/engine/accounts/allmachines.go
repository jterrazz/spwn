package accounts

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

const ShortShare = 30 * time.Minute // value: a share with less time left than this needs the owner (SPEC B5.8).

type MacAccounts struct {
	Name    string
	ThisMac bool
	// value: why the Mac's engine gave no reading; empty once it answered.
	Silent   string
	Accounts []Account
	Shares   []Shared
}

type AllMachines struct {
	Now  time.Time
	Macs []MacAccounts
}

type SignIn struct {
	Mac      string
	ThisMac  bool
	Provider string
	State    catalog.State
	Reason   string
	Hint     string
	Words    []string
}

type SignedInOn struct {
	Mac      string
	ThisMac  bool
	Accounts []SignIn
}

type SharedCopy struct {
	Mac, VM, Provider string
	// value: what the copy lets the VM do, in the catalogue's words.
	Scope       string
	Left, Total time.Duration
	Short       bool
}

type Silent struct{ Mac, Why string }

type Reading struct {
	SignedOut []SignIn
	SignedIn  []SignedInOn
	Shared    []SharedCopy
	Silent    []Silent
}

func (a AllMachines) Read() Reading {
	// invariant: SPEC B5.8: the sign-ins every answering Mac is out of come
	// first, This Mac's first; then what each Mac is signed in to, a Mac with
	// none left out; then every copy shared with a VM, the one ending first
	// first. A provider that is no sign-in (git, ssh) is never listed.
	var r Reading
	for _, m := range a.Macs {
		if m.Silent != "" {
			r.Silent = append(r.Silent, Silent{Mac: m.Name, Why: m.Silent})
			continue
		}
		on := SignedInOn{Mac: m.Name, ThisMac: m.ThisMac}
		for _, acc := range m.Accounts {
			p, ok := provider(acc.Provider)
			if !ok || len(p.Words) == 0 {
				continue
			}
			s := SignIn{Mac: m.Name, ThisMac: m.ThisMac, Provider: acc.Provider, State: acc.State, Reason: acc.Reason, Hint: acc.Hint}
			switch acc.State {
			case catalog.StateOK:
				s.Words = Words(acc)
				on.Accounts = append(on.Accounts, s)
			case catalog.StateAbsent, catalog.StateBroken:
				r.SignedOut = append(r.SignedOut, s)
			case catalog.StateDegraded, catalog.StateUnknown, catalog.StatePending:
			}
		}
		if len(on.Accounts) > 0 {
			r.SignedIn = append(r.SignedIn, on)
		}
		for _, sh := range m.Shares {
			r.Shared = append(r.Shared, SharedCopy{
				Mac: m.Name, VM: sh.VM, Provider: sh.Account, Scope: scopeOf(sh.Account),
				Left: sh.Until.Sub(a.Now), Total: sh.Until.Sub(sh.At), Short: sh.Until.Sub(a.Now) < ShortShare,
			})
		}
	}
	slices.SortStableFunc(r.Shared, func(x, y SharedCopy) int { return cmp.Compare(x.Left, y.Left) })
	return r
}

func (a AllMachines) Of(mac, vm string) AllMachines {
	// invariant: one Mac's accounts and the copies it shared; with a VM named,
	// only the copies that Mac shared with that VM, its own sign-ins left out.
	out := AllMachines{Now: a.Now}
	for _, m := range a.Macs {
		if m.Name != mac {
			continue
		}
		if vm != "" {
			m.Accounts = nil
			m.Shares = slices.DeleteFunc(slices.Clone(m.Shares), func(sh Shared) bool { return sh.VM != vm })
		}
		out.Macs = append(out.Macs, m)
	}
	return out
}

func Words(a Account) []string {
	// invariant: SPEC B5.8: the words beside a signed-in account are what
	// its provider's read printed, part by part as the catalogue orders
	// them; a part the read left empty is omitted, never guessed.
	p, _ := provider(a.Provider)
	var out []string
	for _, part := range p.Words {
		switch part {
		case catalog.PartWho:
			for _, who := range a.Identity {
				if who != Unknown && who != "" {
					out = append(out, who)
				}
			}
		case catalog.PartScopes:
			if len(a.Scopes) > 0 {
				out = append(out, strings.Join(a.Scopes, ", "))
			}
		case catalog.PartPlan:
			if len(a.Scopes) > 0 && a.Scopes[0] != "" {
				out = append(out, cmp.Or(catalog.ClaudePlans[a.Scopes[0]], a.Scopes[0]))
			}
		}
	}
	return out
}

func provider(id string) (catalog.Provider, bool) {
	i := slices.IndexFunc(catalog.Providers, func(p catalog.Provider) bool { return p.ID == id })
	if i < 0 {
		return catalog.Provider{}, false
	}
	return catalog.Providers[i], true
}

func scopeOf(account string) string {
	for _, sc := range catalog.ShareScopes {
		if sc.Provider == account {
			return sc.Scope
		}
	}
	return ""
}
