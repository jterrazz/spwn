package fake

import (
	"cmp"
	"context"
	"maps"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Grants struct {
	w     *world.World
	mac   string
	clock Clock
	reads *SignIns
}

var _ contract.Grants = (*Grants)(nil)

func NewGrants(w *world.World, mac string, clock Clock) *Grants {
	return &Grants{w: w, mac: mac, clock: clock, reads: NewSignIns(w, mac, clock)}
}

func (g *Grants) entry(id string) (*accounts.VaultEntry, error) {
	own := state(mac(g.w, g.mac))
	i := slices.IndexFunc(own.Passwords, func(e accounts.VaultEntry) bool { return e.ID == id })
	if i < 0 {
		return nil, accounts.Refusal(fail.NotFound, accounts.GrantUnknownEntry, id)
	}
	return &own.Passwords[i], nil
}

func (g *Grants) target(in contract.GrantIn) error {
	stand, ok := g.reads.standOf(in.To)
	switch {
	case !ok:
		return accounts.Refusal(fail.NotFound, accounts.GrantUnknownMachine, in.To)
	case stand.Kind == machine.Server:
		return accounts.Refusal(fail.Refused, accounts.GrantToServer, in.To)
	case stand.FarMac != "":
		e := accounts.Refusal(fail.Refused, accounts.GrantOtherMac, in.To)
		e.Detail = map[string]string{"mac": stand.FarMac}
		return e
	case stand.Stopped:
		return accounts.Refusal(fail.Refused, accounts.GrantMachineStopped, in.To)
	}
	return nil
}

func (g *Grants) Check(_ context.Context, id, target string) error {
	g.w.Lock()
	defer g.w.Unlock()
	e, err := g.entry(id)
	if err != nil {
		return err
	}
	if _, ok := g.reads.standOf(target); !ok {
		return accounts.Refusal(fail.NotFound, accounts.GrantUnknownMachine, target)
	}
	if !slices.ContainsFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.To == target }) {
		return accounts.Refusal(fail.NotFound, accounts.GrantNotGranted, target)
	}
	return nil
}

func rank(name, first string) int {
	if name == first {
		return 0
	}
	return 1
}

func (g *Grants) Candidates(_ context.Context, id string) ([]accounts.Candidate, error) {
	g.w.Lock()
	defer g.w.Unlock()
	e, err := g.entry(id)
	if err != nil {
		return nil, err
	}
	found := []accounts.Candidate{}
	parents := slices.SortedFunc(maps.Keys(g.w.Macs), func(a, b string) int {
		return cmp.Or(cmp.Compare(rank(a, g.mac), rank(b, g.mac)), cmp.Compare(a, b))
	})
	for _, parent := range parents {
		for _, name := range slices.Sorted(maps.Keys(g.w.Macs[parent].VMs)) {
			vm := g.w.Macs[parent].VMs[name]
			if vm.Kept == nil {
				continue
			}
			stand, _ := g.reads.standOf(name)
			facts := state(mac(g.w, name)).Programs[e.Program]
			granted := slices.ContainsFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.To == name })
			found = append(found, accounts.Candidate{To: name, Where: stand, Facts: facts, Granted: granted})
		}
	}
	return found, nil
}

func (g *Grants) Add(_ context.Context, in contract.GrantIn) (accounts.Granted, error) {
	g.w.Lock()
	defer g.w.Unlock()
	e, err := g.entry(in.Entry)
	if err != nil {
		return accounts.Granted{}, err
	}
	if err := g.target(in); err != nil {
		return accounts.Granted{}, err
	}
	if state(mac(g.w, in.To)).Programs[e.Program].State == accounts.ProgramSignedIn && state(mac(g.w, in.To)).Programs[e.Program].How == accounts.HowOwn {
		return accounts.Granted{}, accounts.Refusal(fail.Refused, accounts.GrantSignedInHere, in.To)
	}
	own := state(mac(g.w, g.mac))
	switch mac(g.w, g.mac).Presence {
	case "refused":
		return accounts.Granted{}, accounts.Refusal(fail.Refused, accounts.GrantPresenceRefused, in.Entry)
	case "absent":
		return accounts.Granted{}, accounts.Refusal(fail.Failed, accounts.GrantNoPresence, in.Entry)
	}
	if slices.Contains(own.Undeliverable, in.To) {
		return accounts.Granted{}, accounts.Refusal(fail.Failed, accounts.GrantCannotDeliver, in.To)
	}
	e.Grants = slices.DeleteFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.To == in.To })
	e.Grants = append(e.Grants, accounts.VaultGrant{To: in.To})
	target := state(mac(g.w, in.To))
	if target.Programs == nil {
		target.Programs = map[accounts.Program]accounts.Facts{}
	}
	target.Programs[e.Program] = accounts.Facts{Program: e.Program, State: accounts.ProgramSignedIn, Who: e.Who, How: accounts.HowGranted, Entry: e.ID, EntryLabel: e.Label}
	return accounts.Granted{}, nil
}

func (g *Grants) Remove(_ context.Context, id, target string) (accounts.Revoked, error) {
	g.w.Lock()
	defer g.w.Unlock()
	e, err := g.entry(id)
	if err != nil {
		return accounts.Revoked{}, err
	}
	e.Grants = slices.DeleteFunc(e.Grants, func(v accounts.VaultGrant) bool { return v.To == target })
	delete(state(mac(g.w, target)).Programs, e.Program)
	return accounts.Revoked{Sentence: accounts.RevokedSentence(e.Program)}, nil
}

func (g *Grants) Forget(_ context.Context, vm string) error {
	// invariant: a machine deleted takes what was granted to it with its
	// disk; a world owes it no removal.
	g.w.Lock()
	defer g.w.Unlock()
	own := state(mac(g.w, g.mac))
	for i := range own.Passwords {
		own.Passwords[i].Grants = slices.DeleteFunc(own.Passwords[i].Grants, func(v accounts.VaultGrant) bool { return v.To == vm })
	}
	return nil
}
