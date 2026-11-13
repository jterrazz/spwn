package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/accounts"
)

type ShareStore interface {
	// invariant: a VM with nothing shared answers no shares and no error.
	Get(ctx context.Context, vm string) ([]accounts.Share, error)
	// invariant: putting a VM's shares replaces them, putting none removes
	// them, and the VM's record beside them is kept; a VM with no record
	// cannot be handed a share.
	Put(ctx context.Context, vm string, shares []accounts.Share) error
	// invariant: every VM holding a share, by VM.
	List(ctx context.Context) (map[string][]accounts.Share, error)
	// invariant: deleting the shares of a VM holding none changes nothing.
	Delete(ctx context.Context, vm string) error
}
