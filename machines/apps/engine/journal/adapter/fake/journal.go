package fake

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/core/journal"
	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/journal/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

// value: the name the journal's file has among a Mac's own files, and the
// path a refusal names.
const (
	File = "journal.jsonl"
	Path = "~/.local/state/station/journal.jsonl"
)

type Journal struct {
	// invariant: the lines one Mac of the world keeps.
	w   *world.World
	mac string
}

var (
	_ contract.Journal = (*Journal)(nil)
)

func New() *Journal { return Of(&world.World{}, "this-mac") }

func Of(w *world.World, mac string) *Journal { return &Journal{w: w, mac: mac} }

// invariant: a journal file the world holds as text is read before the
// lines: once it reads, its lines are the journal's; one that does not
// read stops every read and write, left as it is.
func (j *Journal) load() (*world.Stores, error) {
	m := j.w.Mac(j.mac)
	s := m.Store()
	text, ok := m.Files[File]
	if !ok {
		return s, nil
	}
	var lines []journal.Line
	for i, row := range strings.Split(strings.TrimSpace(text), "\n") {
		if i == 0 {
			var head struct{ Schema string }
			if err := json.Unmarshal([]byte(row), &head); err != nil {
				return nil, &store.UnreadableError{Path: Path, Cause: err, Line: 1}
			}
			continue
		}
		var l journal.Line
		if err := json.Unmarshal([]byte(row), &l); err != nil {
			return nil, &store.UnreadableError{Path: Path, Cause: err, Line: i + 1}
		}
		lines = append(lines, l)
	}
	s.Journal = append(s.Journal, lines...)
	delete(m.Files, File)
	return s, nil
}

func (j *Journal) Lines(_ context.Context, since, before time.Time) ([]journal.Line, error) {
	j.w.Lock()
	defer j.w.Unlock()
	s, err := j.load()
	if err != nil {
		return nil, err
	}
	return journal.Between(s.Journal, since, before), nil
}

func (j *Journal) Write(_ context.Context, now time.Time, lines ...journal.Line) error {
	j.w.Lock()
	defer j.w.Unlock()
	s, err := j.load()
	if err != nil {
		return err
	}
	kept := journal.Kept(append(s.Journal, lines...), now)
	s.Journal = make([]journal.Line, 0, len(kept))
	for i := len(kept) - 1; i >= 0; i-- {
		s.Journal = append(s.Journal, kept[i])
	}
	return nil
}
