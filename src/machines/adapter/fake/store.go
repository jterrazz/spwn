package fake

import (
	"context"
	"maps"
	"slices"

	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/world"
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
		Name: name, Image: k.Image, Version: k.Version, Lifetime: k.Lifetime, Want: k.Want, Network: k.Network, Check: k.Check, Task: k.Task,
		Env: k.Env, CPUs: v.Size.CPUs, MemoryBytes: v.Size.Memory, DiskBytes: v.Size.Disk, CreatedAt: k.CreatedAt, Birth: k.Birth,
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

func (s *Store) Put(_ context.Context, r machines.Record) error {
	s.w.Lock()
	defer s.w.Unlock()
	v := s.vms()[r.Name]
	if v == nil {
		v = &world.VM{}
		s.vms()[r.Name] = v
	}
	v.Kept = &world.Kept{
		Image: r.Image, Version: r.Version, Lifetime: r.Lifetime, Want: r.Want, Network: r.Network, Check: r.Check, Task: r.Task, Env: r.Env,
		CreatedAt: r.CreatedAt, Birth: r.Birth,
	}
	v.Size = machines.Size{CPUs: r.CPUs, Memory: r.MemoryBytes, Disk: r.DiskBytes}
	return nil
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
