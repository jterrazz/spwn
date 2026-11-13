package tailscale

import (
	"context"
	"errors"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	portscontract "github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

const running = "Running"

type Identities struct {
	// value: the identities of every provider, the tailnet's account read
	// from the node itself: who is logged in, on which tailnet, with which
	// client.
	tailnet portscontract.Tailnet
	others  contract.Identities
}

var _ contract.Identities = (*Identities)(nil)

func New(tailnet portscontract.Tailnet, others contract.Identities) *Identities {
	return &Identities{tailnet: tailnet, others: others}
}

func (i *Identities) Offers(p catalog.Provider) bool { return i.others.Offers(p) }

func (i *Identities) Read(ctx context.Context, p catalog.Provider) (accounts.Identity, error) {
	if p.ID != catalog.ProviderTailscale {
		return i.others.Read(ctx, p)
	}
	self, err := i.tailnet.Self(ctx)
	return identity(p, self.Login, self.Network, self.Client.Variant, self.Sharing.Backend, err), nil
}

func identity(p catalog.Provider, login, network, variant, backend string, err error) accounts.Identity {
	var absent *portscontract.Absent
	switch {
	case errors.As(err, &absent):
		return accounts.Absent(p.ID, accounts.ReasonMissing, "not installed")
	case err != nil:
		return accounts.Broken(p.ID, accounts.ReasonFailing, "tailscale status did not answer")
	case backend != running:
		return accounts.Absent(p.ID, accounts.ReasonUnset, "not logged in")
	}
	id := accounts.SignedIn(p.ID)
	id.Who, id.Detail = []string{login}, variant
	if login == "" {
		id.Who = []string{accounts.Unknown}
	}
	if network != "" {
		id.Scopes = []string{network}
	}
	return id
}
