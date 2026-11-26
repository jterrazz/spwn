package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/core/fail"

	"github.com/jterrazz/studio-station/apps/engine/devices"
)

type Store interface {
	// invariant: the device book as the last write left it, seals as they
	// were written, and the head this Mac anchored last; none written reads
	// as a book with no entry and the zero Head.
	Read(ctx context.Context) (devices.Book, devices.Head, error)
	// invariant: a change reads the book and writes it back under one
	// exclusive hold, so two changes at once never lose each other's; its
	// edit works on devices.Base and the book written is devices.Next. The
	// anchor is written before the file, so a write cut between the two
	// leaves a book that is not the anchored one, trusted nowhere. A change
	// whose edit answers an error writes nothing and answers it.
	Change(ctx context.Context, edit func(*devices.Book) error) error
	// invariant: from now on this process is the book's only writer, this
	// Mac's engine: once it holds a head it reads the anchor never again,
	// so a book or an anchor it did not write, at any generation, is not
	// the anchored one, and the next change starts from an empty book.
	Sole()
	// value: the head this process holds, the one it read first or wrote
	// last; changed when the book on disk is not that one; ok false before
	// it read one.
	Held(ctx context.Context) (head devices.Head, changed, ok bool)
}

type Anchor interface {
	// invariant: the head this Mac last wrote of its book, kept where a
	// writer of the state folder cannot roll it back; none kept reads as
	// the zero Head, and one that cannot be read is an error, never zero.
	Read(ctx context.Context) (devices.Head, error)
	Write(ctx context.Context, h devices.Head) error
}

type AnchorUnreadableError struct {
	// value: why the login keychain did not give the book's head.
	Cause string
}

func (e *AnchorUnreadableError) Error() string {
	return "the device book's head is not readable in the login keychain: " + e.Cause
}

func (e *AnchorUnreadableError) Unwrap() error {
	// value: its repair, the same whichever process meets it, the engine's
	// included.
	return &fail.Said{Message: e.Error(), Hint: "unlock this Mac's login keychain, then run it again"}
}
