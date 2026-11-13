package contract

import (
	"context"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type ReservationStore interface {
	// invariant: a port never reserved answers store.ErrNotFound.
	Get(ctx context.Context, vm string, guestPort int) (ports.Reservation, error)
	// invariant: reserving a VM's port again replaces its reservation; a
	// reader never sees half the store.
	Put(ctx context.Context, r ports.Reservation) error
	// invariant: every reservation, by VM then guest port.
	List(ctx context.Context) ([]ports.Reservation, error)
	// invariant: deleting a port never reserved changes nothing.
	Delete(ctx context.Context, vm string, guestPort int) error
}
