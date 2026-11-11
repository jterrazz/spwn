package calls

import (
	"context"
	"errors"
	"github.com/jterrazz/studio-station/src/ports/contract"
	"reflect"
	"slices"
	"sort"

	machinescontract "github.com/jterrazz/studio-station/src/machines/contract"
	setupcontract "github.com/jterrazz/studio-station/src/setup/contract"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/store"
	"github.com/jterrazz/studio-station/src/ports"
)

type Engine interface {
	MacPorts(ctx context.Context) (ports.MacPorts, error)
	Reserve(ctx context.Context, vm string, port int) (ports.SharedPort, error)
	Release(ctx context.Context, vm string, port int) (ports.Reservation, error)
}

type ReserverOptions struct {
	ThisMachine catalog.MachineKind
	State       string
	ThisMacName string
	VMName      string
	MacName     string
	// value: each other host's own services, from those that answered.
	Ask func(ctx context.Context, macs []string) map[string]ports.MacPorts
	// value: a VM's port released by the host it runs on.
	ReleaseOn func(ctx context.Context, mac string, t ports.Target, yes bool) (ports.Reservation, error)
	VMMac     func(ctx context.Context, vm string) (mac string, thisMac bool, err error)
	Exposure  func(ctx context.Context) ports.Exposure
	// value: whether this Mac's engine is installed, read when it does not
	// answer.
	EngineInstalled func() bool
	// value: what listens on this Mac, none where nothing can list it; nil
	// reads nothing.
	Listening func(ctx context.Context) ([]ports.Listener, error)
	// value: the routes a declared port is verified by; nil verifies none.
	Routes contract.PublicRoutes
	// value: the ports the catalog's items serve on: a row's label.
	Ports []catalog.ItemPort
}

type Stores struct {
	Ports        contract.PortsPrefs
	Name         setupcontract.NamePrefs
	Macs         machinescontract.MacsPrefs
	Reservations contract.ReservationStore
	Declarations contract.DeclarationStore
}

type Reserver struct {
	st Stores
	d  Engine
	o  ReserverOptions
}

func NewReserver(st Stores, d Engine, o ReserverOptions) *Reserver {
	return &Reserver{st: st, d: d, o: o}
}

func (s *Reserver) inVM() bool { return s.o.ThisMachine != catalog.Mac }

func (s *Reserver) policy() ports.Policy {
	g, err := s.st.Ports.Get()
	if err != nil {
		return ports.PolicyOf(nil)
	}
	return ports.PolicyOf(g)
}

func (s *Reserver) offline(ctx context.Context) ports.MacPorts {
	e := ports.ExposureOf(false, "")
	if s.o.Exposure != nil {
		e = s.o.Exposure(ctx)
	}
	view := ports.MacPorts{Name: s.o.ThisMacName, ThisMac: true, Reachable: true, PublicName: e.PublicName, Serve: e.Word(), HTTPS: s.policy().HTTPS, Shared: []ports.SharedPort{}}
	reservations, _ := s.st.Reservations.List(ctx)
	for _, l := range reservations {
		view.Shared = append(view.Shared, ports.Hinted(ports.ViewOf(l, e, view.HTTPS, ports.PhaseUnknown), s.o.Ports))
	}
	return view
}

func (s *Reserver) ThisMachine(ctx context.Context) (ports.MacPorts, error) {
	if s.inVM() {
		view, err := s.d.MacPorts(ctx)
		if err != nil {
			return ports.MacPorts{}, &ports.MacAwayError{Mac: s.o.MacName}
		}
		mine := view.Shared[:0:0]
		for _, v := range view.Shared {
			if v.VM == s.o.VMName {
				mine = append(mine, v)
			}
		}
		view.Shared = mine
		view.Ports = slices.DeleteFunc(view.Ports, func(p ports.Port) bool { return p.Machine.Name != s.o.VMName })
		view.Listening = nil
		return view, nil
	}
	view, err := s.d.MacPorts(ctx)
	if errors.Is(err, ports.ErrEngineDown) {
		view, err = s.offline(ctx), nil
		view.Engine.Installed = s.o.EngineInstalled != nil && s.o.EngineInstalled()
	}
	if err != nil {
		return ports.MacPorts{}, err
	}
	view.Name, view.ThisMac, view.Reachable = s.o.ThisMacName, true, true
	view.Range = s.rangeOf()
	ps, err := s.survey(ctx, view.Listening)
	if err != nil {
		return ports.MacPorts{}, err
	}
	view.Ports, view.Listening = ps, nil
	return view, nil
}

