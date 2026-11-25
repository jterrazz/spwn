package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"sync"

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
	// every change takes, `devices.json.lock`; its head is the anchor's.
	files  Files
	at     string
	anchor contract.Anchor
	mu     sync.Mutex
	// value: the highest head this process read from the anchor or wrote
	// to it. While the file's generation is that head's, the anchor is not
	// read again, so a walk costs no read of it; an anchor read lower than
	// it is never believed.
	held  devices.Head
	known bool
}

var _ contract.Store = (*Store)(nil)

func New(files Files, at string, anchor contract.Anchor) *Store {
	return &Store{files: files, at: at, anchor: anchor}
}

type file struct {
	V int `json:"v,omitempty"`
	devices.Book
}

func (s *Store) Read(ctx context.Context) (devices.Book, devices.Head, error) {
	book, err := s.book()
	if err != nil {
		return devices.Book{}, devices.Head{}, err
	}
	anchored, err := s.anchored(ctx, book.Generation)
	if err != nil {
		return devices.Book{}, devices.Head{}, err
	}
	return book, anchored, nil
}

func (s *Store) book() (devices.Book, error) {
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

func (s *Store) anchored(ctx context.Context, generation int) (devices.Head, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.known && s.held.Generation == generation {
		return s.held, nil
	}
	return s.read(ctx)
}

func (s *Store) read(ctx context.Context) (devices.Head, error) {
	got, err := s.anchor.Read(ctx)
	if err != nil {
		return devices.Head{}, err
	}
	if got.Generation >= s.held.Generation {
		s.held = got
	}
	s.known = true
	return s.held, nil
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
	kept, anchored, err := s.Read(ctx)
	if err != nil {
		return err
	}
	next, err := devices.Next(kept, anchored, edit)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(file{Book: next}, "", "  ")
	if err != nil {
		return err
	}
	if err := s.anchor.Write(ctx, next.Head()); err != nil {
		return err
	}
	s.mu.Lock()
	s.held, s.known = next.Head(), true
	s.mu.Unlock()
	if err := s.files.Replace(s.at, append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: s.at, Cause: err}
	}
	return nil
}
