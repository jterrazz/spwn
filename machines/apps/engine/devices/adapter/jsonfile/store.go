package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/devices"
	"github.com/jterrazz/studio-station/apps/engine/devices/contract"
)

const Version = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	Replace(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	// invariant: an exclusive lock on path, made when missing, held until
	// release; a second holder waits for it or for ctx.
	Hold(ctx context.Context, path string) (release func(), err error)
}

type Store struct {
	// invariant: the device book is one file of mode 0600,
	// `~/.local/state/station/devices.json`, replaced whole, beside the lock
	// every change takes, `devices.json.lock`.
	files Files
	at    string
}

var _ contract.Store = (*Store)(nil)

func New(files Files, at string) *Store { return &Store{files: files, at: at} }

type file struct {
	V int `json:"v,omitempty"`
	devices.Book
}

func (s *Store) Read(context.Context) (devices.Book, error) {
	data, err := s.files.ReadFile(s.at)
	if errors.Is(err, fs.ErrNotExist) {
		return devices.Book{Devices: []devices.Device{}}, nil
	}
	if err != nil {
		return devices.Book{}, &store.UnreadableError{Path: s.at, Cause: err}
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return devices.Book{}, &store.UnreadableError{Path: s.at, Cause: err, Line: store.LineOf(data, err)}
	}
	if err := store.Current(f.V, Version); err != nil {
		return devices.Book{}, &store.UnreadableError{Path: s.at, Cause: err}
	}
	if f.Devices == nil {
		f.Devices = []devices.Device{}
	}
	return f.Book, nil
}

func (s *Store) Change(ctx context.Context, edit func(*devices.Book) error) error {
	if err := s.files.MkdirAll(path.Dir(s.at), 0o700); err != nil {
		return &store.UnwritableError{Path: s.at, Cause: err}
	}
	release, err := s.files.Hold(ctx, s.at+".lock")
	if err != nil {
		return &store.UnwritableError{Path: s.at, Cause: err}
	}
	defer release()
	kept, err := s.Read(ctx)
	if err != nil {
		return err
	}
	if refused := edit(&kept); refused != nil {
		return refused
	}
	data, err := json.MarshalIndent(file{Book: kept}, "", "  ")
	if err != nil {
		return err
	}
	if err := s.files.Replace(s.at, append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: s.at, Cause: err}
	}
	return nil
}
