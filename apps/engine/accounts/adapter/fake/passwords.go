package fake

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

const endingWithin = 15 * time.Minute

type Passwords struct {
	w     *world.World
	mac   string
	clock Clock
}

var _ contract.Passwords = (*Passwords)(nil)

func NewPasswords(w *world.World, mac string, clock Clock) *Passwords {
	return &Passwords{w: w, mac: mac, clock: clock}
}

func (p *Passwords) own() *world.Accounts { return state(mac(p.w, p.mac)) }

func (p *Passwords) List(context.Context) (accounts.VaultRead, error) {
	now := p.clock.Now()
	p.w.Lock()
	defer p.w.Unlock()
	entries := make([]accounts.VaultEntry, 0, len(p.own().Passwords))
	for _, e := range p.own().Passwords {
		e.Grants = slices.Clone(e.Grants)
		if e.Grants == nil {
			e.Grants = []accounts.VaultGrant{}
		}
		for i := range e.Grants {
			if !e.Grants[i].Until.IsZero() && e.Grants[i].Until.Sub(now) <= endingWithin {
				e.Grants[i].State = accounts.GrantEnding
			}
		}
		entries = append(entries, e)
	}
	return accounts.VaultRead{Mac: p.mac, Entries: entries}, nil
}

func (p *Passwords) find(id string) (int, bool) {
	i := slices.IndexFunc(p.own().Passwords, func(e accounts.VaultEntry) bool { return e.ID == id })
	return i, i >= 0
}

func (p *Passwords) Check(_ context.Context, id string) error {
	p.w.Lock()
	defer p.w.Unlock()
	if _, ok := p.find(id); !ok {
		return accounts.Refusal(fail.NotFound, accounts.VaultUnknown, id)
	}
	return nil
}

func (p *Passwords) Add(_ context.Context, in contract.PasswordIn) (string, error) {
	added := p.clock.Now()
	p.w.Lock()
	defer p.w.Unlock()
	own := p.own()
	for _, e := range own.Passwords {
		if e.Label == in.Label {
			return "", accounts.Refusal(fail.Refused, accounts.VaultLabelTaken, in.Label)
		}
	}
	who := ""
	if in.From == accounts.FromSignIn {
		facts, signed := own.Programs[in.Program]
		if !signed || facts.State != accounts.ProgramSignedIn {
			return "", accounts.Refusal(fail.Refused, accounts.VaultNotSignedIn, string(in.Program))
		}
		if !in.Program.Captures() {
			return "", accounts.Refusal(fail.Failed, accounts.VaultCannotCapture, string(in.Program))
		}
		who = facts.Who
	}
	own.Counter++
	id := "pw-" + strconv.Itoa(own.Counter)
	entry := accounts.VaultEntry{ID: id, Label: in.Label, Program: in.Program, Kind: in.Kind, Who: who, Added: added, AgentGrantable: in.Kind == accounts.KeyFineGrained}
	own.Passwords = append(own.Passwords, entry)
	return id, nil
}

func (p *Passwords) Remove(_ context.Context, id string) ([]accounts.VaultGrant, error) {
	p.w.Lock()
	defer p.w.Unlock()
	i, ok := p.find(id)
	if !ok {
		return nil, accounts.Refusal(fail.NotFound, accounts.VaultUnknown, id)
	}
	entry := p.own().Passwords[i]
	for _, g := range entry.Grants {
		if g.ToKind == accounts.ToMachine {
			delete(state(mac(p.w, g.To)).Programs, entry.Program)
		}
	}
	p.own().Passwords = slices.Delete(p.own().Passwords, i, i+1)
	return slices.Clone(entry.Grants), nil
}
