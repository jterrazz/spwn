package calls

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/accounts"
	"github.com/jterrazz/studio-station/src/accounts/contract"
	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call/lanes"
)

const (
	expiringWithin  = time.Hour
	engineBudget    = 13 * time.Second
	providerReserve = 500 * time.Millisecond
)

type ReaderPorts struct {
	VMs       func(ctx context.Context) ([]accounts.VM, error)
	Engine    func(ctx context.Context) ([]accounts.VMRead, string)
	ShareList func() []string
}

type ReaderOptions struct {
	ThisMachine catalog.MachineKind
	ThisMacName string
	// value: in a VM, the guest's hello keys: who may ssh in.
	Keys func() []string
	Now  func() time.Time
}

type Reader struct {
	pool *lanes.Pool
	ids  contract.Identities
	p    ReaderPorts
	o    ReaderOptions
}

func NewReader(pool *lanes.Pool, ids contract.Identities, p ReaderPorts, o ReaderOptions) *Reader {
	return &Reader{pool: pool, ids: ids, p: p, o: o}
}

func (s *Reader) where() string {
	if s.o.ThisMachine == catalog.Mac {
		return accounts.KindComputer
	}
	return accounts.KindVM
}

func (s *Reader) providers() []catalog.Provider {
	var out []catalog.Provider
	for _, p := range catalog.Providers {
		if s.where() == accounts.KindComputer && !p.Mac || s.where() == accounts.KindVM && !p.VM {
			continue
		}
		if !s.ids.Offers(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (s *Reader) Own(ctx context.Context) []accounts.Account {
	ps := s.providers()
	out := make([]accounts.Account, len(ps))
	var wg sync.WaitGroup
	for i, p := range ps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out[i] = s.read(ctx, p)
		}()
	}
	wg.Wait()
	return out
}

func (s *Reader) read(ctx context.Context, p catalog.Provider) accounts.Account {
	var acc accounts.Account
	switch p.ID {
	case catalog.ProviderEntry:
		acc = s.entry()
	default:
		id, err := s.ids.Read(ctx, p)
		if err != nil {
			id = accounts.Broken(p.ID, accounts.ReasonFailing, err.Error())
		}
		acc = own(id)
	}
	return accounts.RedactAccount(acc)
}

func (s *Reader) entry() accounts.Account {
	var keys []string
	if s.o.Keys != nil {
		keys = s.o.Keys()
	}
	acc := ok(catalog.ProviderEntry)
	n := 0
	for _, k := range keys {
		fields := strings.Fields(k)
		if len(fields) < 2 {
			continue
		}
		n++
		if comment := strings.Join(fields[2:], " "); comment != "" && !slices.Contains(acc.Scopes, comment) {
			acc.Scopes = append(acc.Scopes, comment)
		}
		if fp, _, found := accounts.Fingerprint(k); found && acc.Detail == "" {
			acc.Detail = fp
		}
	}
	if n == 0 {
		return absent(catalog.ProviderEntry, accounts.ReasonUnset, "no SSH key")
	}
	acc.Identity = []string{strconv.Itoa(n) + " SSH keys"}
	if n == 1 {
		acc.Identity = []string{"1 SSH key"}
	}
	return acc
}

func (s *Reader) Read(ctx context.Context, withVMs bool) <-chan accounts.Report {
	now := s.o.Now().UTC()
	thisMac := accounts.Machine{Name: s.o.ThisMacName, Kind: s.where(), ThisMac: true}
	ps := s.providers()
	for _, p := range ps {
		thisMac.Accounts = append(thisMac.Accounts, accounts.Account{Provider: p.ID, State: catalog.StatePending})
	}
	frame := accounts.Report{Machines: []accounts.Machine{thisMac}, Now: now}
	var vms []accounts.VM
	if s.where() == accounts.KindComputer && s.p.VMs != nil {
		vms, _ = s.p.VMs(ctx)
		slices.SortStableFunc(vms, func(a, b accounts.VM) int {
			if a.Running != b.Running {
				if a.Running {
					return -1
				}
				return 1
			}
			return 0
		})
		for _, v := range vms {
			frame.Machines = append(frame.Machines, s.vmFrame(v, now))
		}
	}
	jobs := make([]lanes.Job[accounts.Report], 0, len(ps)+1)
	for i, p := range ps {
		jobs = append(jobs, lanes.Job[accounts.Report]{
			Key: "accounts/" + p.ID, Lane: lanes.Commands, Budget: p.Budget + providerReserve,
			Do: func(ctx context.Context) func(*accounts.Report) {
				acc := withHint(s.read(ctx, p), "")
				return func(r *accounts.Report) { r.Machines[0].Accounts[i] = acc }
			},
			Late: func(r *accounts.Report) { r.Machines[0].Accounts[i] = withHint(own(accounts.Late(p)), "") },
		})
	}
	if withVMs && len(vms) > 0 && s.p.Engine != nil {
		jobs = append(jobs, lanes.Job[accounts.Report]{
			Key: "accounts/vms", Lane: lanes.Inline, Budget: engineBudget,
			Do: func(ctx context.Context) func(*accounts.Report) {
				reads, engine := s.p.Engine(ctx)
				return func(r *accounts.Report) { s.merge(r, vms, reads, engine) }
			},
			Late: func(r *accounts.Report) { s.merge(r, vms, nil, accounts.EngineRunning) },
		})
	}
	if len(jobs) == 0 {
		out := make(chan accounts.Report, 1)
		out <- frame
		close(out)
		return out
	}
	return lanes.Run(ctx, s.pool, frame, jobs, accounts.Report.Clone)
}

func (s *Reader) vmFrame(v accounts.VM, now time.Time) accounts.Machine {
	m := accounts.Machine{Name: v.Name, Kind: accounts.KindVM, Lifetime: v.Lifetime, Running: v.Running, Pending: v.Running}
	m.Accounts = sharedRows(v, nil, false, now)
	return m
}

func (s *Reader) merge(r *accounts.Report, vms []accounts.VM, reads []accounts.VMRead, engine string) {
	r.Engine = engine
	byName := map[string]accounts.VMRead{}
	for _, v := range reads {
		byName[v.Name] = v
	}
	for i := range r.Machines {
		m := &r.Machines[i]
		if m.Kind != accounts.KindVM || m.ThisMac {
			continue
		}
		var v accounts.VM
		for _, x := range vms {
			if x.Name == m.Name {
				v = x
			}
		}
		m.Pending = false
		read, asked := byName[m.Name]
		switch {
		case !v.Running:
			m.Guest = ""
		case asked:
			m.Guest = read.Guest
		default:
			m.Guest = accounts.GuestAbsent
		}
		m.Accounts = sharedRows(v, read.Accounts, m.Guest == accounts.GuestAbsent, r.Now)
	}
}

// invariant: a running VM whose guest is absent keeps its rows from its
// record: its lent rows whole, every other provider of a VM read `?`.
func sharedRows(v accounts.VM, own []accounts.Account, unreachable bool, now time.Time) []accounts.Account {
	var out []accounts.Account
	for _, p := range catalog.Providers {
		if !p.VM {
			continue
		}
		var acc *accounts.Account
		for i := range own {
			if own[i].Provider == p.ID {
				a := own[i]
				acc = &a
			}
		}
		share := shareFor(v.Shares, p.ID)
		switch {
		case acc == nil && share == nil && unreachable && !p.Optional:
			a := accounts.Account{Provider: p.ID, State: catalog.StateDegraded, Reason: accounts.ReasonUnreachable, Origin: accounts.OriginOwn, Identity: []string{accounts.Unknown}, Detail: "guest not connected"}
			acc = &a
		case acc == nil && share == nil:
			continue
		case acc == nil:
			a := accounts.Account{Provider: p.ID, State: catalog.StateOK, Origin: accounts.OriginShared, Identity: []string{accounts.Unknown}}
			acc = &a
		case share != nil && acc.State == catalog.StateOK:
			acc.Origin = accounts.OriginBoth
		case share != nil:
			acc.State, acc.Reason, acc.Origin, acc.Detail, acc.Identity = catalog.StateOK, "", accounts.OriginShared, "", []string{accounts.Unknown}
		}
		if share != nil {
			l := *share
			acc.Shared = &l
			acc.Scopes = append([]string{l.Name}, acc.Scopes...)
			switch left := l.Until.Sub(now); {
			case left <= 0:
				acc.State, acc.Reason, acc.Detail = catalog.StateBroken, accounts.ReasonFailing, "expired, not taken back"
			case left <= expiringWithin:
				acc.State, acc.Reason = catalog.StateDegraded, accounts.ReasonExpiring
			}
		}
		out = append(out, withHint(*acc, v.Name))
	}
	return out
}

func shareFor(shares []accounts.Loan, provider string) *accounts.Loan {
	for _, l := range shares {
		for _, sc := range catalog.ShareScopes {
			if sc.Name == l.Name && sc.Provider == provider {
				return &l
			}
		}
	}
	return nil
}

func (s *Reader) Forget() {
	if s.pool != nil {
		s.pool.Forget("accounts/")
	}
}

func (s *Reader) ShareList() []string {
	if s.p.ShareList == nil {
		return nil
	}
	return s.p.ShareList()
}

func withHint(a accounts.Account, vm string) accounts.Account {
	a.Hint = ""
	if a.Shared != nil {
		switch a.Reason {
		case accounts.ReasonExpiring:
			a.Hint = "station accounts share " + accounts.AccountOf(a.Shared.Name) + " --with " + vm
		case accounts.ReasonFailing:
			a.Hint = "station accounts take-back " + accounts.AccountOf(a.Shared.Name) + " --from " + vm
		}
		return a
	}
	for _, p := range catalog.Providers {
		if p.ID != a.Provider {
			continue
		}
		for _, h := range p.Hints {
			if h.Reason != a.Reason {
				continue
			}
			a.Hint = h.ThisMachine
			if vm != "" {
				a.Hint = strings.ReplaceAll(h.InVM, catalog.HintVM, vm)
			}
		}
	}
	return a
}

// invariant: a provider's own read is the owner's own right once it is
// signed in or failing; absent or late, it is no one's yet.
func own(id accounts.Identity) accounts.Account {
	a := accounts.Account{Provider: id.Provider, State: id.State, Reason: id.Reason, Identity: id.Who, Scopes: id.Scopes, Source: id.Source, Detail: id.Detail, Notes: id.Notes}
	if id.State == catalog.StateOK || id.State == catalog.StateBroken {
		a.Origin = accounts.OriginOwn
	}
	return a
}

func ok(provider string) accounts.Account { return own(accounts.SignedIn(provider)) }

func absent(provider, reason, detail string) accounts.Account {
	return own(accounts.Absent(provider, reason, detail))
}
