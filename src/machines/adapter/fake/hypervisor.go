package fake

import (
	"context"
	"errors"
	"maps"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/src/catalog"
	"github.com/jterrazz/studio-station/src/core/call"
	imagescontract "github.com/jterrazz/studio-station/src/images/contract"
	"github.com/jterrazz/studio-station/src/machines"
	"github.com/jterrazz/studio-station/src/machines/contract"
	"github.com/jterrazz/studio-station/src/world"
)

var (
	ErrNoDisk    = errors.New("no such VM disk")
	ErrNoImage   = errors.New("no such image")
	ErrNotBoot   = errors.New("the VM does not run")
	ErrTwoRuns   = errors.New("the VM already runs")
	ErrRunning   = errors.New("the VM is running")
	ErrHeldName  = errors.New("a VM disk already has this name")
	ErrScripted  = errors.New("the hypervisor refused to boot it, as the world scripted it")
	ErrNoAddress = errors.New("the VM got no address, as the world scripted it")
)

type Hypervisor struct {
	// invariant: the hypervisor of one Mac of the world: its disks and runs are
	// that Mac's VMs, so a run outlives the adapter, as a VM outlives its engine,
	// and an event of the world ends it.
	w    *world.World
	mac  string
	name string
	// value: how the catalog installs the hypervisor on a Mac, which the
	// world's installers hold once it is installed.
	install catalog.Install
}

func HypervisorOf(w *world.World, mac, name string, install catalog.Install) *Hypervisor {
	return &Hypervisor{w: w, mac: mac, name: name, install: install}
}

func (h *Hypervisor) Name() string {
	if n := h.thisMac().Hypervisor; n != "" && n != "none" {
		return n
	}
	return h.name
}

func (h *Hypervisor) Installed() bool {
	// invariant: a Mac of no hypervisor has one once the catalogue's item
	// for it is installed in the world, as the real one lands on the path.
	m := h.thisMac()
	if m.Hypervisor != "none" {
		return true
	}
	i := h.install
	if i == nil || m.Apps[i.Method()] == nil {
		return false
	}
	_, held := m.Apps[i.Method()].Installed[i.Key()]
	return held
}

func (h *Hypervisor) Version(context.Context) (string, error) { return "1.0.0", nil }

func (h *Hypervisor) thisMac() *world.Mac { return h.w.Mac(h.mac) }

func (h *Hypervisor) disk(vm string) (*world.VM, bool) {
	v, ok := h.thisMac().VMs[vm]
	return v, ok && v.Disk
}

func (h *Hypervisor) Disks(context.Context) ([]contract.Disk, error) {
	h.w.Lock()
	defer h.w.Unlock()
	out := []contract.Disk{}
	for _, vm := range slices.Sorted(maps.Keys(h.thisMac().VMs)) {
		v := h.thisMac().VMs[vm]
		if !v.Disk {
			continue
		}
		state := contract.DiskStopped
		if v.Running() {
			state = contract.DiskRunning
		}
		out = append(out, contract.Disk{VM: vm, Bytes: v.Size.Disk, State: state})
	}
	return out, nil
}

func (h *Hypervisor) Fetch(_ context.Context, img imagescontract.ImageLocation, p call.Progress) error {
	if p != nil {
		p.Step("fetching", img.Ref)
		defer p.Done("fetching", nil)
	}
	h.w.Lock()
	defer h.w.Unlock()
	if m := h.thisMac(); !slices.Contains(m.Images, img.Ref) {
		m.Images = append(m.Images, img.Ref)
	}
	return nil
}

