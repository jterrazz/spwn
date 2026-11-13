package fake

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/images"
	"github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

type Store struct {
	// invariant: the image records one Mac of the world keeps.
	w   *world.World
	mac string
}

var (
	_ contract.ImageStore = (*Store)(nil)
	_ store.Migrator      = (*Store)(nil)
)

func StoreOf(w *world.World, mac string) *Store { return &Store{w: w, mac: mac} }

func NewStore() *Store {
	w := &world.World{ThisMac: "this-mac"}
	w.Mac("this-mac")
	return StoreOf(w, "this-mac")
}

func (s *Store) records() *[]images.Record { return &s.w.Mac(s.mac).Store().Images }

func same(name, version string) func(images.Record) bool {
	return func(r images.Record) bool { return r.Name == name && r.Version == version }
}

func (s *Store) Get(_ context.Context, image, version string) (images.Record, error) {
	s.w.Lock()
	defer s.w.Unlock()
	all := *s.records()
	i := slices.IndexFunc(all, same(image, version))
	if i < 0 {
		return images.Record{}, store.ErrNotFound
	}
	return all[i], nil
}

func (s *Store) Put(_ context.Context, r images.Record) error {
	s.w.Lock()
	defer s.w.Unlock()
	all := s.records()
	*all = append(slices.DeleteFunc(*all, same(r.Name, r.Version)), r)
	return nil
}

func (s *Store) List(_ context.Context, image string) ([]images.Record, error) {
	s.w.Lock()
	defer s.w.Unlock()
	var out []images.Record
	for _, r := range *s.records() {
		if r.Name == image {
			out = append(out, r)
		}
	}
	slices.SortFunc(out, func(a, b images.Record) int { return images.CompareVersions(b.Version, a.Version) })
	return out, nil
}

func (s *Store) Delete(_ context.Context, image, version string) error {
	s.w.Lock()
	defer s.w.Unlock()
	all := s.records()
	*all = slices.DeleteFunc(*all, same(image, version))
	return nil
}

func (s *Store) Migrate(context.Context) error { return nil }
