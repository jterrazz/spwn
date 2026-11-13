package fake

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/jterrazz/studio-station/apps/engine/core/store"
	"github.com/jterrazz/studio-station/apps/engine/ports"
	"github.com/jterrazz/studio-station/apps/engine/ports/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
)

var (
	errScripted   = errors.New("the tailnet refused, as the world scripted it")
	errSharingOff = errors.New("sharing is off on this tailnet node")
)

type Reservations struct {
	w   *world.World
	mac string
}

var (
	_ contract.ReservationStore = (*Reservations)(nil)
	_ contract.DeclarationStore = (*Declarations)(nil)
)

func ReservationsOf(w *world.World, mac string) *Reservations { return &Reservations{w: w, mac: mac} }

func NewReservations() *Reservations { return ReservationsOf(alone(), thisMac) }

func (r *Reservations) all() *[]ports.Reservation { return &r.w.Mac(r.mac).Store().Reservations }

func (r *Reservations) Get(_ context.Context, vm string, guestPort int) (ports.Reservation, error) {
	r.w.Lock()
	defer r.w.Unlock()
	for _, x := range *r.all() {
		if x.VM == vm && x.GuestPort == guestPort {
			return x, nil
		}
	}
	return ports.Reservation{}, store.ErrNotFound
}

func (r *Reservations) Put(_ context.Context, res ports.Reservation) error {
	r.w.Lock()
	defer r.w.Unlock()
	all := r.all()
	*all = append(slices.DeleteFunc(*all, func(x ports.Reservation) bool { return x.VM == res.VM && x.GuestPort == res.GuestPort }), res)
	return nil
}

func (r *Reservations) List(context.Context) ([]ports.Reservation, error) {
	r.w.Lock()
	defer r.w.Unlock()
	out := slices.Clone(*r.all())
	slices.SortFunc(out, func(a, b ports.Reservation) int {
		if c := strings.Compare(a.VM, b.VM); c != 0 {
			return c
		}
		return a.GuestPort - b.GuestPort
	})
	return out, nil
}

func (r *Reservations) Delete(_ context.Context, vm string, guestPort int) error {
	r.w.Lock()
	defer r.w.Unlock()
	all := r.all()
	*all = slices.DeleteFunc(*all, func(x ports.Reservation) bool { return x.VM == vm && x.GuestPort == guestPort })
	return nil
}

func (r *Reservations) Migrate(context.Context) error { return nil }

type Declarations struct {
	w   *world.World
	mac string
}

func DeclarationsOf(w *world.World, mac string) *Declarations { return &Declarations{w: w, mac: mac} }

func NewDeclarations() *Declarations { return DeclarationsOf(alone(), thisMac) }

func (d *Declarations) all() *[]ports.Declaration { return &d.w.Mac(d.mac).Store().Declarations }

func (d *Declarations) Get(_ context.Context, machine string, port int) (ports.Declaration, error) {
	d.w.Lock()
	defer d.w.Unlock()
	for _, x := range *d.all() {
		if x.Machine == machine && x.Port == port {
			return x, nil
		}
	}
	return ports.Declaration{}, store.ErrNotFound
}

func (d *Declarations) Put(_ context.Context, decl ports.Declaration) error {
	d.w.Lock()
	defer d.w.Unlock()
	all := d.all()
	*all = append(slices.DeleteFunc(*all, func(x ports.Declaration) bool { return x.Machine == decl.Machine && x.Port == decl.Port }), decl)
	return nil
}

func (d *Declarations) List(context.Context) ([]ports.Declaration, error) {
	d.w.Lock()
	defer d.w.Unlock()
	out := slices.Clone(*d.all())
	slices.SortFunc(out, func(a, b ports.Declaration) int {
		if c := strings.Compare(a.Machine, b.Machine); c != 0 {
			return c
		}
		return a.Port - b.Port
	})
	return out, nil
}

func (d *Declarations) Delete(_ context.Context, machine string, port int) error {
	d.w.Lock()
	defer d.w.Unlock()
	all := d.all()
	*all = slices.DeleteFunc(*all, func(x ports.Declaration) bool { return x.Machine == machine && x.Port == port })
	return nil
}

func (d *Declarations) Migrate(context.Context) error { return nil }
