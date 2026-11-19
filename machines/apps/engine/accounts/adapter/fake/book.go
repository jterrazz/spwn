package fake

import (
	"context"
	"slices"
	"sync"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

type Book struct {
	// invariant: one Mac's book, in memory, changed under one mutex.
	mu   sync.Mutex
	kept accounts.Book
}

var _ contract.BookStore = (*Book)(nil)

func NewBook() *Book { return &Book{} }

func clone(b accounts.Book) accounts.Book {
	out := accounts.Book{Counter: b.Counter, Entries: make([]accounts.VaultEntry, 0, len(b.Entries)), Owed: slices.Clone(b.Owed)}
	for _, e := range b.Entries {
		e.Grants = slices.Clone(e.Grants)
		out.Entries = append(out.Entries, e)
	}
	return out
}

func (b *Book) Read(context.Context) (accounts.Book, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return clone(b.kept), nil
}

func (b *Book) Change(_ context.Context, edit func(*accounts.Book) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	next := clone(b.kept)
	if err := edit(&next); err != nil {
		return err
	}
	b.kept = next
	return nil
}
