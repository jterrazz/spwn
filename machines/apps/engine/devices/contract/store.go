package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/devices"
)

type Store interface {
	// invariant: the device book as the last change left it, seals as they
	// were written; none written reads as a book with no entry.
	Read(ctx context.Context) (devices.Book, error)
	// invariant: a change reads the book and writes it back under one
	// exclusive hold, so two changes at once never lose each other's; a
	// change whose edit answers an error writes nothing and answers it.
	Change(ctx context.Context, edit func(*devices.Book) error) error
}
