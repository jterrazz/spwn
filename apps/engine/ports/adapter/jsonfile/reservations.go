package jsonfile

import (
	"context"
	"slices"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
)

// value: alias of station.leases/v1, its schema and its key, until its store
// migrates to reservations.json (ARCHITECTURE §12).
const (
	ReservationsSchema = "station.leases/v1"
	reservationsKey    = "leases"
)

type Reservations struct{ list listFile[ports.Reservation] }

var (
	_ contract.ReservationStore = (*Reservations)(nil)
	_ store.Migrator            = (*Reservations)(nil)
)

func NewReservations(files Files, file string) *Reservations {
	return &Reservations{list: listFile[ports.Reservation]{files: files, path: file, schema: ReservationsSchema, key: reservationsKey, version: 1}}
}

func (r *Reservations) Get(_ context.Context, vm string, guestPort int) (ports.Reservation, error) {
	all, err := r.list.load()
	if err != nil {
		return ports.Reservation{}, err
	}
	if i := slices.IndexFunc(all, func(x ports.Reservation) bool { return x.VM == vm && x.GuestPort == guestPort }); i >= 0 {
		return all[i], nil
	}
	return ports.Reservation{}, store.ErrNotFound
}

func (r *Reservations) Put(_ context.Context, res ports.Reservation) error {
	all, err := r.list.load()
	if err != nil {
		return err
	}
	all = slices.DeleteFunc(all, func(x ports.Reservation) bool { return x.VM == res.VM && x.GuestPort == res.GuestPort })
	return r.save(append(all, res))
}

func (r *Reservations) List(context.Context) ([]ports.Reservation, error) {
	all, err := r.list.load()
	if err != nil {
		return nil, err
	}
	sortReservations(all)
	return all, nil
}

func (r *Reservations) Delete(_ context.Context, vm string, guestPort int) error {
	all, err := r.list.load()
	if err != nil {
		return err
	}
	kept := slices.DeleteFunc(slices.Clone(all), func(x ports.Reservation) bool { return x.VM == vm && x.GuestPort == guestPort })
	if len(kept) == len(all) {
		return nil
	}
	return r.save(kept)
}

func (r *Reservations) Migrate(context.Context) error {
	_, err := r.list.load()
	return store.OnlyNewer(err)
}

func (r *Reservations) save(all []ports.Reservation) error {
	sortReservations(all)
	return r.list.save(all)
}

func sortReservations(all []ports.Reservation) {
	slices.SortFunc(all, func(a, b ports.Reservation) int {
		if a.VM != b.VM {
			if a.VM < b.VM {
				return -1
			}
			return 1
		}
		return a.GuestPort - b.GuestPort
	})
}