func (h *Hypervisor) Clone(ctx context.Context, from imagescontract.ImageLocation, vm string) error {
	h.w.Lock()
	defer h.w.Unlock()
	if err := h.held(ctx, "Hypervisor.Clone"); err != nil {
		return err
	}
	m := h.thisMac()
	_, fromDisk := h.disk(from.Ref)
	if !fromDisk && !slices.Contains(m.Images, from.Ref) {
		return ErrNoImage
	}
	if _, ok := h.disk(vm); ok {
		return ErrHeldName
	}
	v := m.VMs[vm]
	if v == nil {
		v = &world.VM{}
		m.VMs[vm] = v
	}
	v.Disk = true
	if fresh := m.Fresh; !fromDisk && fresh != nil {
		v.Size = fresh.Size
		if v.Guest == nil && fresh.Guest != nil {
			guest := *fresh.Guest
			v.Guest = &guest
		}
	}
	if v.Guest == nil {
		v.Guest = &world.Guest{Fresh: true}
	}
	return nil
}

func (h *Hypervisor) Resize(_ context.Context, vm string, s machines.Size) error {
	h.w.Lock()
	defer h.w.Unlock()
	v, ok := h.disk(vm)
	if !ok {
		return ErrNoDisk
	}
	v.Size = s
	return nil
}

func (h *Hypervisor) Rename(_ context.Context, from, to string) error {
	h.w.Lock()
	defer h.w.Unlock()
	v, ok := h.disk(from)
	if !ok {
		return ErrNoDisk
	}
	if _, held := h.disk(to); held {
		return ErrHeldName
	}
	m := h.thisMac()
	delete(m.VMs, from)
	if kept := m.VMs[to]; kept != nil {
		v.Kept = kept.Kept
	}
	m.VMs[to] = v
	return nil
}

func (h *Hypervisor) Boot(ctx context.Context, vm string, opts contract.BootOptions) (contract.Run, error) {
	h.w.Lock()
	defer h.w.Unlock()
	if err := h.held(ctx, "Hypervisor.Boot"); err != nil {
		return nil, err
	}
	v, ok := h.disk(vm)
	if !ok {
		return nil, ErrNoDisk
	}
	if v.Running() {
		return nil, ErrTwoRuns
	}
	if h.w.Fails("Hypervisor.Boot") {
		return nil, ErrScripted
	}
	v.PID, v.Ended, v.UpAt = h.nextPID(), nil, h.w.Now
	v.Booted = &world.Boot{Bridged: opts.Network.Bridged, Folders: folders(opts.Mounts)}
	return &run{h: h, vm: vm, pid: v.PID}, nil
}

// value: the folders a run shares, each as tart's --dir says it:
// `<folder>[:ro],tag=<tag>`.
func folders(mounts []machines.Mount) []string {
	out := make([]string, 0, len(mounts))
	for i, m := range mounts {
		share := m.Computer + ":"
		if m.ReadOnly {
			share += "ro,"
		}
		out = append(out, share+"tag="+machines.MountTag(i))
	}
	return out
}

// invariant: a held call, called under the world's lock, lets the world go
// and waits until it is interrupted, as a long clone or boot caught by ctrl-c.
func (h *Hypervisor) held(ctx context.Context, method string) error {
	if !h.w.Holds(method) {
		return nil
	}
	h.w.Pause()
	h.w.Unlock()
	<-ctx.Done()
	h.w.Lock()
	return ctx.Err()
}

// value: pids count up over the whole world, so no two runs share one.
func (h *Hypervisor) nextPID() int {
	top := 1000
	for _, m := range h.w.Macs {
		for _, v := range m.VMs {
			top = max(top, v.PID)
		}
	}
	return top + 1
}

func (h *Hypervisor) Adopt(_ context.Context, vm string) (contract.Run, bool) {
	h.w.Lock()
	defer h.w.Unlock()
	if v, ok := h.disk(vm); ok && v.Running() {
		return &run{h: h, vm: vm, pid: v.PID}, true
	}
	return nil, false
}

func End(r contract.Run, end machines.RunEnd) {
	// value: ends r's run the way its guest would, wherever the run was adopted.
	if r, ok := r.(*run); ok {
		r.h.end(r.vm, r.pid, end)
	}
}

func (h *Hypervisor) end(vm string, pid int, end machines.RunEnd) {
	h.w.Lock()
	defer h.w.Unlock()
	if v, ok := h.disk(vm); ok && v.PID == pid {
		v.Down(h.w.Now, &end)
	}
}

