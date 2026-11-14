package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type VaultListIn struct{}

var VaultList = call.Watch[Deps, VaultListIn, accounts.VaultRead]{
	Name: "vault.list", Section: section.Accounts, Where: call.AtTarget, Applies: accounts.OnAMac,
	Run: func(ctx context.Context, d Deps, _ VaultListIn, _ call.Progress) (<-chan call.Update[accounts.VaultRead], error) {
		read, err := d.Passwords.List(ctx)
		if err != nil {
			return nil, err
		}
		out := make(chan accounts.VaultRead, 1)
		out <- read
		close(out)
		return call.Updates(ctx, out), nil
	},
}

type VaultAddIn struct {
	Label   string
	Program accounts.Program
	Kind    accounts.KeyKind
	From    accounts.AddFrom
	// value: the typed secret; a password taken from a sign-in sends none.
	Value secret.Sent
}

var VaultAdd = call.Call[Deps, VaultAddIn, accounts.Added]{
	Name: "vault.add", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.VaultLabelTaken, accounts.VaultEmptyValue, accounts.VaultNotSignedIn, accounts.VaultCannotCapture},
	Run: func(ctx context.Context, d Deps, in VaultAddIn, _ call.Progress) (accounts.Added, error) {
		// invariant: a typed password is never empty, and a refusal about an
		// input names it.
		if in.From == accounts.FromTyped && len(in.Value.Reveal()) == 0 {
			return accounts.Added{}, accounts.Fielded(accounts.Refusal(fail.Refused, accounts.VaultEmptyValue, in.Label))
		}
		id, err := d.Passwords.Add(ctx, contract.PasswordIn{Label: in.Label, Program: in.Program, Kind: in.Kind, From: in.From, Value: in.Value})
		return accounts.Added{ID: id}, accounts.Fielded(err)
	},
}

type VaultRemoveIn struct {
	ID      string
	Confirm string
}

var VaultRemove = call.Call[Deps, VaultRemoveIn, accounts.Removed]{
	Name: "vault.remove", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac, Confirm: confirm.RemovePassword,
	Errors: []fail.Code{accounts.VaultUnknown, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in VaultRemoveIn, _ call.Progress) (accounts.Removed, error) {
		// invariant: a removal asks first, since the grants made from the
		// password end with it; an unknown one is refused before the question.
		if err := d.Passwords.Check(ctx, in.ID); err != nil {
			return accounts.Removed{}, err
		}
		if err := confirm.Require(confirm.For(confirm.RemovePassword, d.Kind), in.ID, in.Confirm); err != nil {
			return accounts.Removed{}, err
		}
		ended, err := d.Passwords.Remove(ctx, in.ID)
		return accounts.Removed{Removed: ended}, err
	},
}

type Deps struct {
	Passwords contract.Passwords
	Kind      machine.Kind
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.BindWatch(VaultList, d), call.Bind(VaultAdd, d), call.Bind(VaultRemove, d)}
}
