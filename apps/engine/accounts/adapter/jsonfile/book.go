package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

const BookVersion = 1

type BookFiles interface {
	ReadFile(path string) ([]byte, error)
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	// invariant: an exclusive lock on path, made when missing, held until
	// release; a second holder waits for it or for ctx.
	Hold(ctx context.Context, path string) (release func(), err error)
}

type Book struct {
	// invariant: the vault's book is one file of mode 0600,
	// `~/.local/state/station/vault.json`, beside the lock every change
	// takes, `vault.json.lock`; it never holds a value.
	files BookFiles
	at    string
}

var (
	_ contract.BookStore = (*Book)(nil)
)

func NewBook(files BookFiles, at string) *Book { return &Book{files: files, at: at} }

type bookFile struct {
	V int `json:"v,omitempty"`
	accounts.Book
}

func (b *Book) Read(context.Context) (accounts.Book, error) {
	data, err := b.files.ReadFile(b.at)
	if errors.Is(err, fs.ErrNotExist) {
		return accounts.Book{Entries: []accounts.VaultEntry{}}, nil
	}
	if err != nil {
		return accounts.Book{}, &store.UnreadableError{Path: b.at, Cause: err}
	}
	var f bookFile
	if err := json.Unmarshal(data, &f); err != nil {
		return accounts.Book{}, &store.UnreadableError{Path: b.at, Cause: err, Line: store.LineOf(data, err)}
	}
	if err := store.Current(f.V, BookVersion); err != nil {
		return accounts.Book{}, &store.UnreadableError{Path: b.at, Cause: err}
	}
	if f.Entries == nil {
		f.Entries = []accounts.VaultEntry{}
	}
	return f.Book, nil
}

func (b *Book) Change(ctx context.Context, edit func(*accounts.Book) error) error {
	if err := b.files.MkdirAll(path.Dir(b.at), 0o700); err != nil {
		return &store.UnwritableError{Path: b.at, Cause: err}
	}
	release, err := b.files.Hold(ctx, b.at+".lock")
	if err != nil {
		return &store.UnwritableError{Path: b.at, Cause: err}
	}
	defer release()
	kept, err := b.Read(ctx)
	if err != nil {
		return err
	}
	if refused := edit(&kept); refused != nil {
		return refused
	}
	data, err := json.MarshalIndent(bookFile{Book: kept}, "", "  ")
	if err != nil {
		return err
	}
	if err := b.files.Replace(b.at, append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: b.at, Cause: err}
	}
	return nil
}
