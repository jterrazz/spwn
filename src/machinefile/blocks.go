package machinefile

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
)

// value: one identified block of ~/.zshrc as the plan judges it: the item
// whose method writes it, its id, the zsh apply would write once the
// variables are filled, what the file holds of it (present as it would be
// written, absent, outdated when it is still what station last wrote but
// station writes it otherwise now, changed since by hand, or extra when its
// item is listed no longer), its diff from the file to what apply writes,
// and what apply does with it: set it, keep it, leave a changed one as it
// is, or, with `--remove`, take back the block of an item the plan carries
// no longer.

type Block struct {
	Item      string
	ID        string
	Zsh       string
	Condition Condition
	Action    Action
	Diff      string
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
