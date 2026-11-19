package pages

import (
	"time"

	pagemenubar "github.com/jterrazz/studio-station/apps/engine/pages/menubar"
)

// value: home's reading as the engine's feed holds it, each machine's
// sign-ins and grants, for the Find index.
func (f Scene) find() pagemenubar.FindReading {
	r := f.reading()
	r.SignIns = f.signIns(true)
	return pagemenubar.FindReading{Reading: r, Now: storyNow, Zone: time.UTC, Asks: Levels}
}

// value: the Find stories' worlds: home's machines, each Mac with its free
// memory, This Mac awake with no term yet; then the same with infra down.
func findWorlds() []World {
	find := func(name string, edits ...func(*Scene)) World {
		return World{Name: name, Edit: func(f *Scene) {
			homeScene(f)
			f.Awake.Session = nil
			for _, e := range edits {
				e(f)
			}
		}, Pages: []string{"find"}}
	}
	return []World{
		find("machines.find"),
		find("machines.find.down", infraDown),
	}
}
