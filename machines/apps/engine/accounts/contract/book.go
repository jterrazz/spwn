package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

type BookStore interface {
	// invariant: the vault's book as the last change left it; none written
	// reads as a book with no entry.
	Read(ctx context.Context) (accounts.Book, error)
	// invariant: a change reads the book and writes it back under one
	// exclusive hold, so two changes at once never lose each other's; a
	// change whose edit answers an error writes nothing and answers it.
	Change(ctx context.Context, edit func(*accounts.Book) error) error
}
