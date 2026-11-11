package fake

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/src/machinefile"
	"github.com/jterrazz/studio-station/src/machinefile/contract"
	"github.com/jterrazz/studio-station/src/world"
)

type Source struct {
	w *world.World
	// value: the Mac whose files are the owner's; empty for a shared one.
	mac    string
	shared contract.SharedSource
}

func UserOf(w *world.World, mac string) Source {
	// value: the owner's files of a Mac: its station files under catalog/ and
	// machinefiles/, as `world edit` gives them.
	return Source{w: w, mac: mac}
}

func SharedOf(w *world.World, src contract.SharedSource) Source {
	// value: the shared source over the world's repository at its URL.
	return Source{w: w, shared: src}
}

var errNoCommit = errors.New("no such commit in the repository")

func (s Source) Read(context.Context) (machinefile.Tree, error) {
	s.w.Lock()
	defer s.w.Unlock()
	if s.mac != "" {
		return tree(machinefile.User, "~/.config/station", s.w.Mac(s.mac).Files), nil
	}
	repo := s.w.Sources[s.shared.URL]
	if repo == nil || repo.Commits[s.shared.Commit] == nil || s.w.Fails("Source.Read") {
		return machinefile.Tree{}, &contract.UnavailableError{Source: s.shared.Name, Commit: s.shared.Commit, Cause: errNoCommit}
	}
	return tree(machinefile.Shared, s.shared.Name+"@"+s.shared.Commit[:min(7, len(s.shared.Commit))], repo.Commits[s.shared.Commit]), nil
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
