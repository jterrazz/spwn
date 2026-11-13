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

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
)

const Version = 2

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
	// invariant: one turn of this process at a time holds the records,
	// waited for as long as ctx lasts.
	turn chan struct{}
}

var (
	_ contract.MachineStore = (*Store)(nil)
	_ store.Migrator        = (*Store)(nil)
)

func New(files Files, dir string) *Store {
	return &Store{files: files, dir: dir, turn: make(chan struct{}, 1)}
}

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
	Birth       *machines.Birth   `json:"birth,omitempty"`
	Owner       *machines.Process `json:"owner,omitempty"`
	Agents      []string          `json:"agents,omitempty"`
	Template    string            `json:"template,omitempty"`
	Mounts      []machines.Mount  `json:"mounts,omitempty"`
	Purpose     string            `json:"purpose,omitempty"`
	Creator     string            `json:"creator,omitempty"`
	V           int               `json:"v,omitempty"`
}

func (f file) record() machines.Record {
	return machines.Record{
		Name: f.Name, Image: f.Image, Version: f.Version, Lifetime: f.Lifetime, Want: f.Want, Network: f.Network, Check: f.Check, Env: f.Env,
		CPUs: f.CPUs, MemoryBytes: f.MemoryBytes, DiskBytes: f.DiskBytes, CreatedAt: f.CreatedAt, Birth: f.Birth, Owner: f.Owner, Agents: f.Agents, Template: f.Template, Mounts: f.Mounts,
		Purpose: f.Purpose, Creator: f.Creator,
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

type Locker interface {
	// invariant: what serializes a change between the processes of one
	// Mac: a lock on a file of the records' folder (the disk's flock), held
	// until release; a wait for another holder ends when ctx does, with
	// ctx's error.
	Hold(ctx context.Context, path string) (release func(), err error)
}

// invariant: run runs while this process's turn and the records' lock are
// held, both waited for as long as ctx lasts; a Files with no lock (a
// test's) is one process's alone, which the turn serializes.
func (s *Store) held(ctx context.Context, run func() error) error {
	if err := s.files.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	select {
	case s.turn <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-s.turn }()
	if l, ok := s.files.(Locker); ok {
		release, err := l.Hold(ctx, path.Join(s.dir, ".records.lock"))
		if err != nil {
			return err
		}
		defer release()
	}
	return run()
}

func (s *Store) Amend(ctx context.Context, name string, change func(r *machines.Record)) error {
	return s.held(ctx, func() error {
		r, err := s.Get(ctx, name)
		if err != nil {
			return err
		}
		change(&r)
		return s.put(r)
	})
}

func (s *Store) Claim(ctx context.Context, judge func(held []machines.Record) (machines.Record, error)) error {
	return s.held(ctx, func() error {
		records, err := s.list(false)
		if err != nil {
			return err
		}
		r, err := judge(records)
		if err != nil {
			return err
		}
		return s.put(r)
	})
}

func (s *Store) Put(_ context.Context, r machines.Record) error {
	if err := s.files.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	return s.put(r)
}

func (s *Store) put(r machines.Record) error {
	// invariant: a put keeps the shares on disk as they are: the accounts'
	// store writes them, and a record held in memory may predate a share.
	// A put without agents keeps those on disk too: a stop writes them, and
	// the engine's record of a run that ends predates them.
	var held file
	if current, err := s.files.ReadFile(s.recordFile(r.Name)); err == nil {
		_ = json.Unmarshal(current, &held)
	}
	if r.Agents == nil {
		r.Agents = held.Agents
	}
	data, err := json.MarshalIndent(file{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		CPUs: r.CPUs, MemoryBytes: r.MemoryBytes, DiskBytes: r.DiskBytes, Shared: held.Shared, CreatedAt: r.CreatedAt,
		Birth: r.Birth, Owner: r.Owner, Agents: r.Agents, Template: r.Template, Mounts: r.Mounts, Purpose: r.Purpose, Creator: r.Creator, V: Version,
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

func lifetimeAlias(l machines.Lifetime) machines.Lifetime {
	// value: today's word for a lifetime a record says before version 2,
	// v2's (temporary, lasting) or v3.0's (throwaway, permanent); read by
	// Migrate alone, which writes today's.
	switch string(l) {
	case "temporary", "throwaway":
		return machines.Ephemeral
	case "lasting", "permanent":
		return machines.Persistent
	}
	return l
}

func (s *Store) Migrate(context.Context) error {
	// invariant: a record written before version 2 is rewritten at it in
	// today's lifetime words, its shares kept as they are; a record at
	// version 2, or one that cannot be read, is left as it is.
	entries, err := s.files.ReadDir(s.dir)
	if err != nil {
		return store.OnlyNewer(err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		at := path.Join(s.dir, e.Name())
		f, err := s.read(at)
		if store.OnlyNewer(err) != nil {
			return err
		}
		if err != nil || f.V >= Version {
			continue
		}
		f.Lifetime = lifetimeAlias(f.Lifetime)
		f.V = Version
		data, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return err
		}
		if err := s.files.WriteFile(at, append(data, '\n'), 0o600); err != nil {
			return err
		}
	}
	return nil
}
