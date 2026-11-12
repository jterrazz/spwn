package fake

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Source struct {
	w   *world.World
	mac string
}

func UserOf(w *world.World, mac string) Source {
	// value: the owner's files of a Mac: its station files under catalog/ and
	// machinefiles/, as `world edit` gives them.
	return Source{w: w, mac: mac}
}

func (s Source) Read(context.Context) (machinefile.Tree, error) {
	s.w.Lock()
	defer s.w.Unlock()
	return tree(machinefile.User, "~/.config/station", s.w.Mac(s.mac).Files), nil
}

type Shared struct {
	w *world.World
	// value: the Mac whose clone of the shared source is read.
	mac string
	src contract.SharedSource
}

func SharedOf(w *world.World, mac string, src contract.SharedSource) Shared {
	// value: the shared source over the world's repository at its URL, read
	// from the Mac's clone of it, cloned at the repository's head when the
	// Mac has none.
	return Shared{w: w, mac: mac, src: src}
}

var errNoRepository = errors.New("no repository at its url")

func (s Shared) Read(context.Context) (machinefile.Tree, error) {
	s.w.Lock()
	defer s.w.Unlock()
	at, err := s.clone()
	if err != nil {
		return machinefile.Tree{}, err
	}
	t := tree(machinefile.Shared, s.src.Name+"@"+at[:min(7, len(at))], s.w.Sources[s.src.URL].Commits[at])
	t.Source = &catalog.LockSource{Name: s.src.Name, Commit: at}
	return t, nil
}

func (s Shared) Pull(context.Context) error {
	s.w.Lock()
	defer s.w.Unlock()
	if _, err := s.clone(); err != nil {
		return err
	}
	s.w.Mac(s.mac).Clones[s.src.URL] = s.w.Sources[s.src.URL].Head
	return nil
}

func (s Shared) clone() (string, error) {
	repo := s.w.Sources[s.src.URL]
	if repo == nil || repo.Commits[repo.Head] == nil || s.w.Fails("Source.Read") {
		return "", &contract.UnavailableError{Source: s.src.Name, Cause: errNoRepository}
	}
	m := s.w.Mac(s.mac)
	if m.Clones == nil {
		m.Clones = map[string]string{}
	}
	if _, ok := m.Clones[s.src.URL]; !ok {
		m.Clones[s.src.URL] = repo.Head
	}
	return m.Clones[s.src.URL], nil
}

func tree(layer machinefile.Layer, root string, files map[string]string) machinefile.Tree {
	t := machinefile.Tree{Layer: layer, Root: root}
	for p, body := range files {
		if strings.HasPrefix(p, "catalog/") || strings.HasPrefix(p, "machinefiles/") {
			t.Files = append(t.Files, machinefile.File{Path: p, Body: []byte(body)})
		}
	}
	slices.SortFunc(t.Files, func(a, b machinefile.File) int { return strings.Compare(a.Path, b.Path) })
	return t
}
