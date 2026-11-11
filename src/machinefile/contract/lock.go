package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/catalog"
)

type Locks interface {
	// invariant: the lock of a Machinefile on this machine, as the last
	// Write left it; one never written reads as none (false) with no
	// error, and one that does not decode is refused at its file.
	Read(ctx context.Context, name string) (catalog.LockFile, bool, error)
	// invariant: a lock is written whole, so a run that stops halfway
	// leaves the one before or the one after, never a mix.
	Write(ctx context.Context, l catalog.LockFile) error
	// value: where the owner reads the lock of a Machinefile.
	Where(name string) string
}
