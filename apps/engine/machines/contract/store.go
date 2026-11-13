package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/machines"
)

type MachineStore interface {
	// invariant: a VM never put answers store.ErrNotFound.
	Get(ctx context.Context, name string) (machines.Record, error)
	// invariant: putting a VM again replaces its record and keeps what
	// another store keeps beside it; a record never holds a share. A put
	// whose agents are nil keeps the agents its record held.
	Put(ctx context.Context, r machines.Record) error
	// invariant: an amend reads a VM's record, changes it and writes it
	// back as one step no other Amend of any process on this Mac
	// interleaves, so a writer that owns one field (a create its birth,
	// the engine its env) never writes back another's stale; it never
	// makes a record: a VM with no record answers store.ErrNotFound,
	// nothing written.
	Amend(ctx context.Context, name string, change func(r *machines.Record)) error
	// invariant: a claim reads every VM's record and writes the one judge
	// answers as one step no other Claim or Amend of any process on
	// this Mac interleaves, so what judge decided from is still so when its
	// record lands; a judge's error writes nothing and is the claim's. A
	// claim that waits for another ends when ctx does, with ctx's error.
	Claim(ctx context.Context, judge func(held []machines.Record) (machines.Record, error)) error
	// invariant: every VM put, sorted by name.
	List(ctx context.Context) ([]machines.Record, error)
	// invariant: every VM put whose record parses, sorted by name; a record
	// that does not parse is skipped, one a newer station wrote still fails.
	ListReadable(ctx context.Context) ([]machines.Record, error)
	// invariant: deleting a VM never put changes nothing.
	Delete(ctx context.Context, name string) error
}
