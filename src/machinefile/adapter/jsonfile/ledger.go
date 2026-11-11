package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
)

const Version = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
}

// value: the ledger, one file: `~/.local/state/station/ledger.json`.

type Ledger struct {
	files Files
	at    string
}

var (
	_ contract.Ledger = (*Ledger)(nil)
	_ store.Migrator  = (*Ledger)(nil)
)

func NewLedger(files Files, at string) *Ledger { return &Ledger{files: files, at: at} }

type file struct {
	V         int                     `json:"v"`
	Installed []machinefile.Installed `json:"installed"`
}

func (l *Ledger) read() ([]machinefile.Installed, error) {
	data, err := l.files.ReadFile(l.at)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: l.at, Cause: err}
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, &store.UnreadableError{Path: l.at, Cause: err}
	}
	if _, err := store.Current(f.V, Version); err != nil {
		return nil, &store.UnreadableError{Path: l.at, Cause: err}
	}
	return f.Installed, nil
}

func (l *Ledger) write(list []machinefile.Installed) error {
	slices.SortFunc(list, func(a, b machinefile.Installed) int { return strings.Compare(a.Item, b.Item) })
	if err := l.files.MkdirAll(path.Dir(l.at), 0o700); err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	data, err := json.MarshalIndent(file{V: Version, Installed: list}, "", "  ")
	if err != nil {
		return err
	}
	if err := l.files.WriteFile(l.at, append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	return nil
}

func (l *Ledger) List(context.Context) ([]machinefile.Installed, error) { return l.read() }

func (l *Ledger) Put(_ context.Context, in machinefile.Installed) error {
	list, err := l.read()
	if err != nil {
		return err
	}
	list = slices.DeleteFunc(list, func(e machinefile.Installed) bool { return e.Item == in.Item })
	return l.write(append(list, in))
}

func (l *Ledger) Delete(_ context.Context, item string) error {
	list, err := l.read()
	if err != nil || !slices.ContainsFunc(list, func(e machinefile.Installed) bool { return e.Item == item }) {
		return err
	}
	return l.write(slices.DeleteFunc(list, func(e machinefile.Installed) bool { return e.Item == item }))
}

func (l *Ledger) Migrate(context.Context) error {
	// invariant: version 1 is the first; a ledger a newer station wrote is
	// refused rather than read.
	_, err := l.read()
	return store.OnlyNewer(err)
}
