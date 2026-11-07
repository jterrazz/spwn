package contract

import (
	"context"

	"github.com/jterrazz/studio-station/src/machines"
)

type UptimeStore interface {
	// invariant: a server never marked has an empty record, and no error.
	Get(ctx context.Context, server string) (machines.Uptime, error)
	// invariant: putting a server's record again replaces it whole, and
	// keeps every other server's.
	Put(ctx context.Context, server string, u machines.Uptime) error
	// invariant: deleting a server never marked changes nothing.
	Delete(ctx context.Context, server string) error
}
