package calls

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type ShareIn struct {
	Account string
	With    string
	For     time.Duration
	// value: the Mac the owner shares from, when a surface sent the call to
	// the VM's own Mac because it is another.
	From string
}

var Share = call.Call[Deps, ShareIn, accounts.Shared]{
	Name: "accounts.share", Section: section.Accounts, Where: call.InEngine, Applies: accounts.SharedFrom, Needs: accounts.ShareNeeds(),
	Errors: []fail.Code{Unknown, NotShareable, NotSignedIn, Unreadable, VMStopped, ToServer, OtherMac, UnknownVM, TermOutOfBounds},
	Run: func(ctx context.Context, d Deps, in ShareIn, _ call.Progress) (accounts.Shared, error) {
		// invariant: a term outside the span is refused before any read or
		// effect, whichever surface sent it.
		term, ok := accounts.CheckTerm(in.For)
		if !ok {
			return accounts.Shared{}, coded(&accounts.TermError{Account: in.Account, VM: in.With, Term: term})
		}
		if err := d.Shares.CheckFrom(ctx, in.Account, in.With, in.From); err != nil {
			return accounts.Shared{}, coded(err)
		}
		out, err := d.Shares.Share(ctx, in.Account, in.With, term)
		return out, coded(err)
	},
}

type TakeBackIn struct {
	Account string
	From    string
	Confirm string
}

var TakeBack = call.Call[Deps, TakeBackIn, accounts.Shared]{
	Name: "accounts.take-back", Section: section.Accounts, Where: call.InEngine, Applies: accounts.SharedFrom, Confirm: confirm.TakeBack,
	Errors: []fail.Code{Unknown, NotShared, UnknownVM, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in TakeBackIn, _ call.Progress) (accounts.Shared, error) {
		// invariant: a take-back asks first, since it deletes the VM's copy;
		// what cannot be taken back is refused before the question.
		if _, err := d.Shares.CheckTakeBack(ctx, in.Account, in.From); err != nil {
			return accounts.Shared{}, coded(err)
		}
		if err := confirm.Require(confirm.For(confirm.TakeBack, d.Kind), accounts.Label(in.Account)+" from "+in.From, in.Confirm); err != nil {
			return accounts.Shared{}, err
		}
		if err := d.Shares.TakeBack(ctx, in.Account, in.From); err != nil {
			return accounts.Shared{}, coded(err)
		}
		return accounts.Shared{Account: in.Account, VM: in.From}, nil
	},
}
