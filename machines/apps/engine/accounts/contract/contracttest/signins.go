package contracttest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

type SignInsSubject struct {
	SignIns contract.SignIns
	// value: a program signed in on this machine, and one signed out whose
	// login gives a device code.
	SignedIn, SignedOut accounts.Program
}

func codeOf(err error) fail.Code {
	var f *fail.Error
	if errors.As(err, &f) {
		return f.Code
	}
	return ""
}

func rowOf(rows []accounts.ProgramRow, p accounts.Program) accounts.ProgramRow {
	for _, r := range rows {
		if r.Program == p {
			return r
		}
	}
	return accounts.ProgramRow{}
}

func SignIns(t *testing.T, subject func(t *testing.T) SignInsSubject) {
	t.Run("this machine reads a row per program, each with the action it would accept", func(t *testing.T) {
		s := subject(t)
		read, err := s.SignIns.Read(t.Context(), "")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if len(read.Programs) != len(accounts.Programs()) {
			t.Fatalf("Read gave %d rows; want one per program, %d", len(read.Programs), len(accounts.Programs()))
		}
		if in := rowOf(read.Programs, s.SignedIn); in.State != accounts.ProgramSignedIn || in.Action != accounts.ActionSignOut {
			t.Errorf("%s = %+v; want signed in, to sign out", s.SignedIn, in)
		}
		if out := rowOf(read.Programs, s.SignedOut); out.State != accounts.ProgramSignedOut || out.Action != accounts.ActionSignIn {
			t.Errorf("%s = %+v; want signed out, to sign in", s.SignedOut, out)
		}
	})

	t.Run("a program signed in already is refused before any step", func(t *testing.T) {
		s := subject(t)
		if _, err := s.SignIns.Login(t.Context(), "", s.SignedIn); codeOf(err) != accounts.LoginAlreadySignedIn {
			t.Errorf("Login = %v; want %s", err, accounts.LoginAlreadySignedIn)
		}
	})

	t.Run("a device code is sent as its step, its site opened, and its end closes the exchange", func(t *testing.T) {
		s := subject(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		updates, err := s.SignIns.Login(ctx, "", s.SignedOut)
		if err != nil {
			t.Fatalf("Login: %v", err)
		}
		bound, stop := context.WithTimeout(t.Context(), 10*time.Second)
		defer stop()
		var first contract.LoginUpdate
		select {
		case first = <-updates:
		case <-bound.Done():
			t.Fatal("no step came")
		}
		step := first.Read.Step
		if first.Err != nil || step == nil {
			t.Fatalf("first update = %+v; want the step", first)
		}
		if step.Kind != accounts.StepDeviceCode || step.Code == "" || step.Site != s.SignedOut.Site() || step.Exchange == "" || !step.Expires.After(step.Now) || step.Line == "" {
			t.Errorf("step = %+v; want a device code to enter at %s, before it expires", step, s.SignedOut.Site())
		}
		if site, err := s.SignIns.Open(t.Context(), "", step.Exchange); err != nil || site != s.SignedOut.Site() {
			t.Errorf("Open = %q, %v; want the site %s", site, err, s.SignedOut.Site())
		}
		if err := s.SignIns.Answer(t.Context(), "", step.Exchange, secret.Sent{Value: secret.Hold("", []byte("FAKE000"))}); codeOf(err) != accounts.LoginNotAsking {
			t.Errorf("Answer to a device code = %v; want %s", err, accounts.LoginNotAsking)
		}
		cancel()
		for {
			select {
			case _, open := <-updates:
				if !open {
					return
				}
			case <-bound.Done():
				t.Fatal("the exchange did not end with its context")
			}
		}
	})

	t.Run("an exchange no login holds is refused", func(t *testing.T) {
		s := subject(t)
		if _, err := s.SignIns.Open(t.Context(), "", "ex-none"); codeOf(err) != accounts.LoginNoExchange {
			t.Errorf("Open = %v; want %s", err, accounts.LoginNoExchange)
		}
		if err := s.SignIns.Answer(t.Context(), "", "ex-none", secret.Sent{Value: secret.Hold("", []byte("FAKE000"))}); codeOf(err) != accounts.LoginNoExchange {
			t.Errorf("Answer = %v; want %s", err, accounts.LoginNoExchange)
		}
	})
}
