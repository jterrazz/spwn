package keychain

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
	"github.com/jterrazz/studio-station/apps/engine/platform/local"
)

type Passwords struct{ k *Kept }

var _ contract.Passwords = (*Passwords)(nil)

func (p *Passwords) List(ctx context.Context) (accounts.VaultRead, error) {
	b, err := p.k.book.Read(ctx)
	if err != nil {
		return accounts.VaultRead{}, err
	}
	for i := range b.Entries {
		if b.Entries[i].Grants == nil {
			b.Entries[i].Grants = []accounts.VaultGrant{}
		}
	}
	return accounts.VaultRead{Mac: p.k.o.Mac, Entries: b.Entries}, nil
}

func (p *Passwords) Check(ctx context.Context, id string) error {
	_, err := p.k.entry(ctx, id, accounts.VaultUnknown)
	return err
}

// invariant: a password from a sign-in is the token the program's own
// command prints on this Mac, read from its stdout and never shown; a
// program with no such command cannot be taken, and one that prints none
// is not signed in.
func (p *Passwords) capture(ctx context.Context, program accounts.Program) ([]byte, error) {
	d, ok := program.Delivery()
	if !ok || len(d.Capture) == 0 {
		return nil, accounts.Refusal(fail.Failed, accounts.VaultCannotCapture, string(program))
	}
	out, err := p.k.run.Output(ctx, local.Cmd{Name: d.Capture[0], Args: d.Capture[1:]})
	token := strings.TrimSpace(out.Stdout)
	if err != nil || out.ExitCode != 0 || token == "" {
		return nil, accounts.Refusal(fail.Refused, accounts.VaultNotSignedIn, string(program))
	}
	return []byte(token), nil
}

func (p *Passwords) Add(ctx context.Context, in contract.PasswordIn) (string, error) {
	var id string
	err := p.k.book.Change(ctx, func(b *accounts.Book) error {
		if slices.ContainsFunc(b.Entries, func(e accounts.VaultEntry) bool { return e.Label == in.Label }) {
			return accounts.Refusal(fail.Refused, accounts.VaultLabelTaken, in.Label)
		}
		value := in.Value.Reveal()
		if in.From == accounts.FromSignIn {
			taken, err := p.capture(ctx, in.Program)
			if err != nil {
				return err
			}
			value = taken
		}
		id = b.Next()
		if err := p.k.keys.Put(ctx, keyOf(id), value); err != nil {
			return err
		}
		b.Entries = append(b.Entries, accounts.VaultEntry{ID: id, Label: in.Label, Program: in.Program, Kind: in.Kind, Added: p.k.o.Now().UTC().Truncate(time.Second), AgentGrantable: in.Kind == accounts.KeyFineGrained, Grants: []accounts.VaultGrant{}})
		return nil
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func (p *Passwords) Remove(ctx context.Context, id string) ([]accounts.VaultGrant, error) {
	// invariant: a password removed ends every grant made from it, each by its
	// program's own Remove inside the machine, then its value leaves the
	// Keychain; a Remove that fails leaves the grant to the expiry, past due.
	ended := []accounts.VaultGrant{}
	err := p.k.book.Change(ctx, func(b *accounts.Book) error {
		e, ok := b.Find(id)
		if !ok {
			return accounts.Refusal(fail.NotFound, accounts.VaultUnknown, id)
		}
		ended = slices.Clone(e.Grants)
		program := e.Program
		if err := p.k.keys.Forget(ctx, keyOf(id)); err != nil && !errors.Is(err, accounts.ErrNotKept) {
			return err
		}
		b.Entries = slices.DeleteFunc(b.Entries, func(v accounts.VaultEntry) bool { return v.ID == id })
		for _, g := range ended {
			if g.ToKind == accounts.ToMachine && p.k.takeBack(ctx, program, g.To) != nil {
				b.Owed = append(b.Owed, accounts.Owed{Program: program, To: g.To})
			}
		}
		return nil
	})
	return ended, err
}
