package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type DeclarationStore interface {
	// invariant: a port never declared answers store.ErrNotFound.
	Get(ctx context.Context, machine string, port int) (ports.Declaration, error)
	// invariant: declaring a machine's port again replaces its route.
	Put(ctx context.Context, d ports.Declaration) error
	// invariant: every declaration, by machine then port.
	List(ctx context.Context) ([]ports.Declaration, error)
	// invariant: deleting a port never declared changes nothing.
	Delete(ctx context.Context, machine string, port int) error
}
