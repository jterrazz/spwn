package fake

import (
	"context"
	"errors"
	"maps"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/core/call"
	imagescontract "github.com/jterrazz/studio-station/apps/engine/images/contract"
	"github.com/jterrazz/studio-station/apps/engine/machines"
	"github.com/jterrazz/studio-station/apps/engine/machines/contract"
	"github.com/jterrazz/studio-station/apps/engine/world"
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
	ErrNoListing = errors.New("the hypervisor could not list its disks, as the world scripted it")
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
	h.w.Lock()
	defer h.w.Unlock()
	if n := h.thisMac().Hypervisor; n != "" && n != "none" {
		return n
	}
	return h.name
}

func (h *Hypervisor) Package() string {
	if h.Name() == h.name {
		return h.name
	}
	return ""
}

func (h *Hypervisor) Installed() bool {
	// invariant: a Mac of no hypervisor has one once the catalogue's package
	// for it is installed in the world, as the real one lands on the path.
	h.w.Lock()
	defer h.w.Unlock()
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
	if h.w.Fails("Hypervisor.Disks") {
		return nil, ErrNoListing
	}
	out := []contract.Disk{}
	for _, vm := range slices.Sorted(maps.Keys(h.thisMac().VMs)) {
		v := h.thisMac().VMs[vm]
		if !v.Disk {
			continue
		}
		state := contract.DiskStopped
		switch {
		case v.Running():
			state = contract.DiskRunning
		case v.Suspended:
			state = contract.DiskSuspended
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
	if err := h.held(ctx, "Hypervisor.Clone", vm); err != nil {
		return err
	}
	m := h.thisMac()
	source, fromDisk := h.disk(from.Ref)
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
	v.Disk, v.Made, v.Born = true, true, h.nextBorn()
	if m.Cloned == nil {
		m.Cloned = map[string]int{}
	}
	m.Cloned[vm] = v.Born
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
	h.w.Now = h.w.Now.Add(cloneTook)
	switch {
	case fromDisk && source.Guest != nil && source.Guest.Login != nil:
		login := *source.Guest.Login
		v.Guest.Login = &login
	case !fromDisk && h.w.UpstreamLogin != nil:
		login := *h.w.UpstreamLogin
		v.Guest.Login = &login
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
	born, ours := m.Cloned[from]
	delete(m.Cloned, from)
	delete(m.Cloned, to)
	if ours {
		m.Cloned[to] = born
	}
	return nil
}

func (h *Hypervisor) Cloned(context.Context) ([]string, error) {
	// invariant: the hypervisor adapter's rule: an entry stands while the disk
	// under its name is the one it cloned (its Born); any other is dropped.
	h.w.Lock()
	defer h.w.Unlock()
	m := h.thisMac()
	out := []string{}
	for _, name := range slices.Sorted(maps.Keys(m.Cloned)) {
		if v, ok := h.disk(name); ok && v.Born != 0 && v.Born == m.Cloned[name] {
			out = append(out, name)
			continue
		}
		delete(m.Cloned, name)
	}
	return out, nil
}

func (h *Hypervisor) Forget(_ context.Context, vm string) error {
	h.w.Lock()
	defer h.w.Unlock()
	delete(h.thisMac().Cloned, vm)
	return nil
}

// value: births count up over the whole world, as pids do, so no two
// disks ever share one.
func (h *Hypervisor) nextBorn() int {
	top := 0
	for _, m := range h.w.Macs {
		for _, v := range m.VMs {
			top = max(top, v.Born)
		}
		for _, born := range m.Cloned {
			top = max(top, born)
		}
	}
	return top + 1
}

func (h *Hypervisor) Boot(ctx context.Context, vm string, opts contract.BootOptions) (contract.Run, error) {
	h.w.Lock()
	defer h.w.Unlock()
	if err := h.held(ctx, "Hypervisor.Boot", vm); err != nil {
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
		v.Console = append(v.Console, bootRefused()...)
		return nil, ErrScripted
	}
	v.PID, v.Ended, v.UpAt = h.nextPID(), nil, h.w.Now
	v.Booted = &world.Boot{Bridged: opts.Network.Bridged, Folders: folders(opts.Mounts)}
	return &run{h: h, vm: vm, pid: v.PID}, nil
}

// value: what a clone takes on the world's clock, so a create over the world
// has a step of a duration; a boot takes none, since every uptime a story
// reads begins there.
const cloneTook = 2 * time.Second

// value: the lines the console writes when the world scripts a boot to fail,
// the same each time, so a create that fails there has last lines to keep.
func bootRefused() []string {
	return []string{
		"vz: starting the virtual machine",
		"vz: configuration validated",
		"vz: the hypervisor refused to start the virtual machine",
	}
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
// and waits until it is interrupted, as a long clone or boot caught by ctrl-c;
// a hold the world declares for its VM also ends at a world unhold, and the
// call goes on as if never held.
func (h *Hypervisor) held(ctx context.Context, method, vm string) error {
	kept := h.w.Keeps(method, h.mac, vm)
	if !kept && !h.w.Holds(method) {
		return nil
	}
	reached := world.Fault{Call: method, Hold: true, Mac: h.mac, VM: vm}
	if kept {
		h.w.Reach(reached)
	}
	h.w.Pause()
	unheld := h.w.Unheld(ctx, method, h.mac, vm)
	h.w.Unlock()
	select {
	case <-ctx.Done():
		h.w.Lock()
		h.w.Leave(reached)
		return ctx.Err()
	case <-unheld:
		h.w.Lock()
		h.w.Unhold(h.mac, vm)
		h.w.Leave(reached)
		return nil
	}
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
	v.Disk, v.PID, v.Ended, v.Born = false, 0, nil, 0
	// invariant: what ran inside a VM lived on its disk, so its slice goes
	// with it: a VM made again under its name starts from its image.
	delete(h.w.Macs, vm)
	m := h.thisMac()
	delete(m.Cloned, vm)
	if v.Kept == nil {
		delete(m.VMs, vm)
	}
	return nil
}

func (h *Hypervisor) Discard(_ context.Context, img imagescontract.ImageLocation) error {
	h.w.Lock()
	defer h.w.Unlock()
	m := h.thisMac()
	m.Images = slices.DeleteFunc(m.Images, func(ref string) bool { return ref == img.Ref })
	delete(m.Cloned, img.Ref)
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

// value: the lines the world scripted on the VM's console, then those of
// its hypervisor's own output, each source's last n, as the hypervisor's
// adapter reads its two files.
func console(v *world.VM, n int) []string {
	return append(lastMeaningful(v.Console, n), lastMeaningful(v.Hypervisor, n)...)
}

func lastMeaningful(lines []string, n int) []string {
	var kept []string
	for _, line := range lines {
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