func (h *Hypervisor) Shutdown(_ context.Context, vm string, _ time.Duration) error {
	h.w.Lock()
	defer h.w.Unlock()
	v, ok := h.disk(vm)
	if !ok {
		return ErrNoDisk
	}
	if v.Running() {
		v.Down(h.w.Now, &machines.RunEnd{Reason: machines.GuestPoweroff})
	}
	return nil
}

func (h *Hypervisor) Delete(_ context.Context, vm string) error {
	h.w.Lock()
	defer h.w.Unlock()
	v, ok := h.disk(vm)
	if !ok {
		return ErrNoDisk
	}
	if v.Running() {
		return ErrRunning
	}
	v.Disk, v.PID, v.Ended = false, 0, nil
	if v.Kept == nil {
		delete(h.thisMac().VMs, vm)
	}
	return nil
}

func (h *Hypervisor) Discard(_ context.Context, img imagescontract.ImageLocation) error {
	h.w.Lock()
	defer h.w.Unlock()
	m := h.thisMac()
	m.Images = slices.DeleteFunc(m.Images, func(ref string) bool { return ref == img.Ref })
	if v, ok := m.VMs[img.Ref]; ok && v.Kept == nil {
		delete(m.VMs, img.Ref)
	}
	return nil
}

func (h *Hypervisor) Address(_ context.Context, vm string, _ time.Duration) (netip.Addr, error) {
	h.w.Lock()
	defer h.w.Unlock()
	v, ok := h.disk(vm)
	if !ok || !v.Running() {
		return netip.Addr{}, ErrNotBoot
	}
	if h.w.Fails("Hypervisor.Address") {
		return netip.Addr{}, ErrNoAddress
	}
	if v.Address != "" {
		return netip.ParseAddr(v.Address)
	}
	return netip.MustParseAddr("192.168.64.5"), nil
}

func (h *Hypervisor) Console(_ context.Context, vm string, n int) ([]string, error) {
	h.w.Lock()
	defer h.w.Unlock()
	if v, ok := h.disk(vm); ok {
		return console(v, n), nil
	}
	return nil, nil
}

// value: the lines the world scripted on the VM's console, as the
// hypervisor's adapter keeps them.
func console(v *world.VM, n int) []string {
	var kept []string
	for _, line := range v.Console {
		if machines.Meaningful(line) {
			kept = append(kept, line)
		}
	}
	if n > 0 && len(kept) > n {
		kept = kept[len(kept)-n:]
	}
	return kept
}

// invariant: a run is over once its VM no longer runs under its pid; how it
// ended is read from the world, once, then kept.
type run struct {
	h   *Hypervisor
	vm  string
	pid int

	once sync.Mutex
	end  *machines.RunEnd
}

func (r *run) VM() string  { return r.vm }
func (r *run) PID() int    { return r.pid }
func (r *run) Holder() int { return r.pid }

func (r *run) Ended() <-chan machines.RunEnd {
	r.once.Lock()
	defer r.once.Unlock()
	if r.end == nil {
		r.h.w.Lock()
		v, ok := r.h.thisMac().VMs[r.vm]
		switch {
		case ok && v.PID == r.pid:
		case ok && v.Ended != nil && r.h.w.Fails("Hypervisor.Ended"):
			r.end, v.Ended = &machines.RunEnd{Reason: machines.Unknown, ExitCode: -1}, nil
		case ok && v.Ended != nil:
			end := *v.Ended
			end.LastLines = machines.LastLines(console(v, machines.ConsoleWindow))
			r.end, v.Ended = &end, nil
		default:
			r.end = &machines.RunEnd{Reason: machines.Unknown, ExitCode: -1}
		}
		r.h.w.Unlock()
	}
	ch := make(chan machines.RunEnd, 1)
	if r.end != nil {
		ch <- *r.end
		close(ch)
	}
	return ch
}
