package fake

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

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
)

// value: where a record is kept, as a refusal names it; a file of the
// Mac's files under `images/` is a record no read has taken in yet.
const (
	shownDir = "~/.local/state/station/images"
	filedDir = "images"
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

// invariant: an image's versions as the jsonfile store reads them, under
// the world's lock: a file edited by hand is taken in once it reads, and a
// record that names no known distro reads as its refusal, naming its file.
func (s *Store) held(image string) ([]images.Record, []*images.OldRecordError, error) {
	files := s.w.Mac(s.mac).Files
	for _, name := range slices.Sorted(maps.Keys(files)) {
		dir, file := path.Split(name)
		if path.Clean(dir) != path.Join(filedDir, image) || !strings.HasSuffix(file, ".json") {
			continue
		}
		var r images.Record
		if err := json.Unmarshal([]byte(files[name]), &r); err != nil {
			return nil, nil, fmt.Errorf("image record %s: %w", path.Join(shownDir, image, file), err)
		}
		r.Name, r.Version = image, strings.TrimSuffix(file, ".json")
		all := s.records()
		*all = append(slices.DeleteFunc(*all, same(r.Name, r.Version)), r)
		delete(files, name)
	}
	var out []images.Record
	var old []*images.OldRecordError
	for _, r := range *s.records() {
		switch {
		case r.Name != image:
		case !r.Distro.Known():
			old = append(old, &images.OldRecordError{Image: image, Version: r.Version, File: path.Join(shownDir, image, r.Version+".json")})
		default:
			out = append(out, r)
		}
	}
	return out, old, nil
}

func (s *Store) Get(_ context.Context, image, version string) (images.Record, error) {
	s.w.Lock()
	defer s.w.Unlock()
	all, old, err := s.held(image)
	if err != nil {
		return images.Record{}, err
	}
	if i := slices.IndexFunc(old, func(e *images.OldRecordError) bool { return e.Version == version }); i >= 0 {
		return images.Record{}, old[i]
	}
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
	// invariant: a record of an image built from a Machinefile is put once
	// its build baked it: the Machinefiles this Mac holds then are what that
	// image carries, whatever is edited since.
	if r.Machinefile != "" && !r.Fetched {
		m := s.w.Mac(s.mac)
		files := map[string]string{}
		for at, body := range m.Files {
			if strings.HasPrefix(at, "machinefiles/") {
				files[at] = body
			}
		}
		if m.Built == nil {
			m.Built = map[string]map[string]string{}
		}
		m.Built[r.Machinefile] = files
	}
	return nil
}

func (s *Store) List(_ context.Context, image string) ([]images.Record, error) {
	s.w.Lock()
	defer s.w.Unlock()
	out, old, err := s.held(image)
	if err != nil {
		return nil, err
	}
	if len(old) > 0 {
		slices.SortFunc(old, func(a, b *images.OldRecordError) int { return strings.Compare(a.Version, b.Version) })
		return nil, old[0]
	}
	slices.SortFunc(out, func(a, b images.Record) int { return images.CompareVersions(b.Version, a.Version) })
	return out, nil
}

func (s *Store) Names(_ context.Context) ([]string, error) {
	s.w.Lock()
	defer s.w.Unlock()
	seen := map[string]bool{}
	for _, r := range *s.records() {
		seen[r.Name] = true
	}
	for name := range s.w.Mac(s.mac).Files {
		dir, file := path.Split(name)
		if parent, image := path.Split(path.Clean(dir)); path.Clean(parent) == filedDir && strings.HasSuffix(file, ".json") {
			seen[image] = true
		}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

func (s *Store) Delete(_ context.Context, image, version string) error {
	s.w.Lock()
	defer s.w.Unlock()
	all := s.records()
	*all = slices.DeleteFunc(*all, same(image, version))
	delete(s.w.Mac(s.mac).Files, path.Join(filedDir, image, version+".json"))
	return nil
}
