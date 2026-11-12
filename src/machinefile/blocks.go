package machinefile

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: one identified block of ~/.zshrc as the plan judges it: the item
// whose method writes it, its id, the zsh apply would write once the
// variables are filled (for an extra, what the file holds), what the file
// holds of it (present as it would be written, absent, outdated when it is
// still what station last wrote but station writes it otherwise now,
// changed since by hand, or extra when station wrote it for an extra of the
// plan), its diff from the file to what apply writes, and what apply does
// with it: set it, keep it, leave a changed one as it is, or, with
// `--remove`, take it back with its item. An extra's Fate is what `apply
// --remove` does with it.

type Block struct {
	Item      string
	ID        string
	Zsh       string
	Condition Condition
	Action    Action
	Diff      string
	Fate      Fate
}

func (b Block) Forgets() bool {
	// value: whether `apply --remove` takes back the block, or lets go of
	// its record as the owner's.
	return b.Condition == Extra && b.Fate.Forgets()
}

// value: a block of ~/.zshrc apply wrote: its id, the item it was written
// for, the digest of the text written between its markers, and when.

type Written struct {
	ID     string    `json:"id"`
	Item   string    `json:"item"`
	Digest string    `json:"digest"`
	At     time.Time `json:"at"`
}

func BlockDigest(body string) string {
	// value: the fingerprint of a block's text, as the ledger records it.
	sum := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// value: what apply did with a block.

type BlockDone struct {
	Block
	State  catalog.State
	Detail string
}

func (p Plan) WritesBlocks() bool {
	for _, b := range p.Blocks {
		if b.Action == Set {
			return true
		}
	}
	return false
}

func (p Plan) RemovesBlocks() bool {
	for _, b := range p.Blocks {
		if b.Action == Remove {
			return true
		}
	}
	return false
}
