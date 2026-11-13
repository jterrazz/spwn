package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/machinefile"
)

type Ledger interface {
	// invariant: what station installed for a Machinefile, by item name; an
	// install recorded again replaces the earlier record of its item.
	List(ctx context.Context) ([]machinefile.Installed, error)
	Put(ctx context.Context, in machinefile.Installed) error
	// invariant: deleting an item never recorded changes nothing.
	Delete(ctx context.Context, item string) error
	// invariant: the last night the engine's upgrade recorded, as the last
	// Write left it; none written reads as none (false) with no error.
	Night(ctx context.Context) (machinefile.Night, bool, error)
	Write(ctx context.Context, n machinefile.Night) error
	// invariant: the blocks of ~/.zshrc apply wrote, by id; a block written
	// again replaces the earlier record of its id, and deleting an id never
	// recorded changes nothing.
	Blocks(ctx context.Context) ([]machinefile.Written, error)
	PutBlock(ctx context.Context, w machinefile.Written) error
	DeleteBlock(ctx context.Context, id string) error
}

type Owned interface {
	// invariant: the bytes of a Machinefile the owner names by its path; a
	// path that holds no file answers an error wrapping fs.ErrNotExist.
	Read(ctx context.Context, path string) ([]byte, error)
	// invariant: a Machinefile kept in the owner's folder as
	// machinefiles/<name>.yaml, read back by the owner's source; keeping
	// the same bytes again changes nothing, and other bytes over a file
	// already there are refused with *KeptError, the owner's file left as
	// it is.
	Keep(ctx context.Context, name string, body []byte) error
}

// value: a Machinefile of the owner's folder holding other bytes than the
// file named by its path: where it is.

type KeptError struct{ Name, Path string }

func (e *KeptError) Error() string { return e.Path + " already holds another " + e.Name }
