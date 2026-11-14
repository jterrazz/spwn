package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type GrantAddIn struct {
	Entry     string
	ToKind    accounts.GrantTo
	To        string
	Workspace string
	// value: how long the grant lasts; zero lasts until it is removed, a machine's alone.
	For time.Duration
}

var GrantAdd = call.Call[Deps, GrantAddIn, accounts.Granted]{
	Name: "grants.add", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.GrantUnknownEntry, accounts.GrantUnknownMachine, accounts.GrantUnknownAgent, accounts.GrantToServer, accounts.GrantOtherMac, accounts.GrantMachineStopped, accounts.GrantSignedInHere, accounts.GrantTermOutOfBounds, accounts.GrantNotAgentGrantable, accounts.GrantPresenceRefused, accounts.GrantNoPresence, accounts.GrantCannotDeliver},
	Run: func(ctx context.Context, d Deps, in GrantAddIn, _ call.Progress) (accounts.Granted, error) {
		// invariant: a term outside the bounds is refused before any read or
		// effect, whichever surface sent it.
		if !accounts.TermInBounds(in.ToKind, in.For) {
			e := accounts.Refusal(fail.Refused, accounts.GrantTermOutOfBounds, in.For.String())
			e.Detail = map[string]string{"to": string(in.ToKind)}
			return accounts.Granted{}, accounts.Fielded(e)
		}
		granted, err := d.Grants.Add(ctx, contract.GrantIn{Entry: in.Entry, ToKind: in.ToKind, To: in.To, Workspace: in.Workspace, For: in.For})
		return granted, accounts.Fielded(err)
	},
}

type GrantRemoveIn struct {
	Entry   string
	ToKind  accounts.GrantTo
	To      string
	Confirm string
}

var GrantRemove = call.Call[Deps, GrantRemoveIn, accounts.Revoked]{
	Name: "grants.remove", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac, Confirm: confirm.RemoveGrant,
	Errors: []fail.Code{accounts.GrantNotGranted, accounts.GrantUnknownEntry, accounts.GrantUnknownMachine, accounts.GrantUnknownAgent, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in GrantRemoveIn, _ call.Progress) (accounts.Revoked, error) {
		// invariant: a removal asks first, since the grant ends; what is not
		// granted is refused before the question.
		if err := d.Grants.Check(ctx, in.Entry, in.ToKind, in.To); err != nil {
			return accounts.Revoked{}, err
		}
		if err := confirm.Require(confirm.For(confirm.RemoveGrant, d.Kind), in.To, in.Confirm); err != nil {
			return accounts.Revoked{}, err
		}
		return d.Grants.Remove(ctx, in.Entry, in.ToKind, in.To)
	},
}

type Deps struct {
	Grants contract.Grants
	Kind   machine.Kind
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.Bind(GrantAdd, d), call.Bind(GrantRemove, d)}
}
