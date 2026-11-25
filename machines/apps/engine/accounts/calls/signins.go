package calls

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	"github.com/jterrazz/studio-station/apps/engine/core/confirm"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/machine"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
	"github.com/jterrazz/studio-station/apps/engine/core/section"
)

type ProgramsIn struct {
	// value: a VM of the target Mac whose programs are read through it; empty reads the target's own.
	Machine string
}

var Programs = call.Watch[Deps, ProgramsIn, accounts.ProgramsRead]{
	Name: "accounts.list", Section: section.Accounts, Where: call.AtTarget, Applies: accounts.Applies,
	Run: func(ctx context.Context, d Deps, in ProgramsIn, _ call.Progress) (<-chan call.Update[accounts.ProgramsRead], error) {
		read, err := d.SignIns.Read(ctx, in.Machine)
		if err != nil {
			return nil, err
		}
		out := make(chan accounts.ProgramsRead, 1)
		out <- read
		close(out)
		return call.Updates(ctx, out), nil
	},
}

type LoginIn struct {
	// value: a VM of this Mac signed in through it; empty signs in this machine.
	Machine string `json:",omitempty"`
	Program accounts.Program
}

var Login = call.Watch[Deps, LoginIn, accounts.LoginRead]{
	Name: "accounts.login", Section: section.Accounts, Where: call.InEngine, Applies: accounts.Applies,
	Errors: []fail.Code{accounts.LoginUnknownProgram, accounts.LoginByGrant, accounts.LoginNotInstalled, accounts.LoginAlreadySignedIn, accounts.LoginMachineStopped, accounts.LoginBusy, accounts.LoginSiteRefused, accounts.LoginExpired, accounts.LoginFailed, accounts.LoginOtherMac},
	Run: func(ctx context.Context, d Deps, in LoginIn, _ call.Progress) (<-chan call.Update[accounts.LoginRead], error) {
		// invariant: a sign-in runs in the engine of the machine's own Mac,
		// which holds the program's run, its link and its input until the
		// exchange ends, so a later accounts.open or accounts.answer finds
		// them; a VM's own station answers a call at its exit, too late for
		// a step. A program station does not know, or one it signs in
		// only by a grant, is refused before anything runs. A server a Mac
		// holds keeps its own sign-ins: the section's rule refuses it here
		// as it does inside the server.
		if !in.Program.Known() {
			return nil, accounts.Refusal(fail.NotFound, accounts.LoginUnknownProgram, string(in.Program))
		}
		if !in.Program.SignsInItself() {
			return nil, accounts.Refusal(fail.Refused, accounts.LoginByGrant, string(in.Program))
		}
		if where, known := d.SignIns.Where(ctx, in.Machine); known && in.Machine != "" && where.FarMac == "" {
			if err := call.Gate(call.Meta{Name: "accounts.login", Section: section.Accounts, Applies: accounts.Applies}, machine.Profile{Ref: machine.Ref{Name: in.Machine}, Kind: where.Kind}); err != nil {
				return nil, err
			}
		}
		started, err := d.SignIns.Login(ctx, in.Machine, in.Program)
		if err != nil {
			return nil, err
		}
		out := make(chan call.Update[accounts.LoginRead])
		go func() {
			defer close(out)
			for u := range started {
				select {
				case out <- call.Update[accounts.LoginRead]{Value: u.Read, Err: u.Err}:
				case <-ctx.Done():
					return
				}
			}
		}()
		return out, nil
	},
}

type LogoutIn struct {
	// value: a VM of this Mac signed out through it; empty signs out this machine.
	Machine string `json:",omitempty"`
	Program accounts.Program
	Confirm string
}

var Logout = call.Call[Deps, LogoutIn, accounts.ProgramRow]{
	Name: "accounts.logout", Section: section.Accounts, Where: call.InEngine, Applies: accounts.Applies, Confirm: confirm.SignOut,
	Errors: []fail.Code{accounts.LoginUnknownProgram, accounts.LoginNotSignedIn, accounts.LoginGranted, accounts.LoginMachineStopped, accounts.LoginOtherMac, confirm.YesNeeded},
	Run: func(ctx context.Context, d Deps, in LogoutIn, _ call.Progress) (accounts.ProgramRow, error) {
		// invariant: a sign-out asks first, since the sign-in ends; what
		// cannot be signed out is refused before the question. It runs in
		// the engine of the machine's own Mac, as a sign-in does.
		if !in.Program.Known() {
			return accounts.ProgramRow{}, accounts.Refusal(fail.NotFound, accounts.LoginUnknownProgram, string(in.Program))
		}
		if err := d.SignIns.CheckSignOut(ctx, in.Machine, in.Program); err != nil {
			return accounts.ProgramRow{}, err
		}
		if err := confirm.Require(confirm.For(confirm.SignOut, d.Kind), in.Program.Label(), in.Confirm); err != nil {
			return accounts.ProgramRow{}, err
		}
		return d.SignIns.SignOut(ctx, in.Machine, in.Program)
	},
}

type OpenIn struct {
	Machine  string
	Exchange string
}

var Open = call.Call[Deps, OpenIn, accounts.Opened]{
	Name: "accounts.open", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.LoginNoExchange, accounts.LoginOtherMac, accounts.LoginExpired},
	Run: func(ctx context.Context, d Deps, in OpenIn, _ call.Progress) (accounts.Opened, error) {
		site, err := d.SignIns.Open(ctx, in.Machine, in.Exchange)
		return accounts.Opened{Site: site}, err
	},
}

type AnswerIn struct {
	Machine  string
	Exchange string
	Code     secret.Sent
}

var Answer = call.Call[Deps, AnswerIn, accounts.Answered]{
	Name: "accounts.answer", Section: section.Accounts, Where: call.InEngine, Applies: accounts.OnAMac,
	Errors: []fail.Code{accounts.LoginNoExchange, accounts.LoginNotAsking, accounts.LoginCodeRefused},
	Run: func(ctx context.Context, d Deps, in AnswerIn, _ call.Progress) (accounts.Answered, error) {
		return accounts.Answered{}, d.SignIns.Answer(ctx, in.Machine, in.Exchange, in.Code)
	},
}
