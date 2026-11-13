package process

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/ports"
)

type noPolicy struct{}

func (noPolicy) Get() (*ports.PortsGroup, error) { return nil, nil }

// value: the reserved ports in memory, as the store keeps them.
type memReservations struct{ all []ports.Reservation }

func (m *memReservations) Get(_ context.Context, vm string, port int) (ports.Reservation, error) {
	for _, r := range m.all {
		if r.VM == vm && r.GuestPort == port {
			return r, nil
		}
	}
	return ports.Reservation{}, context.Canceled
}

func (m *memReservations) Put(ctx context.Context, r ports.Reservation) error {
	_ = m.Delete(ctx, r.VM, r.GuestPort)
	m.all = append(m.all, r)
	return nil
}

func (m *memReservations) List(context.Context) ([]ports.Reservation, error) {
	return slices.Clone(m.all), nil
}

func (m *memReservations) Delete(_ context.Context, vm string, port int) error {
	m.all = slices.DeleteFunc(m.all, func(r ports.Reservation) bool { return r.VM == vm && r.GuestPort == port })
	return nil
}

func memStores() Stores { return Stores{Ports: noPolicy{}, Reservations: &memReservations{}} }
