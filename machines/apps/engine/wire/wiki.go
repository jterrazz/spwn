package wire

import (
	"github.com/jterrazz/studio-station/apps/engine/wiki/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

// invariant: a Mac outside a world declares no wiki: the engine names no
// folder to the files adapter until the entry condition of machines/TODO.md,
// "Before a Mac may name its wiki folder", is met; the architecture rule
// no-wiki-folder holds it.
func (m machine) wikiCorpus() contract.Corpus {
	if m.world != nil {
		return m.world.wiki()
	}
	return files.New("")
}
