package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type GrantAddIn struct {
	Entry string
	To    string
}

var GrantAdd = call.Call[Deps, GrantAddIn, accounts.Granted]{
	Name: "grants.add", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.GrantUnknownEntry, accounts.GrantUnknownMachine, accounts.GrantToServer, accounts.GrantOtherMac, accounts.GrantMachineStopped, accounts.GrantSignedInHere, accounts.GrantPresenceRefused, accounts.GrantNoPresence, accounts.GrantCannotDeliver},
	Run: func(ctx context.Context, d Deps, in GrantAddIn, _ call.Progress) (accounts.Granted, error) {
		granted, err := d.Grants.Add(ctx, contract.GrantIn{Entry: in.Entry, To: in.To})
		return granted, accounts.Fielded(err)
	},
}

type GrantNewIn struct {
	Entry string
}

var GrantNew = call.Watch[Deps, GrantNewIn, accounts.GrantOffer]{
	// invariant: the sheet of a grant is read from this Mac's vault and
	// machines: each target says whether the password can go there and why
	// not when it cannot.
	Name: "grants.new", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.GrantUnknownEntry},
	Run: func(ctx context.Context, d Deps, in GrantNewIn, _ call.Progress) (<-chan call.Update[accounts.GrantOffer], error) {
		found, err := d.Grants.Candidates(ctx, in.Entry)
		if err != nil {
			return nil, err
		}
		out := make(chan accounts.GrantOffer, 1)
		out <- accounts.Offer(found)
		close(out)
		return call.Updates(ctx, out), nil
	},
}

type GrantRemoveIn struct {
	Entry   string
	To      string
	Confirm string
}

var GrantRemove = call.Call[Deps, GrantRemoveIn, accounts.Revoked]{
	Name: "grants.remove", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac, Confirm: confirm.RemoveGrant,
	Errors: []fail.Code{accounts.GrantNotGranted, accounts.GrantUnknownEntry, accounts.GrantUnknownMachine, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in GrantRemoveIn, _ call.Progress) (accounts.Revoked, error) {
		// invariant: a removal asks first, since the grant ends; what is not
		// granted is refused before the question.
		if err := d.Grants.Check(ctx, in.Entry, in.To); err != nil {
			return accounts.Revoked{}, err
		}
		if err := confirm.Require(confirm.For(confirm.RemoveGrant, d.Kind), in.To, in.Confirm); err != nil {
			return accounts.Revoked{}, err
		}
		return d.Grants.Remove(ctx, in.Entry, in.To)
	},
}

type Deps struct {
	Grants contract.Grants
	Kind   machine.Kind
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(GrantAdd, d), call.Bind(GrantRemove, d), call.BindWatch(GrantNew, d)}
}