// invariant: this Mac's ports and its VMs', each in its class: the Mac's
// read here, its VMs' as its engine heard them, nothing of a VM while the
// engine is stopped. What listens here that cannot be read fails the read,
// as an empty list would say nothing listens; a Mac with no reader of its
// listeners has nothing to say of them.
func (s *Reserver) survey(ctx context.Context, vms []ports.Machine) ([]ports.Port, error) {
	var mine []ports.Listener
	if s.o.Listening != nil {
		var err error
		if mine, err = s.o.Listening(ctx); err != nil {
			return nil, err
		}
	}
	thisMac := ports.MacOf(s.o.ThisMacName, mine)
	var decl []ports.Declaration
	if s.st.Declarations != nil {
		decl, _ = s.st.Declarations.List(ctx)
	}
	ms := append([]ports.Machine{thisMac}, vms...)
	ps := ports.Survey(ms, decl)
	if s.o.Routes == nil {
		return ps, nil
	}
	return s.o.Routes.Verify(ctx, ms, ps), nil
}

// invariant: the range is the host's policy and its lease file, so it reads the
// same with the daemon up or down; a lease left outside a narrowed range is not
// counted against it.
func (s *Reserver) rangeOf() *ports.Range {
	p := s.policy()
	reservations, _ := s.st.Reservations.List(context.Background())
	reserved := 0
	for _, l := range reservations {
		if l.MacPort >= p.Low && l.MacPort <= p.High {
			reserved++
		}
	}
	return &ports.Range{Low: p.Low, High: p.High, Reserved: reserved, Free: p.High - p.Low + 1 - reserved, MaxPerVM: p.MaxPerVM}
}

func (s *Reserver) List(ctx context.Context, thisMacOnly bool) (ports.AllMachines, error) {
	thisMac, err := s.ThisMachine(ctx)
	if err != nil {
		return ports.AllMachines{}, err
	}
	allMachines := ports.AllMachines{Schema: ports.ReservationsSchema, Macs: []ports.MacPorts{thisMac}}
	if thisMacOnly || s.inVM() {
		return allMachines, nil
	}
	macs, err := s.st.Macs.Get()
	if err != nil {
		return ports.AllMachines{}, err
	}
	name, err := s.st.Name.Get()
	if err != nil {
		return ports.AllMachines{}, err
	}
	var others []string
	for mac := range macs {
		if mac != name {
			others = append(others, mac)
		}
	}
	sort.Strings(others)
	var answers map[string]ports.MacPorts
	if s.o.Ask != nil {
		answers = s.o.Ask(ctx, others)
	}
	views := make([]ports.MacPorts, len(others))
	for i, name := range others {
		views[i] = answered(name, answers)
	}
	allMachines.Macs = append(allMachines.Macs, views...)
	return allMachines, nil
}

func answered(name string, answers map[string]ports.MacPorts) ports.MacPorts {
	view, ok := answers[name]
	if !ok {
		return ports.MacPorts{Name: name, Shared: []ports.SharedPort{}}
	}
	view.Name, view.ThisMac, view.Reachable = name, false, true
	if view.Shared == nil {
		view.Shared = []ports.SharedPort{}
	}
	return view
}

func (s *Reserver) Placed() (vm string, inside bool) {
	// value: the VM this binary runs in, as its guest placed it, inside one.
	if !s.inVM() {
		return "", false
	}
	return s.o.VMName, true
}

func (s *Reserver) CheckPort(text string) (int, error) {
	port, err := ports.ParsePort(text)
	if err != nil {
		return 0, err
	}
	// invariant: a share the owner asks for may open any port but the
	// system's (below catalog.SystemPorts) and the VM's own ssh, 22.
	if port < catalog.SystemPorts || port == catalog.GuestSSH {
		return 0, &ports.IgnoredPortError{Port: port}
	}
	return port, nil
}

