package calls

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
)

type ReaderOptions struct {
	ThisMachine catalog.MachineKind
	// value: in a VM, the guest's hello keys: who may ssh in.
	Keys func() []string
}

type Reader struct {
	ids contract.Identities
	o   ReaderOptions
}

func NewReader(ids contract.Identities, o ReaderOptions) *Reader {
	return &Reader{ids: ids, o: o}
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
