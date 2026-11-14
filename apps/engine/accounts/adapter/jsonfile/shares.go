package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/accounts/contract"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
)

const Version = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
}

type Shares struct {
	// invariant: every share of this Mac lives in one file, by VM; a VM's
	// record beside it (records) is read to know the VM, and written only by
	// the migration that moves the shares station v2 kept there.
	files   Files
	file    string
	records string
}

var (
	_ contract.ShareStore = (*Shares)(nil)
	_ store.Migrator      = (*Shares)(nil)
)

func NewShares(files Files, file, records string) *Shares {
	return &Shares{files: files, file: file, records: records}
}

type document struct {
	V      int                         `json:"v,omitempty"`
	Shares map[string][]accounts.Share `json:"shares"`
}

func (s *Shares) load() (map[string][]accounts.Share, error) {
	data, err := s.files.ReadFile(s.file)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string][]accounts.Share{}, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: s.file, Cause: err}
	}
	var doc document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, &store.UnreadableError{Path: s.file, Cause: err, Line: store.LineOf(data, err)}
	}
	if _, err := store.Current(doc.V, Version); err != nil {
		return nil, &store.UnreadableError{Path: s.file, Cause: err}
	}
	if doc.Shares == nil {
		doc.Shares = map[string][]accounts.Share{}
	}
	return doc.Shares, nil
}

func (s *Shares) save(held map[string][]accounts.Share) error {
	data, err := json.MarshalIndent(document{Shares: held}, "", "  ")
	if err != nil {
		return err
	}
	if err := s.files.MkdirAll(path.Dir(s.file), 0o700); err != nil {
		return err
	}
	return s.files.WriteFile(s.file, append(data, '\n'), 0o600)
}

func (s *Shares) record(vm string) string { return path.Join(s.records, vm+".json") }

func (s *Shares) Get(_ context.Context, vm string) ([]accounts.Share, error) {
	held, err := s.load()
	if err != nil {
		return nil, err
	}
	return slices.Clone(held[vm]), nil
}

func (s *Shares) Put(_ context.Context, vm string, shares []accounts.Share) error {
	if _, err := s.files.Stat(s.record(vm)); err != nil {
		return errors.Join(store.ErrNotFound, errors.New("no VM "+vm))
	}
	held, err := s.load()
	if err != nil {
		return err
	}
	if len(shares) == 0 {
		delete(held, vm)
	} else {
		held[vm] = slices.Clone(shares)
	}
	return s.save(held)
}

func (s *Shares) List(context.Context) (map[string][]accounts.Share, error) {
	held, err := s.load()
	if err != nil {
		return nil, err
	}
	return maps.Clone(held), nil
}

func (s *Shares) Delete(_ context.Context, vm string) error {
	held, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := held[vm]; !ok {
		return nil
	}
	delete(held, vm)
	return s.save(held)
}

func (s *Shares) Migrate(context.Context) error {
	// invariant: the shares station v2 kept under each VM record's `lent`
	// key move into the one file once, merged by name with any kept there,
	// and leave the record; a second run finds none and writes nothing.
	entries, err := s.files.ReadDir(s.records)
	if err != nil {
		return store.OnlyNewer(err)
	}
	held, err := s.load()
	if err != nil {
		return store.OnlyNewer(err)
	}
	for _, e := range entries {
		vm, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() {
			continue
		}
		shares, rest, found := s.sharedIn(vm)
		if !found {
			continue
		}
		for _, sh := range shares {
			if !slices.ContainsFunc(held[vm], func(have accounts.Share) bool { return have.Name == sh.Name }) {
				held[vm] = append(held[vm], sh)
			}
		}
		if err := s.save(held); err != nil {
			return err
		}
		data, err := json.MarshalIndent(rest, "", "  ")
		if err != nil {
			return err
		}
		if err := s.files.WriteFile(s.record(vm), append(data, '\n'), 0o600); err != nil {
			return err
		}
	}
	return nil
}

// value: alias of v2's `lent` key of a VM record, read once and moved.
//
// invariant: a v2 record holds its shares as plain names (before shares
// expired) or as {name, at, until}; a plain name reads as shared at the
// record's mtime for the default term.
func (s *Shares) sharedIn(vm string) (shares []accounts.Share, rest map[string]json.RawMessage, found bool) {
	data, err := s.files.ReadFile(s.record(vm))
	if err != nil {
		return nil, nil, false
	}
	rest = map[string]json.RawMessage{}
	if json.Unmarshal(data, &rest) != nil {
		return nil, nil, false
	}
	raw, ok := rest["lent"]
	if !ok {
		return nil, nil, false
	}
	delete(rest, "lent")
	var timed []accounts.Share
	if json.Unmarshal(raw, &timed) == nil {
		return timed, rest, true
	}
	var names []string
	if json.Unmarshal(raw, &names) != nil {
		return nil, rest, true
	}
	at := time.Time{}
	if info, statErr := s.files.Stat(s.record(vm)); statErr == nil {
		at = info.ModTime().UTC().Truncate(time.Second)
	}
	for _, n := range names {
		shares = append(shares, accounts.Share{Name: n, At: at, Until: at.Add(accounts.DefaultShareTerm)})
	}
	return shares, rest, true
}
