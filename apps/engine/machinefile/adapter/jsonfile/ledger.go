package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machinefile"
	"github.com/jterrazz/studio-station/apps/engine/machinefile/contract"
)

const Version = 2

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Rename(from, to string) error
	// invariant: an exclusive lock on path, made when missing, held until
	// release; a second holder waits for it.
	Lock(path string) (release func(), err error)
}

// value: the ledger, one file: `~/.local/state/station/ledger.json`,
// beside the lock its writers take, `ledger.json.lock`.

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
	Blocks    []machinefile.Written   `json:"blocks,omitempty"`
	// value: the last night the engine's upgrade recorded.
	Night *machinefile.Night `json:"night,omitempty"`
}

func (l *Ledger) read() (file, error) {
	data, err := l.files.ReadFile(l.at)
	if errors.Is(err, fs.ErrNotExist) {
		return file{}, nil
	}
	if err != nil {
		return file{}, &store.UnreadableError{Path: l.at, Cause: err}
	}
	var f file
	if err = json.Unmarshal(data, &f); err != nil {
		return file{}, &store.UnreadableError{Path: l.at, Cause: err}
	}
	v, err := store.Current(f.V, Version)
	if err != nil {
		return file{}, &store.UnreadableError{Path: l.at, Cause: err}
	}
	if v < Version {
		return file{}, &store.UnreadableError{Path: l.at, Cause: &store.OlderError{Version: v, Current: Version}}
	}
	return f, nil
}

func (l *Ledger) write(f file) error {
	slices.SortFunc(f.Installed, func(a, b machinefile.Installed) int { return strings.Compare(a.Package, b.Package) })
	slices.SortFunc(f.Blocks, func(a, b machinefile.Written) int { return strings.Compare(a.ID, b.ID) })
	if err := l.files.MkdirAll(path.Dir(l.at), 0o700); err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	f.V = Version
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.at + ".tmp"
	if err := l.files.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	if err := l.files.Rename(tmp, l.at); err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	return nil
}

// invariant: a record changes under the ledger's lock, read then written
// whole to a temporary file renamed over it, so two applies at once (two
// terminals, the menu bar's and the CLI's) never lose each other's record,
// and a reader never sees half a file. The lock is held for one record,
// so a second apply waits for it rather than being refused.
func (l *Ledger) change(edit func(*file) bool) error {
	if err := l.files.MkdirAll(path.Dir(l.at), 0o700); err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	release, err := l.files.Lock(l.at + ".lock")
	if err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	defer release()
	f, err := l.read()
	if err != nil {
		return err
	}
	if !edit(&f) {
		return nil
	}
	return l.write(f)
}

func (l *Ledger) Night(context.Context) (machinefile.Night, bool, error) {
	f, err := l.read()
	if err != nil || f.Night == nil {
		return machinefile.Night{}, false, err
	}
	return *f.Night, true, nil
}

func (l *Ledger) Write(_ context.Context, n machinefile.Night) error {
	return l.change(func(f *file) bool {
		f.Night = &n
		return true
	})
}

func (l *Ledger) List(context.Context) ([]machinefile.Installed, error) {
	f, err := l.read()
	return f.Installed, err
}

func (l *Ledger) Put(_ context.Context, in machinefile.Installed) error {
	return l.change(func(f *file) bool {
		f.Installed = append(slices.DeleteFunc(f.Installed, func(e machinefile.Installed) bool { return e.Package == in.Package }), in)
		return true
	})
}

func (l *Ledger) Delete(_ context.Context, pkg string) error {
	return l.change(func(f *file) bool {
		of := func(e machinefile.Installed) bool { return e.Package == pkg }
		held := slices.ContainsFunc(f.Installed, of)
		f.Installed = slices.DeleteFunc(f.Installed, of)
		return held
	})
}

func (l *Ledger) Blocks(context.Context) ([]machinefile.Written, error) {
	f, err := l.read()
	return f.Blocks, err
}

func (l *Ledger) PutBlock(_ context.Context, w machinefile.Written) error {
	return l.change(func(f *file) bool {
		f.Blocks = append(slices.DeleteFunc(f.Blocks, func(e machinefile.Written) bool { return e.ID == w.ID }), w)
		return true
	})
}

func (l *Ledger) DeleteBlock(_ context.Context, id string) error {
	return l.change(func(f *file) bool {
		of := func(e machinefile.Written) bool { return e.ID == id }
		held := slices.ContainsFunc(f.Blocks, of)
		f.Blocks = slices.DeleteFunc(f.Blocks, of)
		return held
	})
}

// value: the ledger's version 1, whose records name their package by
// `item`: read by Migrate alone, which writes version 2.
type fileV1 struct {
	Installed []struct {
		machinefile.Installed
		Item string `json:"item"`
	} `json:"installed"`
	Blocks []struct {
		machinefile.Written
		Item string `json:"item"`
	} `json:"blocks,omitempty"`
	Night *machinefile.Night `json:"night,omitempty"`
}

func (l *Ledger) Migrate(context.Context) error {
	// invariant: a ledger at version 1 is rewritten at 2 under the lock, each
	// record's package kept; one at 2, or none, is left as it is, and one a
	// newer station wrote is refused rather than read.
	if _, err := l.read(); !older(err) {
		return store.OnlyNewer(err)
	}
	release, err := l.files.Lock(l.at + ".lock")
	if err != nil {
		return &store.UnwritableError{Path: l.at, Cause: err}
	}
	defer release()
	if _, err = l.read(); !older(err) {
		return store.OnlyNewer(err)
	}
	data, err := l.files.ReadFile(l.at)
	if err != nil {
		return &store.UnreadableError{Path: l.at, Cause: err}
	}
	var v1 fileV1
	if err := json.Unmarshal(data, &v1); err != nil {
		return &store.UnreadableError{Path: l.at, Cause: err}
	}
	f := file{Night: v1.Night}
	for _, r := range v1.Installed {
		r.Package = r.Item
		f.Installed = append(f.Installed, r.Installed)
	}
	for _, w := range v1.Blocks {
		w.Package = w.Item
		f.Blocks = append(f.Blocks, w.Written)
	}
	return l.write(f)
}

func older(err error) bool {
	var o *store.OlderError
	return errors.As(err, &o)
}
