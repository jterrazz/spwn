package contracttest

import (
	"context"
	"testing"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/secret"
)

// value: a program whose sign-in this Mac can take a token from.
func capturing() accounts.Program {
	for _, d := range catalog.Deliveries {
		if len(d.Capture) > 0 {
			return accounts.Program(d.Program)
		}
	}
	return ""
}

func typed(label string) contract.PasswordIn {
	return contract.PasswordIn{Label: label, Program: capturing(), Kind: accounts.KeyFineGrained, From: accounts.FromTyped, Value: secret.Sent{Value: secret.Hold("", []byte("FAKE000"))}}
}

func listed(t *testing.T, p contract.Passwords, id string) (accounts.VaultEntry, bool) {
	t.Helper()
	read, err := p.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, e := range read.Entries {
		if e.ID == id {
			return e, true
		}
	}
	return accounts.VaultEntry{}, false
}

func Passwords(t *testing.T, subject func(t *testing.T) contract.Passwords) {
	// value: the subject is an empty vault of a Mac where no program is signed in.
	ctx := context.Background()

	t.Run("a typed password is listed by its label, program and kind, its grants none", func(t *testing.T) {
		p := subject(t)
		id, err := p.Add(ctx, typed("GitHub deploy"))
		if err != nil || id == "" {
			t.Fatalf("Add = %q, %v", id, err)
		}
		e, ok := listed(t, p, id)
		if !ok || e.Label != "GitHub deploy" || e.Program != capturing() || e.Kind != accounts.KeyFineGrained || len(e.Grants) != 0 || e.Grants == nil {
			t.Fatalf("listed %+v, %v; want the typed password, no grant", e, ok)
		}
		if err := p.Check(ctx, id); err != nil {
			t.Fatalf("Check: %v", err)
		}
	})

	t.Run("a label taken is refused", func(t *testing.T) {
		p := subject(t)
		if _, err := p.Add(ctx, typed("GitHub deploy")); err != nil {
			t.Fatalf("Add: %v", err)
		}
		if _, err := p.Add(ctx, typed("GitHub deploy")); codeOf(err) != accounts.VaultLabelTaken {
			t.Fatalf("Add again = %v; want %s", err, accounts.VaultLabelTaken)
		}
	})

	t.Run("a password taken from a program signed out here is refused", func(t *testing.T) {
		in := typed("from a sign-in")
		in.From, in.Value = accounts.FromSignIn, secret.Sent{}
		if _, err := subject(t).Add(ctx, in); codeOf(err) != accounts.VaultNotSignedIn {
			t.Fatalf("Add = %v; want %s", err, accounts.VaultNotSignedIn)
		}
	})

	t.Run("a password the vault does not keep is unknown", func(t *testing.T) {
		p := subject(t)
		if err := p.Check(ctx, "pw-404"); codeOf(err) != accounts.VaultUnknown {
			t.Fatalf("Check = %v; want %s", err, accounts.VaultUnknown)
		}
		if _, err := p.Remove(ctx, "pw-404"); codeOf(err) != accounts.VaultUnknown {
			t.Fatalf("Remove = %v; want %s", err, accounts.VaultUnknown)
		}
	})

	t.Run("a password removed is no longer listed nor known", func(t *testing.T) {
		p := subject(t)
		id, err := p.Add(ctx, typed("GitHub deploy"))
		if err != nil {
			t.Fatalf("Add: %v", err)
		}
		if ended, err := p.Remove(ctx, id); err != nil || ended == nil || len(ended) != 0 {
			t.Fatalf("Remove = %v, %v; want no grant ended", ended, err)
		}
		if _, ok := listed(t, p, id); ok {
			t.Fatal("the removed password is still listed")
		}
		if err := p.Check(ctx, id); codeOf(err) != accounts.VaultUnknown {
			t.Fatalf("Check = %v; want %s", err, accounts.VaultUnknown)
		}
	})
}