func (s *Reserver) ParsePort(text string) (int, error) { return ports.ParsePort(text) }

func (s *Reserver) Keeper(ctx context.Context, name string) (named, mac string, err error) {
	// invariant: this Mac keeps its own declarations and its VMs'; another
	// Mac of the fleet keeps its own and its VMs'.
	if name == "" || name == s.o.ThisMacName {
		return s.o.ThisMacName, "", nil
	}
	if macs, read := s.st.Macs.Get(); read == nil {
		if _, ok := macs[name]; ok {
			return name, name, nil
		}
	}
	on, thisMac, err := s.o.VMMac(ctx, name)
	if err != nil || thisMac {
		return name, "", err
	}
	return name, on, nil
}

func (s *Reserver) CheckVM(ctx context.Context, vm string) (mac string, err error) {
	// invariant: a VM of another Mac is shared by that Mac's engine: its
	// name is returned, and its share asked there.
	mac, thisMac, err := s.o.VMMac(ctx, vm)
	if err != nil || thisMac {
		return "", err
	}
	return mac, nil
}

func (s *Reserver) Share(ctx context.Context, vm string, port int) (ports.SharedPort, error) {
	v, err := s.d.Reserve(ctx, vm, port)
	var down *ports.MacAwayError
	if s.inVM() && (errors.Is(err, ports.ErrEngineDown) || errors.As(err, &down)) {
		return ports.SharedPort{}, &ports.MacAwayError{Mac: s.o.MacName}
	}
	return v, err
}

func (s *Reserver) CheckRelease(ctx context.Context, t ports.Target) (mac string, err error) {
	if s.inVM() {
		return "", nil
	}
	mac, thisMac, err := s.o.VMMac(ctx, t.VM)
	if err != nil {
		return "", err
	}
	if !thisMac {
		return mac, nil
	}
	reservations, err := s.st.Reservations.List(ctx)
	if err != nil {
		return "", err
	}
	if _, i := ports.Find(reservations, t.VM, t.Port); i < 0 {
		return "", ports.NotFoundAt(t)
	}
	return "", nil
}

func (s *Reserver) Release(ctx context.Context, t ports.Target, mac string, yes bool) (ports.Reservation, error) {
	if mac != "" {
		return s.o.ReleaseOn(ctx, mac, t, yes)
	}
	l, err := s.d.Release(ctx, t.VM, t.Port)
	if s.inVM() && err != nil && errors.Is(err, ports.ErrEngineDown) {
		return ports.Reservation{}, &ports.MacAwayError{Mac: s.o.MacName}
	}
	if !errors.Is(err, ports.ErrEngineDown) {
		return l, err
	}
	l, err = s.st.Reservations.Get(ctx, t.VM, t.Port)
	if errors.Is(err, store.ErrNotFound) {
		return ports.Reservation{}, ports.NotFoundAt(t)
	}
	if err != nil {
		return ports.Reservation{}, err
	}
	return l, s.st.Reservations.Delete(ctx, t.VM, t.Port)
}

func SaveReservations(ctx context.Context, r contract.ReservationStore, reservations []ports.Reservation) error {
	// invariant: saving the whole set changes only what moved: a lease gone is
	// deleted, a lease added or changed is put, each write whole.
	current, err := r.List(ctx)
	if err != nil {
		return err
	}
	want := make(map[string]ports.Reservation, len(reservations))
	for _, l := range reservations {
		want[l.Key()] = l
	}
	for _, c := range current {
		if _, ok := want[c.Key()]; !ok {
			if err := r.Delete(ctx, c.VM, c.GuestPort); err != nil {
				return err
			}
		}
	}
	had := make(map[string]ports.Reservation, len(current))
	for _, c := range current {
		had[c.Key()] = c
	}
	for _, l := range reservations {
		if c, ok := had[l.Key()]; ok && reflect.DeepEqual(c, l) {
			continue
		}
		if err := r.Put(ctx, l); err != nil {
			return err
		}
	}
	return nil
}
