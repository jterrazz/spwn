package machinefile

import "github.com/jterrazz/studio-station/src/catalog"

// value: one identified block of ~/.zshrc as the plan judges it: the item
// whose method writes it, its id, the zsh apply would write once the
// variables are filled, what the file holds of it (present as it would be
// written, absent, or changed since by hand), its diff against the file, and
// what apply does with it: set it, keep it, leave a changed one as it is, or
// remove the block of an item the plan carries no longer.

type Block struct {
	Item      string
	ID        string
	Zsh       string
	Condition Condition
	Action    Action
	Diff      string
}

func (b Block) Acts() bool { return b.Action == Set || b.Action == Remove }

// value: what apply did with a block.

type BlockDone struct {
	Block
	State  catalog.State
	Detail string
}

func (p Plan) WritesBlocks() bool {
	for _, b := range p.Blocks {
		if b.Acts() {
			return true
		}
	}
	return false
}
