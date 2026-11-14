package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

type GrantsSubject struct {
	Grants    contract.Grants
	Passwords contract.Passwords
	// value: two VMs of this Mac, signed out of every program: one running, one stopped.
	Running, Stopped string
}

func Grants(t *testing.T, subject func(t *testing.T) GrantsSubject) {
	// value: the subject is an empty vault of a Mac whose presence check says yes.
	ctx := context.Background()
	entry := func(t *testing.T, s GrantsSubject) string {
		t.Helper()
		id, err := s.Passwords.Add(ctx, typed("GitHub deploy"))
		if err != nil {
			t.Fatalf("Add the password: %v", err)
		}
		return id
	}
	to := func(id, vm string) contract.GrantIn {
		return contract.GrantIn{Entry: id, ToKind: accounts.ToMachine, To: vm}
	}

	t.Run("a password the vault does not keep is an unknown entry", func(t *testing.T) {
		s := subject(t)
		if _, err := s.Grants.Add(ctx, to("pw-404", s.Running)); codeOf(err) != accounts.GrantUnknownEntry {
			t.Fatalf("Add = %v; want %s", err, accounts.GrantUnknownEntry)
		}
		if _, err := s.Grants.Candidates(ctx, "pw-404"); codeOf(err) != accounts.GrantUnknownEntry {
			t.Fatalf("Candidates = %v; want %s", err, accounts.GrantUnknownEntry)
		}
	})

	t.Run("a target that cannot take it is refused by its own code", func(t *testing.T) {
		s := subject(t)
		id := entry(t, s)
		for _, c := range []struct {
			in   contract.GrantIn
			want string
		}{
			{to(id, "nowhere"), string(accounts.GrantUnknownMachine)},
			{to(id, s.Stopped), string(accounts.GrantMachineStopped)},
			{contract.GrantIn{Entry: id, ToKind: accounts.ToAgent, To: "nobody", Workspace: s.Running}, string(accounts.GrantUnknownAgent)},
		} {
			if _, err := s.Grants.Add(ctx, c.in); string(codeOf(err)) != c.want {
				t.Errorf("Add to %s = %v; want %s", c.in.To, err, c.want)
			}
		}
	})

	t.Run("a grant is listed and is offered as granted", func(t *testing.T) {
		s := subject(t)
		id := entry(t, s)
		if _, err := s.Grants.Add(ctx, to(id, s.Running)); err != nil {
			t.Fatalf("Add: %v", err)
		}
		if refused := s.Grants.Check(ctx, id, accounts.ToMachine, s.Running); refused != nil {
			t.Fatalf("Check: %v", refused)
		}
		e, _ := listed(t, s.Passwords, id)
		if len(e.Grants) != 1 || e.Grants[0].To != s.Running {
			t.Fatalf("grants %+v; want the one grant to %s", e.Grants, s.Running)
		}
		found, err := s.Grants.Candidates(ctx, id)
		if err != nil {
			t.Fatalf("Candidates: %v", err)
		}
		offered := false
		for _, c := range found.List {
			offered = offered || c.ToKind == accounts.ToMachine && c.To == s.Running && c.Granted
		}
		if !offered || !found.AgentGrantable {
			t.Fatalf("Candidates = %+v; want %s granted, the password agent-grantable", found, s.Running)
		}
	})

	t.Run("a grant removed is no longer granted, and the sentence says where to revoke", func(t *testing.T) {
		s := subject(t)
		id := entry(t, s)
		if _, err := s.Grants.Add(ctx, to(id, s.Running)); err != nil {
			t.Fatalf("Add: %v", err)
		}
		revoked, err := s.Grants.Remove(ctx, id, accounts.ToMachine, s.Running)
		if err != nil || revoked.Sentence != accounts.RevokedSentence(capturing()) {
			t.Fatalf("Remove = %+v, %v", revoked, err)
		}
		if err := s.Grants.Check(ctx, id, accounts.ToMachine, s.Running); codeOf(err) != accounts.GrantNotGranted {
			t.Fatalf("Check = %v; want %s", err, accounts.GrantNotGranted)
		}
	})

	t.Run("a password removed ends its grants", func(t *testing.T) {
		s := subject(t)
		id := entry(t, s)
		if _, err := s.Grants.Add(ctx, to(id, s.Running)); err != nil {
			t.Fatalf("Add: %v", err)
		}
		ended, err := s.Passwords.Remove(ctx, id)
		if err != nil || len(ended) != 1 || ended[0].To != s.Running {
			t.Fatalf("Remove = %+v, %v; want the grant to %s", ended, err, s.Running)
		}
	})
}
