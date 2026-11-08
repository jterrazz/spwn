package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
)

const Version = 1

type Files interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	ReadDir(path string) ([]fs.DirEntry, error)
	Remove(path string) error
}

type Store struct {
	files Files
	dir   string
}

var (
	_ contract.MachineStore = (*Store)(nil)
	_ store.Migrator        = (*Store)(nil)
)

func New(files Files, dir string) *Store { return &Store{files: files, dir: dir} }

func (s *Store) recordFile(name string) string { return path.Join(s.dir, name+".json") }

// value: alias of the v2 `lent` key, kept in a VM record until Accounts
// moves its shares; the record is otherwise v3's.
type file struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Version     string            `json:"version"`
	Lifetime    machines.Lifetime `json:"lifetime"`
	Want        string            `json:"want,omitempty"`
	Network     string            `json:"network,omitempty"`
	Check       string            `json:"check,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	CPUs        int               `json:"cpus"`
	MemoryBytes int64             `json:"memory_bytes"`
	DiskBytes   int64             `json:"disk_bytes"`
	Shared      json.RawMessage   `json:"lent,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	V           int               `json:"v,omitempty"`
}

func (f file) record() machines.Record {
	return machines.Record{
		Name: f.Name, Image: f.Image, Version: f.Version, Lifetime: f.Lifetime, Want: f.Want, Network: f.Network, Check: f.Check, Env: f.Env,
		CPUs: f.CPUs, MemoryBytes: f.MemoryBytes, DiskBytes: f.DiskBytes, CreatedAt: f.CreatedAt,
	}
}

func (s *Store) read(at string) (file, error) {
	data, err := s.files.ReadFile(at)
	if err != nil {
		return file{}, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return file{}, &store.UnreadableError{Path: at, Cause: err}
	}
	if _, err := store.Current(f.V, Version); err != nil {
		return file{}, &store.UnreadableError{Path: at, Cause: err}
	}
	return f, nil
}

func (s *Store) Get(_ context.Context, name string) (machines.Record, error) {
	f, err := s.read(s.recordFile(name))
	if errors.Is(err, fs.ErrNotExist) {
		return machines.Record{}, store.ErrNotFound
	}
	if err != nil {
		var unreadable *store.UnreadableError
		if !errors.As(err, &unreadable) {
			err = &store.UnreadableError{Path: s.recordFile(name), Cause: err}
		}
		return machines.Record{}, err
	}
	return f.record(), nil
}

func (s *Store) Put(_ context.Context, r machines.Record) error {
	if err := s.files.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	// invariant: a put keeps the shares on disk as they are: the accounts'
	// store writes them, and a record held in memory may predate a share.
	var held file
	if current, err := s.files.ReadFile(s.recordFile(r.Name)); err == nil {
		_ = json.Unmarshal(current, &held)
	}
	data, err := json.MarshalIndent(file{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		CPUs: r.CPUs, MemoryBytes: r.MemoryBytes, DiskBytes: r.DiskBytes, Shared: held.Shared, CreatedAt: r.CreatedAt,
	}, "", "  ")
	if err != nil {
		return err
	}
	return s.files.WriteFile(s.recordFile(r.Name), append(data, '\n'), 0o600)
}

func (s *Store) List(context.Context) ([]machines.Record, error) { return s.list(false) }

func (s *Store) ListReadable(context.Context) ([]machines.Record, error) { return s.list(true) }

func (s *Store) list(skipDamaged bool) ([]machines.Record, error) {
	entries, err := s.files.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &store.UnreadableError{Path: s.dir, Cause: err}
	}
	var out []machines.Record
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		at := path.Join(s.dir, e.Name())
		f, err := s.read(at)
		var unreadable *store.UnreadableError
		if skipDamaged && errors.As(err, &unreadable) && store.OnlyNewer(err) == nil {
			continue
		}
		if err != nil {
			if !errors.As(err, &unreadable) {
				err = &store.UnreadableError{Path: at, Cause: err}
			}
			return nil, err
		}
		out = append(out, f.record())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) Delete(_ context.Context, name string) error {
	if err := s.files.Remove(s.recordFile(name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Store) Migrate(context.Context) error {
	entries, err := s.files.ReadDir(s.dir)
	if err != nil {
		return store.OnlyNewer(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		_, err = s.read(path.Join(s.dir, e.Name()))
		if store.OnlyNewer(err) != nil {
			return err
		}
	}
	return nil
}
