package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/machines"
)

type MachineStore interface {
	// invariant: a VM never put answers store.ErrNotFound.
	Get(ctx context.Context, name string) (machines.Record, error)
	// invariant: putting a VM again replaces its record and keeps what
	// another store keeps beside it; a record never holds a share. A put
	// whose agents are nil keeps the agents its record held.
	Put(ctx context.Context, r machines.Record) error
	// invariant: every VM put, sorted by name.
	List(ctx context.Context) ([]machines.Record, error)
	// invariant: every VM put whose record parses, sorted by name; a record
	// that does not parse is skipped, one a newer station wrote still fails.
	ListReadable(ctx context.Context) ([]machines.Record, error)
	// invariant: deleting a VM never put changes nothing.
	Delete(ctx context.Context, name string) error
}
