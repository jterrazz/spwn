package contracttest

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
)

func BookStore(t *testing.T, open func(t *testing.T) contract.BookStore) {
	// invariant: open hands an empty store; a second open in one test is no
	// promise, so each case works on one store.
	ctx := context.Background()
	entry := func(id string) accounts.VaultEntry {
		return accounts.VaultEntry{ID: id, Label: "deploy " + id, Program: "docs-bot", Kind: accounts.KeyFineGrained, Added: time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC), Grants: []accounts.VaultGrant{}}
	}

	t.Run("an empty store reads a book with no entry", func(t *testing.T) {
		got, err := open(t).Read(ctx)
		if err != nil || len(got.Entries) != 0 || got.Counter != 0 || len(got.Owed) != 0 {
			t.Fatalf("Read = %+v, %v; want an empty book", got, err)
		}
	})

	t.Run("a change is read back whole", func(t *testing.T) {
		s := open(t)
		var id string
		err := s.Change(ctx, func(b *accounts.Book) error {
			id = b.Next()
			b.Entries = append(b.Entries, entry(id))
			b.Owed = append(b.Owed, accounts.Owed{Program: "docs-bot", To: "linux-4"})
			return nil
		})
		if err != nil {
			t.Fatalf("Change: %v", err)
		}
		want := accounts.Book{Counter: 1, Entries: []accounts.VaultEntry{entry(id)}, Owed: []accounts.Owed{{Program: "docs-bot", To: "linux-4"}}}
		if got, err := s.Read(ctx); err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("Read = %+v, %v; want %+v", got, err, want)
		}
	})

	t.Run("a change whose edit fails writes nothing", func(t *testing.T) {
		s := open(t)
		refused := errors.New("refused")
		err := s.Change(ctx, func(b *accounts.Book) error {
			b.Entries = append(b.Entries, entry(b.Next()))
			return refused
		})
		if !errors.Is(err, refused) {
			t.Fatalf("Change = %v; want the edit's error", err)
		}
		if got, err := s.Read(ctx); err != nil || len(got.Entries) != 0 || got.Counter != 0 {
			t.Fatalf("Read = %+v, %v; want the book unchanged", got, err)
		}
	})

	t.Run("an id removed is never given again", func(t *testing.T) {
		s := open(t)
		var first, second string
		_ = s.Change(ctx, func(b *accounts.Book) error {
			first = b.Next()
			b.Entries = append(b.Entries, entry(first))
			return nil
		})
		_ = s.Change(ctx, func(b *accounts.Book) error { b.Entries = nil; return nil })
		_ = s.Change(ctx, func(b *accounts.Book) error { second = b.Next(); return nil })
		if first == second {
			t.Fatalf("the id %s was given twice", first)
		}
	})

	t.Run("changes at once keep every entry", func(t *testing.T) {
		s := open(t)
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				if err := s.Change(ctx, func(b *accounts.Book) error { b.Entries = append(b.Entries, entry(b.Next())); return nil }); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
		if got, err := s.Read(ctx); err != nil || len(got.Entries) != 8 || got.Counter != 8 {
			t.Fatalf("Read = %d entries, counter %d, %v; want 8 and 8", len(got.Entries), got.Counter, err)
		}
	})
}
