package jsonfile

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"
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
	Stat(path string) (fs.FileInfo, error)
}

type Store struct {
	files Files
	dir   string
	// invariant: one turn of this process at a time holds the records,
	// waited for as long as ctx lasts.
	turn chan struct{}

	mu     sync.Mutex
	listed [2]*listing
}

// invariant: a listing is what the folder held when it was read, with what
// stat said of the folder and of each record before it was read; it answers
// again, as a copy, while no store of this process wrote since and every stat
// is the same, so an engine that lists the records each second opens none.
type listing struct {
	gen   uint64
	dir   store.Stamp
	files map[string]store.Stamp
	out   []machines.Record
}

var (
	_ contract.MachineStore = (*Store)(nil)
)

func New(files Files, dir string) *Store {
	return &Store{files: files, dir: dir, turn: make(chan struct{}, 1)}
}

func (s *Store) recordFile(name string) string { return path.Join(s.dir, name+".json") }

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
	CreatedAt   time.Time         `json:"created_at"`
	Birth       *machines.Birth   `json:"birth,omitempty"`
	Owner       *machines.Process `json:"owner,omitempty"`
	Template    string            `json:"template,omitempty"`
	Mounts      []machines.Mount  `json:"mounts,omitempty"`
	Purpose     string            `json:"purpose,omitempty"`
	Creator     string            `json:"creator,omitempty"`
	V           int               `json:"v,omitempty"`
}

func (f file) record() machines.Record {
	return machines.Record{
		Name: f.Name, Image: f.Image, Version: f.Version, Lifetime: f.Lifetime, Want: f.Want, Network: f.Network, Check: f.Check, Env: f.Env,
		CPUs: f.CPUs, MemoryBytes: f.MemoryBytes, DiskBytes: f.DiskBytes, CreatedAt: f.CreatedAt, Birth: f.Birth, Owner: f.Owner, Template: f.Template, Mounts: f.Mounts,
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
	if err := store.Current(f.V, Version); err != nil {
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
	data, err := json.MarshalIndent(file{
		Name: r.Name, Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		CPUs: r.CPUs, MemoryBytes: r.MemoryBytes, DiskBytes: r.DiskBytes, CreatedAt: r.CreatedAt,
		Birth: r.Birth, Owner: r.Owner, Template: r.Template, Mounts: r.Mounts, Purpose: r.Purpose, Creator: r.Creator, V: Version,
	}, "", "  ")
	if err != nil {
		return err
	}
	err = s.files.WriteFile(s.recordFile(r.Name), append(data, '\n'), 0o600)
	store.Wrote()
	return err
}

func (s *Store) List(context.Context) ([]machines.Record, error) { return s.list(false) }

func (s *Store) ListReadable(context.Context) ([]machines.Record, error) { return s.list(true) }

func (s *Store) list(skipDamaged bool) ([]machines.Record, error) {
	which := 0
	if skipDamaged {
		which = 1
	}
	gen := store.Written()
	var dir store.Stamp
	info, statErr := s.files.Stat(s.dir)
	if statErr == nil {
		dir = store.StampOf(info)
	}
	s.mu.Lock()
	kept := s.listed[which]
	s.mu.Unlock()
	if statErr == nil && kept != nil && kept.gen == gen && kept.dir == dir && s.unchanged(kept.files) {
		return copyRecords(kept.out), nil
	}
	out, stamps, err := s.readAll(skipDamaged)
	if err != nil || statErr != nil || stamps == nil {
		return out, err
	}
	kept = &listing{gen: gen, dir: dir, files: stamps, out: copyRecords(out)}
	s.mu.Lock()
	s.listed[which] = kept
	s.mu.Unlock()
	return out, nil
}

func (s *Store) unchanged(files map[string]store.Stamp) bool {
	for at, was := range files {
		info, err := s.files.Stat(at)
		if err != nil || store.StampOf(info) != was {
			return false
		}
	}
	return true
}

func copyRecords(in []machines.Record) []machines.Record {
	if in == nil {
		return nil
	}
	out := make([]machines.Record, len(in))
	for i, r := range in {
		r.Env = maps.Clone(r.Env)
		r.Mounts = slices.Clone(r.Mounts)
		if r.Birth != nil {
			b := *r.Birth
			b.LastLines = slices.Clone(b.LastLines)
			r.Birth = &b
		}
		if r.Owner != nil {
			o := *r.Owner
			r.Owner = &o
		}
		out[i] = r
	}
	return out
}

func (s *Store) readAll(skipDamaged bool) ([]machines.Record, map[string]store.Stamp, error) {
	entries, err := s.files.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, &store.UnreadableError{Path: s.dir, Cause: err}
	}
	var out []machines.Record
	stamps, stamped := map[string]store.Stamp{}, true
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		at := path.Join(s.dir, e.Name())
		if info, err := s.files.Stat(at); err == nil {
			stamps[at] = store.StampOf(info)
		} else {
			stamped = false
		}
		f, err := s.read(at)
		var unreadable *store.UnreadableError
		if skipDamaged && errors.As(err, &unreadable) && store.OnlyNewer(err) == nil {
			continue
		}
		if err != nil {
			if !errors.As(err, &unreadable) {
				err = &store.UnreadableError{Path: at, Cause: err}
			}
			return nil, nil, err
		}
		out = append(out, f.record())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if !stamped {
		stamps = nil
	}
	return out, stamps, nil
}

func (s *Store) Delete(_ context.Context, name string) error {
	err := s.files.Remove(s.recordFile(name))
	store.Wrote()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
