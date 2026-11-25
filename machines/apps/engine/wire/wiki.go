package wire

import (
	"github.com/jterrazz/studio-station/apps/engine/wiki/adapter/files"
	"github.com/jterrazz/studio-station/apps/engine/wiki/contract"
)

// invariant: a Mac reads the wiki its Machinefile declares; until a
// Machinefile can declare one, a Mac outside a world declares none.
func (m machine) wikiCorpus() contract.Corpus {
	if m.world != nil {
		return m.world.wiki()
	}
	return files.New(m.sys.FS, "")
}
