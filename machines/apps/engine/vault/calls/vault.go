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
		for i, e := range read.Entries {
			read.Entries[i] = accounts.DescribeEntry(e)
		}
		read.Verdict = accounts.SummarizeVault(read.Entries)
		out := make(chan accounts.VaultRead, 1)
		out <- read
		close(out)
		return call.Updates(ctx, out), nil
	},
}

type VaultNewIn struct{}

var VaultNew = call.Watch[Deps, VaultNewIn, accounts.VaultOffer]{
	// invariant: the sheet of a password is read from this Mac's sign-ins:
	// each program it holds and whose token can be taken is a capture, and
	// every program says which kinds a typed password of it may be.
	Name: "vault.new", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Run: func(ctx context.Context, d Deps, _ VaultNewIn, _ call.Progress) (<-chan call.Update[accounts.VaultOffer], error) {
		read, err := d.SignIns.Read(ctx, "")
		if err != nil {
			return nil, err
		}
		out := make(chan accounts.VaultOffer, 1)
		out <- accounts.NewPassword(d.Mac, d.Separator, read.Programs)
		close(out)
		return call.Updates(ctx, out), nil
	},
}

type VaultAddIn struct {
	Label   string
	Program accounts.Program
	// value: the kind of a typed password; a password taken from a sign-in
	// is of the kind its program holds, and sends none.
	Kind accounts.KeyKind
	From accounts.AddFrom
	// value: the typed secret; a password taken from a sign-in sends none.
	Value secret.Sent
}

var VaultAdd = call.Call[Deps, VaultAddIn, accounts.Added]{
	Name: "vault.add", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.VaultEmptyLabel, accounts.VaultLabelTaken, accounts.VaultEmptyValue, accounts.VaultNotSignedIn, accounts.VaultCannotCapture},
	Run: func(ctx context.Context, d Deps, in VaultAddIn, _ call.Progress) (accounts.Added, error) {
		// invariant: a password has a label, a typed one a kind and a value,
		// a password from a sign-in the kind its program holds; a refusal
		// about an input names it.
		if accounts.BlankLabel(in.Label) {
			return accounts.Added{}, accounts.Fielded(accounts.Refusal(fail.Refused, accounts.VaultEmptyLabel, in.Label))
		}
		if in.From == accounts.FromTyped && in.Kind == "" {
			return accounts.Added{}, fail.New(fail.Refused, call.BadInput, "vault.add", map[string]string{"in": "Kind"})
		}
		if in.From == accounts.FromTyped && len(in.Value.Reveal()) == 0 {
			return accounts.Added{}, accounts.Fielded(accounts.Refusal(fail.Refused, accounts.VaultEmptyValue, in.Label))
		}
		if in.From == accounts.FromSignIn {
			in.Kind = in.Program.CaptureKind()
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
		if err := confirm.Require(confirm.Of(confirm.RemovePassword, d.Kind), in.ID, in.Confirm); err != nil {
			return accounts.Removed{}, err
		}
		ended, err := d.Passwords.Remove(ctx, in.ID)
		return accounts.Removed{Removed: ended}, err
	},
}

type Deps struct {
	Passwords contract.Passwords
	SignIns   contract.SignIns
	// value: this Mac's name, whose Keychain keeps the passwords.
	Mac string
	// value: the glyph between a program and who a proposed label names.
	Separator string
	Kind      machine.Kind
}

func All(d Deps) []call.Handler {
	return []call.Handler{call.BindWatch(VaultList, d), call.BindWatch(VaultNew, d), call.Bind(VaultAdd, d), call.Bind(VaultRemove, d)}
}
