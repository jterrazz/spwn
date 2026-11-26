package keychain

import (
	"context"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/fail"
)

type Options struct {
	// value: this Mac's name, whose vault it is.
	Mac string
	Now func() time.Time
	// value: the system root `security` is read under; empty on a Mac.
	Root string
}

type Kept struct {
	// invariant: the vault is one Mac's: each value is a generic password of
	// station's Keychain service, kept on this Mac and never synced; the
	// book, its store, lists the passwords and their grants without a value.
	keys Keychain
	run  Runner
	book contract.BookStore
	o    Options
	d    GrantDeps
}

func NewKept(run Runner, book contract.BookStore, o Options, d GrantDeps) *Kept {
	return &Kept{keys: New(run, o.Root), run: run, book: book, o: o, d: d}
}

func (k *Kept) Passwords() *Passwords { return &Passwords{k} }

func (k *Kept) Grants() *Grants { return &Grants{k} }

func keyOf(id string) string { return "vault:" + id }

func (k *Kept) entry(ctx context.Context, id string, code fail.Code) (accounts.VaultEntry, error) {
	b, err := k.book.Read(ctx)
	if err != nil {
		return accounts.VaultEntry{}, err
	}
	e, ok := b.Find(id)
	if !ok {
		return accounts.VaultEntry{}, accounts.Refusal(fail.NotFound, code, id)
	}
	return *e, nil
}
