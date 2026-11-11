package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/machinefile"
)

type Ledger interface {
	// invariant: what station installed for a Machinefile, by item name; an
	// install recorded again replaces the earlier record of its item.
	List(ctx context.Context) ([]machinefile.Installed, error)
	Put(ctx context.Context, in machinefile.Installed) error
	// invariant: deleting an item never recorded changes nothing.
	Delete(ctx context.Context, item string) error
}

type Owned interface {
	// invariant: the bytes of a Machinefile the owner names by its path; a
	// path that holds no file answers an error wrapping fs.ErrNotExist.
	Read(ctx context.Context, path string) ([]byte, error)
	// invariant: a Machinefile kept in the owner's folder as
	// machinefiles/<name>.yaml, read back by the owner's source; keeping it
	// again replaces it.
	Keep(ctx context.Context, name string, body []byte) error
}
