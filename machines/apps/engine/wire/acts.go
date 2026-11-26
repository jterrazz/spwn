package wire

import (
	"errors"
	"io/fs"
	"path"

	"github.com/jterrazz/studio-station/apps/engine/core/guide"
	"github.com/jterrazz/studio-station/apps/engine/engine/feeds"
	"github.com/jterrazz/studio-station/apps/engine/engine/process"
	"github.com/jterrazz/studio-station/apps/engine/engine/readers"
	"github.com/jterrazz/studio-station/apps/engine/engine/store"
)

// invariant: the engine's act board, kept beside its stop mark and
// put in the store as This Mac's acts slice; each write also makes the machines
// slice due, so a create's step reaches a watcher of either at once, never
// at the next tick.
func (m machine) board(state *readers.Set) *process.Acts {
	key := store.Of(store.FamilyActs, m.where.Machine)
	file := actFile{fs: m.sys.FS, name: path.Join(m.paths.State, "acts.json")}
	return process.OpenActs(file, m.processes(), m.sys.Clock.Now, func(a guide.Acts) error {
		err := state.Store().Put(store.Put{Key: key, Value: a, ReadAt: m.sys.Clock.Now()})
		state.Due(readers.NameMac)
		return err
	}, firstGuide)
}

// invariant: an act just put has no record its projection reads yet, so
// its first Guide is projected from the act alone.
func firstGuide(a guide.Act) (guide.Guide, bool) {
	project, known := feeds.Guided(a.Family)
	if !known {
		return guide.Guide{}, false
	}
	return project(a, nil), true
}

// invariant: a world's door onto This Mac's engine keeps the acts it runs
// on the engine's own board file, as the socket would; the engine of the
// next run reads them there.
func (m machine) doorBoard() *process.Acts {
	file := actFile{fs: m.sys.FS, name: path.Join(m.paths.State, "acts.json")}
	return process.OpenActs(file, m.processes(), m.sys.Clock.Now, func(guide.Acts) error { return nil }, firstGuide)
}

type actFiles interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm fs.FileMode) error
	MkdirAll(name string, perm fs.FileMode) error
}

type actFile struct {
	fs   actFiles
	name string
}

func (f actFile) Read() ([]byte, error) {
	data, err := f.fs.ReadFile(f.name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func (f actFile) Write(data []byte) error {
	if err := f.fs.MkdirAll(path.Dir(f.name), 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}
	return f.fs.WriteFile(f.name, data, 0o600)
}
