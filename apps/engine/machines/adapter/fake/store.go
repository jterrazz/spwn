package fake

import (
	"context"
	"maps"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Store struct {
	// invariant: the VM records one Mac of the world keeps, beside its disks;
	// a record deleted takes its account copies, which live in its file.
	w   *world.World
	mac string
}

var (
	_ contract.MachineStore = (*Store)(nil)
	_ store.Migrator        = (*Store)(nil)
)

func NewStore() *Store {
	// value: a store over a world of its own, for a test that needs no other fake.
	return StoreOf(&world.World{}, "this-mac")
}

func StoreOf(w *world.World, mac string) *Store { return &Store{w: w, mac: mac} }

func (s *Store) vms() map[string]*world.VM { return s.w.Mac(s.mac).VMs }

func record(name string, v *world.VM) machines.Record {
	k := v.Kept
	return machines.Record{
		Name: name, Image: k.Image, Version: k.Version, Lifetime: k.Lifetime, Want: k.Want, Network: k.Network, Check: k.Check,
		Env: k.Env, CPUs: v.Size.CPUs, MemoryBytes: v.Size.Memory, DiskBytes: v.Size.Disk, CreatedAt: k.CreatedAt, Birth: k.Birth, Owner: k.Owner, Agents: k.Agents, Template: k.Template, Mounts: k.Mounts,
		Purpose: k.Purpose, Creator: k.Creator,
	}
}

func (s *Store) Get(_ context.Context, name string) (machines.Record, error) {
	s.w.Lock()
	defer s.w.Unlock()
	v, ok := s.vms()[name]
	if !ok || v.Kept == nil {
		return machines.Record{}, store.ErrNotFound
	}
	return record(name, v), nil
}

func (s *Store) Update(_ context.Context, name string, change func(r *machines.Record)) error {
	// invariant: the world's lock is every process's of a run, so an
	// update holds it from its read to its write.
	s.w.Lock()
	defer s.w.Unlock()
	r := machines.Record{Name: name}
	if v, ok := s.vms()[name]; ok && v.Kept != nil {
		r = record(name, v)
	}
	change(&r)
	s.put(r)
	return nil
}

func (s *Store) Amend(_ context.Context, name string, change func(r *machines.Record)) error {
	s.w.Lock()
	defer s.w.Unlock()
	v, ok := s.vms()[name]
	if !ok || v.Kept == nil {
		return store.ErrNotFound
	}
	r := record(name, v)
	change(&r)
	s.put(r)
	return nil
}

// invariant: the claims of every world of this process take turns, so
// judge may read the world, whose lock it does not hold, and no claim
// writes between another's read and its write.
var claims = make(chan struct{}, 1)

func (s *Store) Claim(ctx context.Context, judge func(held []machines.Record) (machines.Record, error)) error {
	select {
	case claims <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-claims }()
	held, err := s.List(ctx)
	if err != nil {
		return err
	}
	r, err := judge(held)
	if err != nil {
		return err
	}
	return s.Put(ctx, r)
}

func (s *Store) Put(_ context.Context, r machines.Record) error {
	s.w.Lock()
	defer s.w.Unlock()
	s.put(r)
	return nil
}

func (s *Store) put(r machines.Record) {
	v := s.vms()[r.Name]
	if v == nil {
		v = &world.VM{}
		s.vms()[r.Name] = v
	}
	agents := r.Agents
	if agents == nil && v.Kept != nil {
		agents = v.Kept.Agents
	}
	v.Kept = &world.Kept{
		Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Env: r.Env,
		CreatedAt: r.CreatedAt, Birth: r.Birth, Owner: r.Owner, Agents: agents, Template: r.Template, Mounts: r.Mounts,
		Purpose: r.Purpose, Creator: r.Creator,
	}
	v.Size = machines.Size{CPUs: r.CPUs, Memory: r.MemoryBytes, Disk: r.DiskBytes}
}

func (s *Store) List(context.Context) ([]machines.Record, error) {
	s.w.Lock()
	defer s.w.Unlock()
	out := []machines.Record{}
	for _, name := range slices.Sorted(maps.Keys(s.vms())) {
		if v := s.vms()[name]; v.Kept != nil {
			out = append(out, record(name, v))
		}
	}
	return out, nil
}

func (s *Store) ListReadable(ctx context.Context) ([]machines.Record, error) { return s.List(ctx) }

func (s *Store) Delete(_ context.Context, name string) error {
	s.w.Lock()
	defer s.w.Unlock()
	v, ok := s.vms()[name]
	if !ok {
		return nil
	}
	v.Kept = nil
	if !v.Disk {
		delete(s.vms(), name)
	}
	return nil
}

func (s *Store) Migrate(context.Context) error { return nil }
