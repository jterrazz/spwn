package process

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/engine"
)

type BuildHypervisor interface {
	// value: what the engine needs of the hypervisor to boot a build VM: the
	// clone's boot and the address it leases.
	BootBuild(ctx context.Context, vm string) error
	Address(ctx context.Context, vm string, wait time.Duration) (netip.Addr, error)
}

type BuildLogin struct {
	// value: who a build speaks to its VM as: the base's admin by its
	// password until the recipe installs the image key, then the VM's user by
	// that key.
	User, Key, Password string
}

type BuildGuests interface {
	// invariant: a build VM is a fresh clone, so its guest is dialed with no
	// host key kept.
	Dial(ctx context.Context, vm string, at netip.AddrPort, as BuildLogin) (BuildGuest, error)
}

type BuildGuest interface {
	Exec(ctx context.Context, line, stdin string) (engine.GuestRan, error)
	Put(ctx context.Context, path string, data []byte, mode uint32) error
	Close() error
}

const (
	buildAnswer = 60 * time.Second
	buildPoll   = time.Second
)

type BuildVMs struct {
	// value: the engine's build VMs: it boots each clone and holds the guest a
	// build speaks to, one session per login, until the build lets it go.
	hv     BuildHypervisor
	guests BuildGuests
	sleep  func(ctx context.Context, d time.Duration) error
	port   uint16
}

func NewBuildVMs(hv BuildHypervisor, guests BuildGuests, sleep func(ctx context.Context, d time.Duration) error, port uint16) *BuildVMs {
	return &BuildVMs{hv: hv, guests: guests, sleep: sleep, port: port}
}

func (b *BuildVMs) Boot(ctx context.Context, vm string) error { return b.hv.BootBuild(ctx, vm) }

func (b *BuildVMs) Open(ctx context.Context, vm string, wait time.Duration) (*BuildVM, error) {
	// value: the booted VM's guest, reached at the address it leased within wait.
	addr, err := b.hv.Address(ctx, vm, wait)
	if err != nil {
		return nil, err
	}
	return &BuildVM{b: b, vm: vm, at: netip.AddrPortFrom(addr, b.port), held: map[BuildLogin]BuildGuest{}}, nil
}

type BuildVM struct {
	b    *BuildVMs
	vm   string
	at   netip.AddrPort
	held map[BuildLogin]BuildGuest
}

func (v *BuildVM) guest(ctx context.Context, as BuildLogin) (BuildGuest, error) {
	if g, ok := v.held[as]; ok {
		return g, nil
	}
	for range int(buildAnswer / buildPoll) {
		g, err := v.b.guests.Dial(ctx, v.vm, v.at, as)
		if err == nil {
			v.held[as] = g
			return g, nil
		}
		if err := v.b.sleep(ctx, buildPoll); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("the build VM did not answer in " + buildAnswer.String())
}

func (v *BuildVM) Put(ctx context.Context, as BuildLogin, path string, data []byte, mode uint32) error {
	g, err := v.guest(ctx, as)
	if err != nil {
		return err
	}
	return g.Put(ctx, path, data, mode)
}

func (v *BuildVM) Run(ctx context.Context, as BuildLogin, line, stdin string) (engine.GuestRan, error) {
	g, err := v.guest(ctx, as)
	if err != nil {
		return engine.GuestRan{}, err
	}
	return g.Exec(ctx, line, stdin)
}

func (v *BuildVM) Close() error {
	errs := make([]error, 0, len(v.held))
	for as, g := range v.held {
		errs = append(errs, g.Close())
		delete(v.held, as)
	}
	return errors.Join(errs...)
}
