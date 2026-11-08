package pages

import (
	"time"

	pagemenubar "github.com/jterrazz/studio-station/src/pages/menubar"
)

// value: home's reading as the engine's feed holds it, each Mac's own
// accounts with the copies it shared, for the Find index.
func (f Scene) find() pagemenubar.FindReading {
	r := f.reading()
	r.Accounts = f.needs().Accounts
	return pagemenubar.FindReading{Reading: r, Now: storyNow, Zone: time.UTC, Asks: Levels}
}
